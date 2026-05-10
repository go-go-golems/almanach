package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetupStaticRoutesServeBundledAssets(t *testing.T) {
	mux := http.NewServeMux()
	registerStaticRoutes(mux, "")

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/setup", contentType: "text/html; charset=utf-8", contains: "/setup/bundle.js"},
		{path: "/setup/bundle.js", contentType: "application/javascript; charset=utf-8", contains: "ALMANACH SETUP"},
		{path: "/almanach", contentType: "text/html; charset=utf-8", contains: "/almanach/bundle.js"},
		{path: "/almanach/bundle.js", contentType: "application/javascript; charset=utf-8", contains: "Almanach"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if got := rr.Header().Get("Content-Type"); got != tt.contentType {
				t.Fatalf("content-type: got %q, want %q", got, tt.contentType)
			}
			if body := rr.Body.String(); !strings.Contains(body, tt.contains) {
				t.Fatalf("body for %s does not contain %q", tt.path, tt.contains)
			}
		})
	}
}
