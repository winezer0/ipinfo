package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFoundConfigReadError(t *testing.T) {
	app := filepath.Join(t.TempDir(), "service")
	if err := os.Mkdir(app+".yaml", 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig("", app); err == nil {
		t.Fatal("unreadable discovered config silently accepted")
	}
}

func TestGeneratedConfigAndTokenCompatibility(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.yaml")
	if err := GenDefaultConfig(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != GetDefaultConfig() {
		t.Fatal("generation changed the template")
	}
	cfg, err := LoadConfig(path, "service")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "test-token-123456", "your-secret-token"} {
		cfg.Auth.Token = token
		if err := cfg.Validate(); err != nil {
			t.Fatalf("token %q unexpectedly rejected: %v", token, err)
		}
	}
	cfg.HTTP.MaxBatchSize = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative batch limit accepted")
	}
	cfg.HTTP.MaxBatchSize = 100
	cfg.HTTP.TrustedProxies = []string{"invalid"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid proxy CIDR accepted")
	}
}
