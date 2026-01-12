package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type Lead struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Business   string    `json:"business"`
	Playback   string    `json:"system"` // Mapped from "system" in JSON payload
	Email      string    `json:"email"`
	Message    string    `json:"message"`
	Palette    string    `json:"palette"`
	HoursEst   int       `json:"hours_est,string"` // Handle string/int conversion from JSON
	StoreCount int       `json:"store_count,string"`
	CreatedAt  time.Time `json:"created_at"`
}

type Analytics struct {
	ID        int       `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

func InitDB(filepath string) error {
	var err error
	DB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	// Enable WAL mode for better concurrency
	if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	return createTables()
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS leads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			business TEXT,
			playback TEXT,
			email TEXT,
			message TEXT,
			palette TEXT,
			hours_est INTEGER,
			store_count INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS analytics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT,
			user_agent TEXT,
			path TEXT,
			method TEXT,
			country TEXT,
			city TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func SaveLead(l Lead) error {
	stmt, err := DB.Prepare("INSERT INTO leads(name, business, playback, email, message, palette, hours_est, store_count) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(l.Name, l.Business, l.Playback, l.Email, l.Message, l.Palette, l.HoursEst, l.StoreCount)
	return err
}

func GetLeads() ([]Lead, error) {
	rows, err := DB.Query("SELECT id, name, business, playback, email, message, palette, hours_est, store_count, created_at FROM leads ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []Lead
	for rows.Next() {
		var l Lead
		if err := rows.Scan(&l.ID, &l.Name, &l.Business, &l.Playback, &l.Email, &l.Message, &l.Palette, &l.HoursEst, &l.StoreCount, &l.CreatedAt); err != nil {
			return nil, err
		}
		leads = append(leads, l)
	}
	return leads, nil
}

func SaveAnalytics(a Analytics) error {
	// Enrich with GeoIP
	country, city := GetGeoLocation(a.IP)
	a.Country = country
	a.City = city

	stmt, err := DB.Prepare("INSERT INTO analytics(ip, user_agent, path, method, country, city) VALUES(?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(a.IP, a.UserAgent, a.Path, a.Method, a.Country, a.City)
	return err
}

func GetAnalytics() ([]Analytics, error) {
	rows, err := DB.Query("SELECT id, ip, user_agent, path, method, country, city, created_at FROM analytics ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []Analytics
	for rows.Next() {
		var a Analytics
		if err := rows.Scan(&a.ID, &a.IP, &a.UserAgent, &a.Path, &a.Method, &a.Country, &a.City, &a.CreatedAt); err != nil {
			return nil, err
		}
		analytics = append(analytics, a)
	}
	return analytics, nil
}

// Simple GeoIP struct for JSON unmarshalling
type GeoIPResponse struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Status  string `json:"status"`
}

func GetGeoLocation(ip string) (string, string) {
	// Skip localhost/private IPs for external lookup speedup
	if ip == "127.0.0.1" || ip == "::1" {
		return "Localhost", "Localhost"
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "Unknown", "Unknown"
	}
	defer resp.Body.Close()

	var geo GeoIPResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return "Unknown", "Unknown"
	}

	if geo.Status == "fail" {
		return "Unknown", "Unknown"
	}

	return geo.Country, geo.City
}
