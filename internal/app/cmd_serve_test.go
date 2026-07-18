package app

import (
	"testing"
	"time"
)

func TestHTTPWriteTimeoutCoversRenderAndResponse(t *testing.T) {
	if got, want := httpWriteTimeout(2*time.Minute), 2*time.Minute+renderResponseWriteOverhead; got != want {
		t.Errorf("httpWriteTimeout(2m) = %s, want %s", got, want)
	}
}

func TestHTTPWriteTimeoutUsesRenderDefault(t *testing.T) {
	if got, want := httpWriteTimeout(0), defaultChromeRenderTimeout+renderResponseWriteOverhead; got != want {
		t.Errorf("httpWriteTimeout(0) = %s, want %s", got, want)
	}
}
