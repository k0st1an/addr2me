package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*
var templateFiles embed.FS

// IPInfo holds geo/ASN data from ipinfo.io.
type IPInfo struct {
	ASN           string `json:"asn"`
	ASName        string `json:"as_name"`
	CountryCode   string `json:"country_code"`
	Country       string `json:"country"`
	ContinentCode string `json:"continent_code"`
	Continent     string `json:"continent"`
}

const ipinfoTimeout = 2 * time.Second

var (
	ipinfoClient  = &http.Client{Timeout: ipinfoTimeout}
	ipinfoBaseURL = "https://api.ipinfo.io/lite/"
)

// isTimeoutError reports whether err is a context or network timeout.
// Timeouts are expected when ipinfo.io is unreachable and are not logged.
func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// lookupIPInfo queries ipinfo.io Lite API for geo/ASN data.
// Returns nil, nil when token is not set.
func lookupIPInfo(ctx context.Context, ip, token string) (*IPInfo, error) {
	if token == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipinfoBaseURL+ip, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ipinfoClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipinfo returned %d", resp.StatusCode)
	}
	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

type PageData struct {
	IP            string
	Time          string
	Host          string
	Country       string
	CountryCode   string
	Flag          string
	Continent     string
	ContinentCode string
	ASName        string
	ASN           string
	HasGeo        bool
}

// countryFlag converts a two-letter ISO country code to its emoji flag.
// e.g. "US" → "🇺🇸", "DE" → "🇩🇪"
func countryFlag(code string) string {
	if len(code) != 2 {
		return ""
	}
	const base = 0x1F1E6 - 'A'
	return string(rune(base+rune(code[0]))) + string(rune(base+rune(code[1])))
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check X-Real-IP header (nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func wantsBrowser(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

type JSONResponse struct {
	IP            string `json:"ip"`
	Time          string `json:"time_utc"`
	Country       string `json:"country,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	Continent     string `json:"continent,omitempty"`
	ContinentCode string `json:"continent_code,omitempty"`
	ASN           string `json:"asn,omitempty"`
	ASName        string `json:"as_name,omitempty"`
}

func jsonHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		resp := JSONResponse{
			IP:   getClientIP(r),
			Time: time.Now().UTC().Format(time.RFC3339),
		}
		ctx, cancel := context.WithTimeout(r.Context(), ipinfoTimeout)
		defer cancel()
		if info, err := lookupIPInfo(ctx, resp.IP, token); err != nil {
			if !isTimeoutError(err) {
				log.Printf("ipinfo lookup error: %v", err)
			}
		} else if info != nil {
			resp.Country = info.Country
			resp.CountryCode = info.CountryCode
			resp.Continent = info.Continent
			resp.ContinentCode = info.ContinentCode
			resp.ASN = info.ASN
			resp.ASName = info.ASName
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func indexHandler(tmpl *template.Template, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		ip := getClientIP(r)
		now := time.Now().UTC().Format(time.RFC3339)

		data := PageData{
			IP:   ip,
			Time: now,
			Host: r.Host,
		}

		ctx, cancel := context.WithTimeout(r.Context(), ipinfoTimeout)
		defer cancel()
		if info, err := lookupIPInfo(ctx, ip, token); err != nil {
			if !isTimeoutError(err) {
				log.Printf("ipinfo lookup error: %v", err)
			}
		} else if info != nil {
			data.Country = info.Country
			data.CountryCode = info.CountryCode
			data.Flag = countryFlag(info.CountryCode)
			data.Continent = info.Continent
			data.ContinentCode = info.ContinentCode
			data.ASName = info.ASName
			data.ASN = info.ASN
			data.HasGeo = true
		}

		if !wantsBrowser(r) {
			w.Header().Set("Content-Type", "text/plain")
			parts := []string{ip, now}
			if data.HasGeo {
				parts = append(parts, data.Country, data.CountryCode,
					data.Continent, data.ContinentCode, data.ASN, data.ASName)
			}
			w.Write([]byte(strings.Join(parts, ",") + "\n"))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("template error: %v", err)
		}
	}
}

func main() {
	port := flag.String("port", "7007", "port to listen on")
	flag.Parse()

	tmpl, err := template.ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	token := os.Getenv("IPINFO_TOKEN")

	if token != "" {
		log.Printf("ipinfo.io geo enrichment enabled")
	}

	http.HandleFunc("/", indexHandler(tmpl, token))
	http.HandleFunc("/json", jsonHandler(token))

	addr := ":" + *port
	log.Printf("Server started at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
