// Package config loads runtime configuration from environment variables only
// (12-factor). See deploy/.env.example for every key.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr          string
	DatabaseURL         string
	GuacdAddr           string
	AllowedOrigin       string
	AllowedHosts        []string
	CookieSecure        bool
	SessionTTL          time.Duration
	AllowPrivateTargets bool
	MaxTunnels          int
	MaxTunnelsPerUser   int
	MaxTunnelDuration   time.Duration
	AuditRetentionDays  int
	IdempotencyTTL      time.Duration
	ClipboardUpload     bool   // local → remote (guacd disable-paste)
	ClipboardDownload   bool   // remote → local (guacd disable-copy)
	FileUpload          bool   // local → remote drive (guacd disable-upload)
	FileDownload        bool   // remote drive → local (guacd disable-download)
	DriveRoot           string // shared guacd/api volume holding per-session drives
	MaxDriveBytes       int64
	KeyID               byte
	Keys                map[byte][]byte // current + optional previous KEK
}

func Load() (*Config, error) {
	c := &Config{
		ListenAddr:    env("LISTEN_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		GuacdAddr:     env("GUACD_ADDR", "guacd:4822"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}
	var errs []string
	parse := func(key, def string, fn func(string) error) {
		if err := fn(env(key, def)); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	parse("COOKIE_SECURE", "true", func(v string) (err error) { c.CookieSecure, err = strconv.ParseBool(v); return })
	parse("ALLOW_PRIVATE_TARGETS", "false", func(v string) (err error) { c.AllowPrivateTargets, err = strconv.ParseBool(v); return })
	parse("SESSION_TTL", "12h", func(v string) (err error) { c.SessionTTL, err = time.ParseDuration(v); return })
	parse("MAX_TUNNEL_DURATION", "8h", func(v string) (err error) { c.MaxTunnelDuration, err = time.ParseDuration(v); return })
	parse("IDEMPOTENCY_TTL", "24h", func(v string) (err error) { c.IdempotencyTTL, err = time.ParseDuration(v); return })
	// FEATURE_CLIPBOARD is the master default; _UPLOAD/_DOWNLOAD override per direction.
	clip := env("FEATURE_CLIPBOARD", "true")
	parse("FEATURE_CLIPBOARD_UPLOAD", clip, func(v string) (err error) { c.ClipboardUpload, err = strconv.ParseBool(v); return })
	parse("FEATURE_CLIPBOARD_DOWNLOAD", clip, func(v string) (err error) { c.ClipboardDownload, err = strconv.ParseBool(v); return })
	files := env("FEATURE_FILE_TRANSFER", "false")
	parse("FEATURE_FILE_UPLOAD", files, func(v string) (err error) { c.FileUpload, err = strconv.ParseBool(v); return })
	parse("FEATURE_FILE_DOWNLOAD", files, func(v string) (err error) { c.FileDownload, err = strconv.ParseBool(v); return })
	parse("MAX_DRIVE_MB", "1024", func(v string) error {
		n, err := strconv.ParseInt(v, 10, 64)
		c.MaxDriveBytes = n << 20
		return err
	})
	c.DriveRoot = env("DRIVE_ROOT", "/drives")
	parse("MAX_TUNNELS", "20", func(v string) (err error) { c.MaxTunnels, err = strconv.Atoi(v); return })
	parse("MAX_TUNNELS_PER_USER", "5", func(v string) (err error) { c.MaxTunnelsPerUser, err = strconv.Atoi(v); return })
	parse("AUDIT_RETENTION_DAYS", "180", func(v string) (err error) { c.AuditRetentionDays, err = strconv.Atoi(v); return })

	if c.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.AllowedOrigin == "" {
		errs = append(errs, "ALLOWED_ORIGIN is required (e.g. https://rdp.example.com)")
	}
	if hosts := env("ALLOWED_HOSTS", strings.TrimPrefix(strings.TrimPrefix(c.AllowedOrigin, "https://"), "http://")); hosts != "" {
		for _, h := range strings.Split(hosts, ",") {
			if h = strings.TrimSpace(h); h != "" {
				c.AllowedHosts = append(c.AllowedHosts, strings.ToLower(h))
			}
		}
	}
	if err := c.loadKeys(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("config: %s", strings.Join(errs, "; "))
	}
	return c, nil
}

// loadKeys reads APP_KEK (+APP_KEK_ID) and, during a rotation, the old key
// as APP_KEK_PREVIOUS (+APP_KEK_PREVIOUS_ID).
func (c *Config) loadKeys() error {
	c.Keys = map[byte][]byte{}
	id, err := keyID("APP_KEK_ID", "1")
	if err != nil {
		return err
	}
	k, err := kek("APP_KEK")
	if err != nil {
		return err
	}
	c.KeyID, c.Keys[id] = id, k
	if os.Getenv("APP_KEK_PREVIOUS") == "" {
		return nil
	}
	pid, err := keyID("APP_KEK_PREVIOUS_ID", "")
	if err != nil {
		return err
	}
	if pid == id {
		return fmt.Errorf("APP_KEK_PREVIOUS_ID must differ from APP_KEK_ID")
	}
	pk, err := kek("APP_KEK_PREVIOUS")
	if err != nil {
		return err
	}
	c.Keys[pid] = pk
	return nil
}

func keyID(name, def string) (byte, error) {
	n, err := strconv.Atoi(env(name, def))
	if err != nil || n < 1 || n > 255 {
		return 0, fmt.Errorf("%s must be 1-255", name)
	}
	return byte(n), nil
}

func kek(name string) ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(os.Getenv(name))
	if err != nil || len(k) != 32 {
		return nil, fmt.Errorf("%s must be base64 of exactly 32 bytes (openssl rand -base64 32)", name)
	}
	return k, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
