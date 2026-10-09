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

func TestFeatureParams(t *testing.T) {
	tests := []struct {
		name     string
		f        Features
		copyOff  string
		pasteOff string
		driveOn  string
	}{
		{"all off", Features{}, "true", "true", "false"},
		{"clipboard both ways", Features{ClipboardUpload: true, ClipboardDownload: true}, "false", "false", "false"},
		{"upload only", Features{ClipboardUpload: true}, "true", "false", "false"},
		{"download only", Features{ClipboardDownload: true}, "false", "true", "false"},
		{"files on, no drive path", Features{FileUpload: true}, "true", "true", "false"},
		{"files on with drive", Features{FileUpload: true, FileDownload: true, DrivePath: "/drives/x"}, "true", "true", "true"},
		{"drive path but files off", Features{DrivePath: "/drives/x"}, "true", "true", "false"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := rdpParams(Target{Width: 800, Height: 600, DPI: 96}, tc.f)
			if p["disable-copy"] != tc.copyOff || p["disable-paste"] != tc.pasteOff || p["enable-drive"] != tc.driveOn {
				t.Fatalf("copy=%s paste=%s drive=%s", p["disable-copy"], p["disable-paste"], p["enable-drive"])
			}
		})
	}
}
