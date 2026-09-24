package config

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// xkbLayouts maps XKB layouts (and "layout(variant)") to the on-screen
// keyboard layouts shipped with eDEX-UI.
var xkbLayouts = map[string]string{
	"us": "en-US", "us(dvorak)": "en-DVORAK", "us(colemak)": "en-COLEMAK", "us(workman)": "en-WORKMAN",
	"us(norman)": "en-NORMAN", "gb": "en-GB", "br": "pt-BR", "de": "de-DE", "fr": "fr-FR",
	"fr(bepo)": "fr-BEPO", "es": "es-ES", "latam": "es-LAT", "it": "it-IT", "dk": "da-DK",
	"hu": "hu-HU", "se": "sv-SE", "be": "nl-BE", "tr": "tr-TR-Q", "tr(f)": "tr-TR-F",
}

// keyboardFor returns the on-screen layout for an XKB layout and variant.
func keyboardFor(layout, variant string) (string, bool) {
	layout, variant = strings.TrimSpace(layout), strings.TrimSpace(variant)
	if kb, ok := xkbLayouts[layout+"("+variant+")"]; ok && variant != "" {
		return kb, true
	}
	kb, ok := xkbLayouts[layout]
	return kb, ok
}

var gnomeSource = regexp.MustCompile(`\('xkb', '([a-z]+)(?:\+([a-z0-9_-]+))?'\)`)

// DetectKeyboard guesses the on-screen keyboard layout from the system
// keyboard, for new installs. It falls back to en-US.
func DetectKeyboard() string {
	if runtime.GOOS != "linux" {
		return "en-US"
	}
	// The layout chosen in GNOME applies to the user session.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "gsettings", "get", "org.gnome.desktop.input-sources", "sources").Output(); err == nil {
		if m := gnomeSource.FindStringSubmatch(string(out)); m != nil {
			if kb, ok := keyboardFor(m[1], m[2]); ok {
				return kb
			}
		}
	}
	// System default (Debian, Ubuntu...).
	if data, err := os.ReadFile("/etc/default/keyboard"); err == nil {
		var layout, variant string
		for _, line := range strings.Split(string(data), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			v = strings.Trim(v, `"'`)
			switch k {
			case "XKBLAYOUT":
				layout, _, _ = strings.Cut(v, ",")
			case "XKBVARIANT":
				variant, _, _ = strings.Cut(v, ",")
			}
		}
		if kb, ok := keyboardFor(layout, variant); ok {
			return kb
		}
	}
	return "en-US"
}
