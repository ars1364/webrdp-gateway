package guac

// Bandwidth profiles map a user-facing quality to FreeRDP/guacd parameters.
// "low" trades fidelity for bandwidth: 16-bit colour, every visual effect
// off, bitmap/glyph caches on (guacd also switches to lossy JPEG/WebP on its
// own when updates are large; the client advertises both).
var qualityProfiles = map[string]map[string]string{
	"high": {
		"color-depth":                "32",
		"enable-wallpaper":           "true",
		"enable-theming":             "true",
		"enable-font-smoothing":      "true",
		"enable-full-window-drag":    "true",
		"enable-desktop-composition": "true",
		"enable-menu-animations":     "true",
	},
	"balanced": {
		"color-depth":                "24",
		"enable-wallpaper":           "false",
		"enable-theming":             "true",
		"enable-font-smoothing":      "true",
		"enable-full-window-drag":    "false",
		"enable-desktop-composition": "false",
		"enable-menu-animations":     "false",
	},
	"low": {
		"color-depth":                "16",
		"enable-wallpaper":           "false",
		"enable-theming":             "false",
		"enable-font-smoothing":      "false",
		"enable-full-window-drag":    "false",
		"enable-desktop-composition": "false",
		"enable-menu-animations":     "false",
	},
}

// qualityParams returns the profile for q, defaulting to "balanced".
func qualityParams(q string) map[string]string {
	p, ok := qualityProfiles[q]
	if !ok {
		p = qualityProfiles["balanced"]
	}
	out := map[string]string{
		"disable-bitmap-caching":    "false",
		"disable-offscreen-caching": "false",
		"disable-glyph-caching":     "false",
	}
	for k, v := range p {
		out[k] = v
	}
	return out
}
