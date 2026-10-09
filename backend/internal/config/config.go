// Package config loads runtime configuration from environment variables only.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr          string
	DatabaseURL         string
	GuacdAddr           string
	AllowedOrigin       string
	CookieSecure        bool
	SessionTTL          time.Duration
	AllowPrivateTargets bool
	KEK                 []byte
}

func Load() (*Config, error) {
	c := &Config{
		ListenAddr:    env("LISTEN_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		GuacdAddr:     env("GUACD_ADDR", "guacd:4822"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}
	var err error
	if c.CookieSecure, err = strconv.ParseBool(env("COOKIE_SECURE", "true")); err != nil {
		return nil, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	if c.AllowPrivateTargets, err = strconv.ParseBool(env("ALLOW_PRIVATE_TARGETS", "false")); err != nil {
		return nil, fmt.Errorf("ALLOW_PRIVATE_TARGETS: %w", err)
	}
	if c.SessionTTL, err = time.ParseDuration(env("SESSION_TTL", "12h")); err != nil {
		return nil, fmt.Errorf("SESSION_TTL: %w", err)
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.AllowedOrigin == "" {
		return nil, fmt.Errorf("ALLOWED_ORIGIN is required (e.g. https://rdp.example.com)")
	}
	kek, err := base64.StdEncoding.DecodeString(os.Getenv("APP_KEK"))
	if err != nil || len(kek) != 32 {
		return nil, fmt.Errorf("APP_KEK must be base64 of exactly 32 bytes (openssl rand -base64 32)")
	}
	c.KEK = kek
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
