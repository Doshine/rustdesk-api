package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestSecretFileOverrideLoadsWithoutLoggingValue(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "redis-password")
	if err := os.WriteFile(secretPath, []byte("strong-redis-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUSTDESK_API_CACHE_REDIS_PWD_FILE", secretPath)

	v := viper.New()
	if err := applySecretFileOverrides(v); err != nil {
		t.Fatalf("apply secret file: %v", err)
	}
	if got := v.GetString("cache.redis-pwd"); got != "strong-redis-secret" {
		t.Fatal("mounted secret was not loaded exactly")
	}
}

func TestSecretFileOverrideRejectsAmbiguousDirectValue(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "database-password")
	if err := os.WriteFile(secretPath, []byte("file-value"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUSTDESK_API_POSTGRESQL_PASSWORD_FILE", secretPath)
	t.Setenv("RUSTDESK_API_POSTGRESQL_PASSWORD", "direct-value")

	if err := applySecretFileOverrides(viper.New()); err == nil {
		t.Fatal("direct secret and _FILE secret must be mutually exclusive")
	}
}

func TestSecretFileRejectsEmptyAndOversizedValues(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretFile(empty); err == nil {
		t.Fatal("empty secret file must be rejected")
	}

	large := filepath.Join(t.TempDir(), "large")
	if err := os.WriteFile(large, make([]byte, maxSecretFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretFile(large); err == nil {
		t.Fatal("oversized secret file must be rejected")
	}
}
