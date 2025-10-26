package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHomeRedirectsToLanding(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	Home(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/assets/index.html" {
		t.Fatalf("expected redirect to /assets/index.html, got %q", loc)
	}
}
