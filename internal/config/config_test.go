package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ENV", "")
	t.Setenv("MAGIC_LINK_BASE_URL", "")
	t.Setenv("MAGIC_LINK_TTL_MINUTES", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	require.Equal(t, "6176", cfg.Port)
	require.Equal(t, "prod", cfg.Env)
	require.Equal(t, "http://localhost:3000/auth/verify", cfg.MagicLinkBaseURL)
	require.Equal(t, 15*time.Minute, cfg.MagicLinkTTL)
	require.Equal(t, "", cfg.DatabaseURL)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "7000")
	t.Setenv("ENV", "dev")
	t.Setenv("MAGIC_LINK_BASE_URL", "https://example.com/auth/verify")
	t.Setenv("MAGIC_LINK_TTL_MINUTES", "30")
	t.Setenv("DATABASE_URL", "postgres://example")

	cfg := Load()

	require.Equal(t, "7000", cfg.Port)
	require.Equal(t, "dev", cfg.Env)
	require.Equal(t, "https://example.com/auth/verify", cfg.MagicLinkBaseURL)
	require.Equal(t, 30*time.Minute, cfg.MagicLinkTTL)
	require.Equal(t, "postgres://example", cfg.DatabaseURL)
}

func TestLoadIgnoresInvalidTTL(t *testing.T) {
	t.Setenv("MAGIC_LINK_TTL_MINUTES", "not-a-number")

	cfg := Load()

	require.Equal(t, 15*time.Minute, cfg.MagicLinkTTL)
}
