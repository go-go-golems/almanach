package app

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all service configuration, loaded from environment variables.
type Config struct {
	Port         int    // HTTP listen port (default 8199)
	WebDir       string // Directory serving almanach SPA static files
	PrinterIP    string // ESP32 stoms3r device IP address
	ChromePath   string // Path to Chrome/Chromium binary (empty = auto-detect)
	ChromeWSURL  string // WebSocket URL for remote Chrome (e.g. "ws://chrome:9222")
	PaperWidth   int    // Default paper width in pixels (default 384)
	BodyScale    float64
	FeedLines    int // Default feed lines after printing (default 3)
	DefaultTheme string
	LogLevel     string // debug, info, warn, error
	StateFile    string // Local render/setup server state file
}

func LoadConfig() Config {
	return Config{
		Port:         envInt("ALMANACH_PORT", 8199),
		WebDir:       envStr("ALMANACH_WEB_DIR", "./web/dist"),
		PrinterIP:    envStr("ALMANACH_PRINTER_IP", ""),
		ChromePath:   envStr("ALMANACH_CHROME_PATH", ""),
		ChromeWSURL:  envStr("CHROME_WS_URL", ""),
		PaperWidth:   envInt("ALMANACH_PAPER_WIDTH", 384),
		BodyScale:    envFloat("ALMANACH_FONT_SCALE", 1.4),
		FeedLines:    envInt("ALMANACH_DEFAULT_FEED", 3),
		DefaultTheme: envStr("ALMANACH_DEFAULT_THEME", "minimal"),
		LogLevel:     envStr("ALMANACH_LOG_LEVEL", "info"),
		StateFile:    envStr("ALMANACH_STATE_FILE", defaultStateFile()),
	}
}

func defaultStateFile() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "almanach", "render-service", "state.json")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "almanach", "render-service", "state.json")
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
