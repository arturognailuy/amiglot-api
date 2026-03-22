package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	Port                   string
	DatabaseURL            string
	Env                    string
	MagicLinkBaseURL       string
	MagicLinkTTL           time.Duration
	MatchMinOverlapMinutes int
}

// Load reads configuration from environment variables.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "6176"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "prod"
	}

	magicLinkBaseURL := os.Getenv("MAGIC_LINK_BASE_URL")
	if magicLinkBaseURL == "" {
		magicLinkBaseURL = "http://localhost:3000/auth/verify"
	}

	ttlMinutes := 15
	if raw := os.Getenv("MAGIC_LINK_TTL_MINUTES"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			ttlMinutes = value
		}
	}

	magicLinkTTL := time.Duration(ttlMinutes) * time.Minute

	minOverlap := 60
	if raw := os.Getenv("MATCH_MIN_OVERLAP_MINUTES"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			minOverlap = value
		}
	}

	return Config{
		Port:                   port,
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		Env:                    env,
		MagicLinkBaseURL:       magicLinkBaseURL,
		MagicLinkTTL:           magicLinkTTL,
		MatchMinOverlapMinutes: minOverlap,
	}
}
