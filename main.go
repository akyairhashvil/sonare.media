package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"sonare.media/internal/store"
	"sonare.media/internal/tui"
)

func main() {
	mode := flag.String("mode", "serve-test", "Mode: 'serve-test' (Test :8080), 'serve-prod' (Prod :443/:80), 'serve-cfd' (Cloudflare Tunnel), or 'view' (TUI)")
	port := flag.String("port", "8080", "Port to serve on (Test/CFD modes)")
	dbPath := flag.String("db", "sonare.db", "Path to SQLite database")
	flag.Parse()

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

	if *mode == "view" {
		if err := tui.Start(); err != nil {
			log.Fatalf("TUI Error: %v", err)
		}
		return
	}

	// Server Mode
	mux := http.NewServeMux()

	// Static File Server
	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", securityHeadersMiddleware(analyticsMiddleware(fileServer)))

	// API Endpoints
	mux.HandleFunc("/api/lead", analyticsMiddlewareFunc(handleLead))

	certPath := "certs/server.crt"
	keyPath := "certs/server.key"

	// Validate Certs
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		log.Println("WARNING: No 'server.crt' found in certs/. Server may fail to start.")
	}

	var servers []*http.Server

	if *mode == "serve-prod" {
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
			Handler: mux,
		}
		servers = append(servers, httpsServer)

		go func() {
			log.Println("LISTENING: :443 (HTTPS)")
			if err := httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS Server Failed: %v", err)
			}
		}()

	} else if *mode == "serve-cfd" {
		// --- CLOUDFLARE TUNNEL MODE ---
		addr := ":" + *port

		cfdServer := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		servers = append(servers, cfdServer)

		go func() {
			log.Printf("SERVER START: Cloudflare Tunnel Mode on http://localhost%s (HTTP)\n", addr)
			if err := cfdServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP Server Failed: %v\n", err)
			}
		}()

	} else {
		// --- TEST MODE ---
		// Default to test mode if mode is serve-test or anything else (e.g. legacy 'serve')
		addr := ":" + *port
		
		testServer := &http.Server{
			Addr:    addr,
			Handler: mux,
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
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}


// Middleware to track analytics
func analyticsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackRequest(r)
		next.ServeHTTP(w, r)
	})
}

func analyticsMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trackRequest(r)
		next(w, r)
	}
}

func trackRequest(r *http.Request) {
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

