package app

import (
	"testing"
	"time"
)

func TestEnvDuration(t *testing.T) {
	t.Setenv("ALMANACH_TEST_DURATION", "90s")
	if got := envDuration("ALMANACH_TEST_DURATION", time.Minute); got != 90*time.Second {
		t.Errorf("envDuration = %s, want 90s", got)
	}
}

func TestEnvDurationFallsBackForInvalidOrNonPositiveValues(t *testing.T) {
	const fallback = time.Minute
	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Setenv("ALMANACH_TEST_DURATION", value)
		if got := envDuration("ALMANACH_TEST_DURATION", fallback); got != fallback {
			t.Errorf("envDuration(%q) = %s, want fallback %s", value, got, fallback)
		}
	}
}
