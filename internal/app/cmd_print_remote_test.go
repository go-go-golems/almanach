package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPostLayoutToRemoteAlmanachPostsJSONLayout(t *testing.T) {
	var gotContentType string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"printed":    true,
			"width":      384,
			"height":     1200,
			"renderedAt": "2026-05-15T22:00:00Z",
			"printerResponse": map[string]any{
				"segments": 3,
			},
		})
	}))
	defer server.Close()

	resp, err := postLayoutToRemoteAlmanach(context.Background(), remotePostRequest{
		URL:        server.URL,
		LayoutJSON: `{"blocks":[]}`,
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("postLayoutToRemoteAlmanach returned error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", gotContentType)
	}
	if _, ok := gotBody["blocks"]; !ok {
		t.Fatalf("request body = %#v, want blocks field", gotBody)
	}
	if !resp.OK || !resp.Printed {
		t.Fatalf("response ok/printed = %v/%v, want true/true", resp.OK, resp.Printed)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestPostLayoutToRemoteAlmanachReturnsRemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": "printer failed",
		})
	}))
	defer server.Close()

	_, err := postLayoutToRemoteAlmanach(context.Background(), remotePostRequest{
		URL:        server.URL,
		LayoutJSON: `{"blocks":[]}`,
		Timeout:    time.Second,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); got != "remote almanach returned 502: printer failed" {
		t.Fatalf("error = %q", got)
	}
}
