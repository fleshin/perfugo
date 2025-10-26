package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterRegistersHealthRoute(t *testing.T) {
	router := newRouter()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected /healthz to return 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
}
