package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRestrictToLoopbackHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want int
	}{
		{name: "IPv4 with port", host: "127.0.0.1:8080", want: http.StatusNoContent},
		{name: "localhost with port", host: "localhost:8080", want: http.StatusNoContent},
		{name: "Vite localhost port", host: "localhost:5173", want: http.StatusNoContent},
		{name: "IPv6 with port", host: "[::1]:8080", want: http.StatusNoContent},
		{name: "bare localhost", host: "localhost", want: http.StatusNoContent},
		{name: "attacker host", host: "attacker.example:8080", want: http.StatusForbidden},
		{name: "loopback prefix attacker host", host: "127.0.0.1.attacker.example:8080", want: http.StatusForbidden},
		{name: "empty host", host: "", want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Host = tt.host
			response := httptest.NewRecorder()

			restrictToLoopback(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(response, request)

			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d", response.Code, tt.want)
			}
			if tt.want == http.StatusForbidden {
				assertForbiddenHostError(t, response)
			}
		})
	}
}

func TestRestrictToLoopbackFetchMetadata(t *testing.T) {
	tests := []struct {
		name         string
		secFetchSite string
		want         int
	}{
		{name: "cross-site", secFetchSite: "cross-site", want: http.StatusForbidden},
		{name: "same-origin", secFetchSite: "same-origin", want: http.StatusNoContent},
		{name: "none", secFetchSite: "none", want: http.StatusNoContent},
		{name: "absent", want: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Host = "127.0.0.1:8080"
			if tt.secFetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			response := httptest.NewRecorder()

			restrictToLoopback(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(response, request)

			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d", response.Code, tt.want)
			}
			if tt.want == http.StatusForbidden {
				assertForbiddenHostError(t, response)
			}
		})
	}
}

func assertForbiddenHostError(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "forbidden_host" {
		t.Errorf("error code = %q, want forbidden_host", body.Error.Code)
	}
}
