package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Network.Port != 25565 {
		t.Fatalf("port = %d, want 25565", cfg.Network.Port)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(defaultConfig) {
		t.Fatal("created config does not match the default")
	}
}

func TestLoadConfigDoesNotOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	want := []byte("network:\n  port: 25570\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("existing config was overwritten")
	}
}
