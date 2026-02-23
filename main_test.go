package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// -- getClientIP --

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "1.2.3.4:5678"

	if got := getClientIP(r); got != "1.2.3.4" {
		t.Errorf("got %q, want %q", got, "1.2.3.4")
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"203.0.113.1", "203.0.113.1"},
		{"203.0.113.1, 10.0.0.1", "203.0.113.1"},   // multiple — take first
		{"  203.0.113.2 , 10.0.0.1", "203.0.113.2"}, // whitespace trimmed
		{"not-an-ip", "127.0.0.1"},                   // invalid — fall back to RemoteAddr
	}

	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Forwarded-For", tc.header)

		if got := getClientIP(r); got != tc.want {
			t.Errorf("X-Forwarded-For %q: got %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Real-IP", "198.51.100.5")

	if got := getClientIP(r); got != "198.51.100.5" {
		t.Errorf("got %q, want %q", got, "198.51.100.5")
	}
}

func TestGetClientIP_XForwardedForTakesPriorityOverXRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.10")
	r.Header.Set("X-Real-IP", "198.51.100.5")

	if got := getClientIP(r); got != "203.0.113.10" {
		t.Errorf("got %q, want %q", got, "203.0.113.10")
	}
}

func TestGetClientIP_IPv6(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[::1]:1234"

	if got := getClientIP(r); got != "::1" {
		t.Errorf("got %q, want %q", got, "::1")
	}
}

// -- wantsBrowser --

func TestWantsBrowser(t *testing.T) {
	tests := []struct {
		accept string
		want   bool
	}{
		{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", true},
		{"text/html", true},
		{"*/*", false},
		{"application/json", false},
		{"", false},
	}

	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.accept != "" {
			r.Header.Set("Accept", tc.accept)
		}

		if got := wantsBrowser(r); got != tc.want {
			t.Errorf("Accept %q: got %v, want %v", tc.accept, got, tc.want)
		}
	}
}

// -- countryFlag --

func TestCountryFlag(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"US", "🇺🇸"},
		{"DE", "🇩🇪"},
		{"RU", "🇷🇺"},
		{"", ""},
		{"U", ""},
		{"USA", ""},
	}
	for _, tc := range tests {
		if got := countryFlag(tc.code); got != tc.want {
			t.Errorf("countryFlag(%q): got %q, want %q", tc.code, got, tc.want)
		}
	}
}

// -- isTimeoutError --

func TestIsTimeoutError(t *testing.T) {
	if isTimeoutError(nil) {
		t.Error("nil should not be a timeout")
	}
	if isTimeoutError(fmt.Errorf("some other error")) {
		t.Error("generic error should not be a timeout")
	}
	if !isTimeoutError(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should be a timeout")
	}
	if !isTimeoutError(context.Canceled) {
		t.Error("context.Canceled should be a timeout")
	}
}

// -- lookupIPInfo --

func TestLookupIPInfo_EmptyToken(t *testing.T) {
	info, err := lookupIPInfo(context.Background(), "8.8.8.8", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil info, got %+v", info)
	}
}

func TestLookupIPInfo_WithToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IPInfo{
			ASN:           "AS15169",
			ASName:        "Google LLC",
			CountryCode:   "US",
			Country:       "United States",
			ContinentCode: "NA",
			Continent:     "North America",
		})
	}))
	defer srv.Close()

	old := ipinfoBaseURL
	ipinfoBaseURL = srv.URL + "/"
	defer func() { ipinfoBaseURL = old }()

	info, err := lookupIPInfo(context.Background(), "8.8.8.8", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Country != "United States" {
		t.Errorf("Country: got %q, want %q", info.Country, "United States")
	}
	if info.ASN != "AS15169" {
		t.Errorf("ASN: got %q, want %q", info.ASN, "AS15169")
	}
}

func TestLookupIPInfo_ErrorOnNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	old := ipinfoBaseURL
	ipinfoBaseURL = srv.URL + "/"
	defer func() { ipinfoBaseURL = old }()

	_, err := lookupIPInfo(context.Background(), "8.8.8.8", "bad-token")
	if err == nil {
		t.Error("expected error for non-200 response")
	}
}

// -- indexHandler --

func newTestTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("index.html").Parse(
		`<ip>{{.IP}}</ip><time>{{.Time}}</time>` +
			`{{if .HasGeo}}<geo>{{.Country}}({{.CountryCode}}),{{.Continent}}({{.ContinentCode}}),{{.ASName}}({{.ASN}})</geo>{{end}}`,
	)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}
	return tmpl
}

func TestIndexHandler_PlainText(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "1.2.3.4:9999"

	w := httptest.NewRecorder()
	indexHandler(newTestTemplate(t), "")(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Content-Type: got %q, want %q", ct, "text/plain")
	}

	body, _ := io.ReadAll(resp.Body)
	if got := strings.TrimSpace(string(body)); !strings.HasPrefix(got, "1.2.3.4,") {
		t.Errorf("body: got %q, want prefix %q", got, "1.2.3.4,")
	}
}

func TestIndexHandler_PlainText_WithGeo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IPInfo{
			ASN:           "AS15169",
			ASName:        "Google LLC",
			CountryCode:   "US",
			Country:       "United States",
			ContinentCode: "NA",
			Continent:     "North America",
		})
	}))
	defer srv.Close()

	old := ipinfoBaseURL
	ipinfoBaseURL = srv.URL + "/"
	defer func() { ipinfoBaseURL = old }()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "8.8.8.8:1234"

	w := httptest.NewRecorder()
	indexHandler(newTestTemplate(t), "test-token")(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Content-Type: got %q, want %q", ct, "text/plain")
	}

	body, _ := io.ReadAll(resp.Body)
	s := strings.TrimSpace(string(body))
	fields := strings.Split(s, ",")
	if len(fields) != 8 {
		t.Fatalf("plain text with geo: got %d fields, want 8: %q", len(fields), s)
	}
	if fields[0] != "8.8.8.8" {
		t.Errorf("field[0] ip: got %q, want %q", fields[0], "8.8.8.8")
	}
	if fields[2] != "United States" {
		t.Errorf("field[2] country: got %q, want %q", fields[2], "United States")
	}
	if fields[3] != "US" {
		t.Errorf("field[3] country_code: got %q, want %q", fields[3], "US")
	}
	if fields[4] != "North America" {
		t.Errorf("field[4] continent: got %q, want %q", fields[4], "North America")
	}
	if fields[5] != "NA" {
		t.Errorf("field[5] continent_code: got %q, want %q", fields[5], "NA")
	}
	if fields[6] != "AS15169" {
		t.Errorf("field[6] asn: got %q, want %q", fields[6], "AS15169")
	}
	if fields[7] != "Google LLC" {
		t.Errorf("field[7] as_name: got %q, want %q", fields[7], "Google LLC")
	}
}

func TestIndexHandler_HTML_NoGeo(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "1.2.3.4:9999"
	r.Header.Set("Accept", "text/html")

	w := httptest.NewRecorder()
	indexHandler(newTestTemplate(t), "")(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type: got %q, want text/html", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	if !strings.Contains(s, "1.2.3.4") {
		t.Errorf("body does not contain IP: %q", s)
	}
	if strings.Contains(s, "<geo>") {
		t.Error("body should not contain geo block when token is not set")
	}
}

func TestIndexHandler_HTML_WithGeo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IPInfo{
			ASN:           "AS15169",
			ASName:        "Google LLC",
			CountryCode:   "US",
			Country:       "United States",
			ContinentCode: "NA",
			Continent:     "North America",
		})
	}))
	defer srv.Close()

	old := ipinfoBaseURL
	ipinfoBaseURL = srv.URL + "/"
	defer func() { ipinfoBaseURL = old }()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "8.8.8.8:1234"
	r.Header.Set("Accept", "text/html")

	w := httptest.NewRecorder()
	indexHandler(newTestTemplate(t), "test-token")(w, r)

	body, _ := io.ReadAll(w.Result().Body)
	s := string(body)

	if !strings.Contains(s, "United States(US)") {
		t.Errorf("body missing country: %q", s)
	}
	if !strings.Contains(s, "North America(NA)") {
		t.Errorf("body missing continent: %q", s)
	}
	if !strings.Contains(s, "Google LLC(AS15169)") {
		t.Errorf("body missing ASN: %q", s)
	}
}

func TestIndexHandler_MethodNotAllowed(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		r := httptest.NewRequest(method, "/", nil)
		w := httptest.NewRecorder()
		indexHandler(newTestTemplate(t), "")(w, r)

		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("method %s: got %d, want %d", method, w.Result().StatusCode, http.StatusMethodNotAllowed)
		}
	}
}

// -- jsonHandler --

func TestJsonHandler_Response_NoGeo(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/json", nil)
	r.RemoteAddr = "5.6.7.8:1234"

	w := httptest.NewRecorder()
	jsonHandler("")(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	var payload JSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if payload.IP != "5.6.7.8" {
		t.Errorf("ip: got %q, want %q", payload.IP, "5.6.7.8")
	}
	if payload.Time == "" {
		t.Error("time_utc is empty")
	}
	if payload.Country != "" || payload.ASN != "" {
		t.Error("geo fields should be empty when token is not set")
	}
}

func TestJsonHandler_Response_WithGeo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IPInfo{
			ASN:           "AS15169",
			ASName:        "Google LLC",
			CountryCode:   "US",
			Country:       "United States",
			ContinentCode: "NA",
			Continent:     "North America",
		})
	}))
	defer srv.Close()

	old := ipinfoBaseURL
	ipinfoBaseURL = srv.URL + "/"
	defer func() { ipinfoBaseURL = old }()

	r := httptest.NewRequest(http.MethodGet, "/json", nil)
	r.RemoteAddr = "8.8.8.8:1234"

	w := httptest.NewRecorder()
	jsonHandler("test-token")(w, r)

	var payload JSONResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if payload.Country != "United States" {
		t.Errorf("Country: got %q, want %q", payload.Country, "United States")
	}
	if payload.CountryCode != "US" {
		t.Errorf("CountryCode: got %q, want %q", payload.CountryCode, "US")
	}
	if payload.Continent != "North America" {
		t.Errorf("Continent: got %q, want %q", payload.Continent, "North America")
	}
	if payload.ContinentCode != "NA" {
		t.Errorf("ContinentCode: got %q, want %q", payload.ContinentCode, "NA")
	}
	if payload.ASN != "AS15169" {
		t.Errorf("ASN: got %q, want %q", payload.ASN, "AS15169")
	}
	if payload.ASName != "Google LLC" {
		t.Errorf("ASName: got %q, want %q", payload.ASName, "Google LLC")
	}
}

func TestJsonHandler_MethodNotAllowed(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		r := httptest.NewRequest(method, "/json", nil)
		w := httptest.NewRecorder()
		jsonHandler("")(w, r)

		if w.Result().StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("method %s: got %d, want %d", method, w.Result().StatusCode, http.StatusMethodNotAllowed)
		}
	}
}
