package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetEnvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "")
	if got := getEnvDuration("TEST_DURATION", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback for empty value, got %v", got)
	}

	t.Setenv("TEST_DURATION", "15")
	if got := getEnvDuration("TEST_DURATION", 5*time.Second); got != 15*time.Second {
		t.Fatalf("expected seconds parsing, got %v", got)
	}

	t.Setenv("TEST_DURATION", "250ms")
	if got := getEnvDuration("TEST_DURATION", 5*time.Second); got != 250*time.Millisecond {
		t.Fatalf("expected duration parsing, got %v", got)
	}

	t.Setenv("TEST_DURATION", "0")
	if got := getEnvDuration("TEST_DURATION", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback for non-positive value, got %v", got)
	}

	t.Setenv("TEST_DURATION", "garbage")
	if got := getEnvDuration("TEST_DURATION", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback for invalid value, got %v", got)
	}
}

func TestHealthAndReady(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from /healthz, got %d", rr.Code)
	}

	var health map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Fatalf("invalid json response from /healthz: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("expected status=ok from /healthz, got %q", health["status"])
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	handleReady(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from /readyz, got %d", rr.Code)
	}

	var ready map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &ready); err != nil {
		t.Fatalf("invalid json response from /readyz: %v", err)
	}
	if ready["status"] != "ready" {
		t.Fatalf("expected status=ready from /readyz, got %q", ready["status"])
	}

	t.Setenv("KUBERNETES_SERVICE_HOST", "kubernetes.default.svc")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")
	httpClient = nil
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	handleReady(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from /readyz when in-cluster client is not ready, got %d", rr.Code)
	}
}

func TestHandleIngressesMethodAndFallback(t *testing.T) {
	h := handleIngresses(time.Second)

	req := httptest.NewRequest(http.MethodPost, "/api/ingresses", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for non-GET request, got %d", rr.Code)
	}

	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	req = httptest.NewRequest(http.MethodGet, "/api/ingresses", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET request without kube env, got %d", rr.Code)
	}

	var payload map[string][]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json from /api/ingresses: %v", err)
	}
	items, ok := payload["items"]
	if !ok {
		t.Fatalf("expected items key in response")
	}
	if len(items) != 0 {
		t.Fatalf("expected empty items when not running in cluster, got %d items", len(items))
	}
}
