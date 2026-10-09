package guac

import "testing"

func TestQualityParams(t *testing.T) {
	tests := []struct {
		quality   string
		depth     string
		wallpaper string
		smoothing string
	}{
		{"high", "32", "true", "true"},
		{"balanced", "24", "false", "true"},
		{"low", "16", "false", "false"},
		{"", "24", "false", "true"},      // default
		{"bogus", "24", "false", "true"}, // unknown → balanced
	}
	for _, tc := range tests {
		t.Run(tc.quality, func(t *testing.T) {
			p := rdpParams(Target{IP: "203.0.113.1", Port: 3389, Quality: tc.quality, Width: 800, Height: 600, DPI: 96}, Features{})
			if p["color-depth"] != tc.depth || p["enable-wallpaper"] != tc.wallpaper || p["enable-font-smoothing"] != tc.smoothing {
				t.Fatalf("got depth=%s wallpaper=%s smoothing=%s", p["color-depth"], p["enable-wallpaper"], p["enable-font-smoothing"])
			}
			if p["disable-bitmap-caching"] != "false" || p["disable-copy"] != "true" {
				t.Fatalf("caching/clipboard defaults wrong: %v", p)
			}
		})
	}
}
