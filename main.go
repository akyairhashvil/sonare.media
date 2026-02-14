package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sonare.media/internal/store"
	"sonare.media/internal/tui"
)

func main() {
	mode := flag.String("mode", "serve-test", "Mode: 'serve-test' (TLS test), 'serve-http' (HTTP only), 'serve-cfd' (Cloudflare Tunnel alias), 'serve-prod' (Prod :443/:80), or 'view' (TUI)")
	port := flag.String("port", "8080", "Port to serve on (test/http/cfd modes)")
	dbPath := flag.String("db", "sonare.db", "Path to SQLite database")
	flag.Parse()

	runMode, ok := normalizeMode(*mode)
	if !ok {
		log.Fatalf("Invalid mode %q. Valid modes: serve-test, serve-http, serve-cfd, serve-prod, view (aliases: test/http/cfd/prod/tui).", *mode)
	}

	// Ensure browsers receive a playable type for preview assets.
	if err := mime.AddExtensionType(".m4a", "audio/mp4"); err != nil {
		log.Printf("MIME REGISTER WARNING (.m4a): %v", err)
	}

	// Setup Logging
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
	} else {
		multi := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multi)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Initialize Database
	if err := store.InitDB(*dbPath); err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer func() {
		if store.DB != nil {
			store.DB.Close()
		}
	}()

	if runMode == "view" {
		if err := tui.Start(); err != nil {
			log.Fatalf("TUI Error: %v", err)
		}
		return
	}

	// Server Mode
	mux := http.NewServeMux()

	// Static File Server
	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", staticCacheHeadersMiddleware(fileServer))

	// API Endpoints
	mux.HandleFunc("/api/lead", handleLead)
	mux.HandleFunc("/api/preview-sources", handlePreviewSources)
	mux.HandleFunc("/healthz", handleHealth)

	// Apply observability and security controls to all routes.
	handler := securityHeadersMiddleware(analyticsMiddleware(mux))

	certPath := "certs/server.crt"
	keyPath := "certs/server.key"

	if runMode == "serve-test" || runMode == "serve-prod" {
		if err := ensureTLSFiles(certPath, keyPath); err != nil {
			log.Fatalf("TLS setup error: %v (use -mode serve-http for HTTP-only)", err)
		}
	}

	var servers []*http.Server

	if runMode == "serve-prod" {
		// Check for root privileges
		if os.Geteuid() != 0 {
			log.Println("PRODUCTION MODE REQUIRES ROOT PRIVILEGES (Binding ports 80/443).")
			log.Println("Attempting to relaunch with sudo...")

			cmd := exec.Command("sudo", os.Args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			if err := cmd.Run(); err != nil {
				log.Fatalf("Failed to run with sudo: %v", err)
			}
			return // Exit parent process
		}

		// --- PRODUCTION MODE ---
		log.Println("SERVER START: Production Mode Enabled")

		// 1. HTTP Redirect Server (:80 -> :443)
		redirectMux := http.NewServeMux()
		redirectMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.String()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})

		httpServer := &http.Server{
			Addr:    ":80",
			Handler: redirectMux,
		}
		servers = append(servers, httpServer)

		go func() {
			log.Println("LISTENING: :80 (HTTP Redirect)")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP Server Failed: %v", err)
			}
		}()

		// 2. HTTPS Main Server (:443)
		httpsServer := &http.Server{
			Addr:    ":443",
			Handler: handler,
		}
		servers = append(servers, httpsServer)

		go func() {
			log.Println("LISTENING: :443 (HTTPS)")
			if err := httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS Server Failed: %v", err)
			}
		}()

	} else if runMode == "serve-http" || runMode == "serve-cfd" {
		// --- HTTP-ONLY MODE (Cloudflare Tunnel origin-compatible) ---
		addr := ":" + *port

		httpServer := &http.Server{
			Addr:    addr,
			Handler: handler,
		}
		servers = append(servers, httpServer)

		go func() {
			if runMode == "serve-cfd" {
				log.Println("NOTICE: '-mode serve-cfd' is retained for compatibility; prefer '-mode serve-http'.")
				log.Printf("SERVER START: Cloudflare Tunnel Mode on http://localhost%s (HTTP)\n", addr)
			} else {
				log.Printf("SERVER START: HTTP-Only Mode on http://localhost%s (HTTP)\n", addr)
			}

			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP Server Failed: %v\n", err)
			}
		}()

	} else {
		// --- TEST MODE ---
		// Default to test mode if mode is serve-test or anything else (e.g. legacy 'serve')
		addr := ":" + *port

		testServer := &http.Server{
			Addr:    addr,
			Handler: handler,
		}
		servers = append(servers, testServer)

		go func() {
			log.Printf("SERVER START: Test Mode on https://localhost%s (TLS self-signed)\n", addr)
			if err := testServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("TLS Failed: %v\n", err)
			}
		}()
	}

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("SHUTDOWN SIGNAL RECEIVED: Stopping servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}

	log.Println("SERVER STOPPED: Clean exit.")
}

// Middleware: Security Headers
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"font-src 'self' data:",
			"media-src 'self'",
			"connect-src 'self'",
			"object-src 'none'",
			"base-uri 'self'",
			"frame-ancestors 'none'",
			"form-action 'self'",
		}, "; ")

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// Static caching profile for web assets.
func staticCacheHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/" || path == "/index.html":
			w.Header().Set("Cache-Control", "no-cache")
		case strings.HasPrefix(path, "/music/"):
			w.Header().Set("Cache-Control", "public, max-age=604800")
		case strings.HasPrefix(path, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=86400")
		case hasCacheableStaticExt(path):
			w.Header().Set("Cache-Control", "public, max-age=86400")
		default:
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func hasCacheableStaticExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css", ".js", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".m4a":
		return true
	default:
		return false
	}
}

// Middleware to track analytics
func analyticsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackRequest(r)
		next.ServeHTTP(w, r)
	})
}

func trackRequest(r *http.Request) {
	// Keep health checks cheap and noise-free for monitors.
	if r.URL.Path == "/healthz" {
		return
	}

	// Filter noise if needed, but user requested complete logs
	// if strings.Contains(r.URL.Path, "favicon.ico") { return }

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}

	// Log to file/console
	log.Printf("REQUEST: [%s] %s %s | UA: %s", r.Method, r.URL.Path, ip, r.UserAgent())

	// Save to DB in background
	go func() {
		err := store.SaveAnalytics(store.Analytics{
			IP:        ip,
			UserAgent: r.UserAgent(),
			Path:      r.URL.Path,
			Method:    r.Method,
		})
		if err != nil {
			log.Printf("DB ERROR (Analytics): %v", err)
		}
	}()
}

func handleLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var l store.Lead
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		log.Printf("BAD REQUEST (Lead): %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Log the Form Entry Details
	log.Printf("LEAD RECEIVED: Name='%s' Business='%s' Email='%s' System='%s' Palette='%s' Scale='%dh/%d stores'",
		l.Name, l.Business, l.Email, l.Playback, l.Palette, l.HoursEst, l.StoreCount)

	if err := store.SaveLead(l); err != nil {
		log.Printf("DB ERROR (Lead): %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func handlePreviewSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	palette := normalizePalette(r.URL.Query().Get("palette"))
	if palette == "" {
		http.Error(w, "Missing palette query parameter", http.StatusBadRequest)
		return
	}

	sources, err := previewSourcesForPalette(filepath.Join("web", "music"), palette)
	if err != nil {
		log.Printf("PREVIEW SOURCE ERROR: %v", err)
		http.Error(w, "Failed to load preview sources", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"palette": palette,
		"sources": sources,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := http.StatusOK
	overall := "ok"
	dbState := "ok"

	if store.DB == nil {
		status = http.StatusServiceUnavailable
		overall = "degraded"
		dbState = "uninitialized"
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := store.DB.PingContext(ctx); err != nil {
			status = http.StatusServiceUnavailable
			overall = "degraded"
			dbState = "down"
		}
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if r.Method == http.MethodHead {
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": overall,
		"db":     dbState,
	})
}

func previewSourcesForPalette(musicDir, palette string) (map[string]string, error) {
	sources := map[string]string{
		"open":    "",
		"peak":    "",
		"offpeak": "",
		"close":   "",
		"beacon":  "",
	}

	entries, err := os.ReadDir(musicDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entryPalette, trackKey, ok := parsePreviewTrackFilename(entry.Name())
		if !ok || entryPalette != palette {
			continue
		}

		sources[trackKey] = "/music/" + url.PathEscape(entry.Name())
	}

	return sources, nil
}

func parsePreviewTrackFilename(filename string) (palette string, trackKey string, ok bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".m4a" {
		return "", "", false
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	palette = normalizePalette(parts[0])
	if palette == "" {
		return "", "", false
	}

	suffix := strings.ToUpper(parts[1])
	switch {
	case suffix == "SONARE":
		return palette, "beacon", true
	case strings.HasPrefix(suffix, "OPEN-"):
		return palette, "open", true
	case strings.HasPrefix(suffix, "PEAK-"):
		return palette, "peak", true
	case strings.HasPrefix(suffix, "OFFPEAK-"):
		return palette, "offpeak", true
	case strings.HasPrefix(suffix, "CLOSE-"):
		return palette, "close", true
	default:
		return "", "", false
	}
}

func normalizePalette(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeMode(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "serve-test", "test", "serve":
		return "serve-test", true
	case "serve-http", "http":
		return "serve-http", true
	case "serve-cfd", "cfd", "cloudflared":
		return "serve-cfd", true
	case "serve-prod", "prod":
		return "serve-prod", true
	case "view", "tui":
		return "view", true
	default:
		return "", false
	}
}

func ensureTLSFiles(certPath, keyPath string) error {
	if _, err := os.Stat(certPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing cert file %q", certPath)
		}
		return fmt.Errorf("failed to stat cert file %q: %w", certPath, err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing key file %q", keyPath)
		}
		return fmt.Errorf("failed to stat key file %q: %w", keyPath, err)
	}
	return nil
}
