package enrichment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPEnricherPublicIPv4Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ip") != "8.8.8.8" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("request = %s auth=%q", r.URL.String(), r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ip_address": "8.8.8.8", "country": "US", "region": "CA", "city": "Mountain View", "latitude": 37.4, "longitude": -122.1, "asn": "AS15169", "organization": "Example ISP", "confidence": 0.8})
	}))
	defer server.Close()
	analyzer, err := NewHTTPEnricher(HTTPConfig{APIKey: "test-key", Endpoint: server.URL, Provider: "test-provider", Timeout: time.Second, MaxResponseSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Lookup(context.Background(), "8.8.8.8")
	if err != nil || result.Status != StatusEnriched || result.Provider == nil || *result.Provider != "test-provider" || result.Country == nil || result.RetrievedAt == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestHTTPEnricherPublicIPv6Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ip") != "2001:4860:4860::8888" {
			t.Fatalf("ip = %q", r.URL.Query().Get("ip"))
		}
		_, _ = w.Write([]byte(`{"ip_address":"2001:4860:4860::8888","asn":"AS15169"}`))
	}))
	defer server.Close()
	analyzer, err := NewHTTPEnricher(HTTPConfig{APIKey: "key", Endpoint: server.URL, Timeout: time.Second, MaxResponseSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Lookup(context.Background(), "2001:4860:4860::8888")
	if err != nil || result.Status != StatusEnriched || result.ASN == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestNonPublicAndInvalidIPsNeverCallProvider(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { calls++ }))
	defer server.Close()
	analyzer, err := NewHTTPEnricher(HTTPConfig{APIKey: "key", Endpoint: server.URL, Timeout: time.Second, MaxResponseSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "fe80::1", "224.0.0.1", "192.0.2.10", "2001:db8::1", "not-an-ip"} {
		result, err := analyzer.Lookup(context.Background(), ip)
		if err != nil || result.Status != StatusNotApplicable {
			t.Fatalf("ip=%s result=%#v err=%v", ip, result, err)
		}
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d", calls)
	}
}

func TestUnavailableEnricherIsExplicit(t *testing.T) {
	analyzer, err := NewHTTPEnricher(HTTPConfig{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Lookup(context.Background(), "8.8.8.8")
	if err != nil || result.Status != StatusNotConfigured || result.Failure == nil || result.Failure.Code != "IP_ENRICHMENT_NOT_CONFIGURED" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestHTTPEnricherProviderFailuresAreStructured(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
		want    string
	}{
		{"http error", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusBadGateway) }), "IP_ENRICHMENT_HTTP_ERROR"},
		{"malformed", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not-json")) }), "IP_ENRICHMENT_MALFORMED_RESPONSE"},
		{"empty object", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) }), "IP_ENRICHMENT_INVALID_RESPONSE"},
		{"unknown field", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"country":"US","unexpected":true}`))
		}), "IP_ENRICHMENT_MALFORMED_RESPONSE"},
		{"invalid value", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"confidence":2}`)) }), "IP_ENRICHMENT_INVALID_RESPONSE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.handler)
			defer server.Close()
			analyzer, err := NewHTTPEnricher(HTTPConfig{APIKey: "key", Endpoint: server.URL, Timeout: time.Second, MaxResponseSize: 4096})
			if err != nil {
				t.Fatal(err)
			}
			result, err := analyzer.Lookup(context.Background(), "8.8.8.8")
			if err == nil || result.Status != StatusFailed || result.Failure == nil || result.Failure.Code != test.want {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}
}

func TestHTTPEnricherTimeoutAndResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer server.Close()
	analyzer, err := NewHTTPEnricher(HTTPConfig{APIKey: "key", Endpoint: server.URL, Timeout: 10 * time.Millisecond, MaxResponseSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Lookup(context.Background(), "8.8.8.8")
	if err == nil || result.Status != StatusFailed || result.Failure == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	limitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", 100))) }))
	defer limitServer.Close()
	limited, err := NewHTTPEnricher(HTTPConfig{APIKey: "key", Endpoint: limitServer.URL, Timeout: time.Second, MaxResponseSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	result, err = limited.Lookup(context.Background(), "8.8.8.8")
	if err == nil || result.Failure == nil || result.Failure.Code != "IP_ENRICHMENT_RESPONSE_TOO_LARGE" {
		t.Fatalf("oversized result=%#v err=%v", result, err)
	}
}

func TestHTTPConfigFromEnvRejectsInvalidValues(t *testing.T) {
	t.Setenv("IP_ENRICHMENT_TIMEOUT", "bad")
	if _, err := HTTPConfigFromEnv(); err == nil {
		t.Fatal("expected invalid timeout error")
	}
	t.Setenv("IP_ENRICHMENT_TIMEOUT", "")
	t.Setenv("IP_ENRICHMENT_MAX_RESPONSE_BYTES", "0")
	if _, err := HTTPConfigFromEnv(); err == nil {
		t.Fatal("expected invalid response size error")
	}
}
