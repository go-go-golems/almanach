package app

import (
	"log"
	"os"
	"path/filepath"
)

// fontAntialiasDisabledConf is a fontconfig configuration that disables
// grayscale anti-aliasing while still including the system font configuration so
// font matching keeps working. It is fed to the headless render browser via the
// FONTCONFIG_FILE environment variable.
//
// Why: the render pipeline always produces a 1-bit thermal bitmap. With AA on,
// Chrome/FreeType draws sub-pixel glyph strokes as light-gray pixels, and the
// downstream luminance threshold (PngToBitmap) discards them, dropping strokes in
// small text. With AA off, FreeType's hint-aware monochrome rasterizer makes the
// 1-bit decision per glyph, which preserves stems and keeps small text legible.
// This affects only the headless render browser, not the studio UI a human edits.
const fontAntialiasDisabledConf = `<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <include ignore_missing="yes">/etc/fonts/fonts.conf</include>
  <include ignore_missing="yes">/usr/local/etc/fonts/fonts.conf</include>
  <match target="pattern">
    <edit name="antialias" mode="assign"><bool>false</bool></edit>
  </match>
  <match target="font">
    <edit name="antialias" mode="assign"><bool>false</bool></edit>
    <edit name="hinting" mode="assign"><bool>true</bool></edit>
    <edit name="hintstyle" mode="assign"><const>hintfull</const></edit>
  </match>
</fontconfig>
`

// renderFontEnv returns environment variables for the headless render browser
// that disable font anti-aliasing, so text rasterizes monochrome and survives the
// 1-bit conversion. Set ALMANACH_FONT_ANTIALIAS=1 to keep the browser default
// (anti-aliased) instead. Returns nil if AA is left enabled or the config file
// cannot be written (in which case the browser falls back to its default).
func renderFontEnv() []string {
	switch os.Getenv("ALMANACH_FONT_ANTIALIAS") {
	case "1", "true", "yes":
		return nil
	}
	path := filepath.Join(os.TempDir(), "almanach-fonts-noaa.conf")
	if err := os.WriteFile(path, []byte(fontAntialiasDisabledConf), 0o600); err != nil {
		log.Printf("[render] could not write no-AA fontconfig (%v); using browser default", err)
		return nil
	}
	return []string{"FONTCONFIG_FILE=" + path}
}
