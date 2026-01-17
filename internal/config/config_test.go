package config

import (
	"os"
	"testing"
)

func TestConfig_StorageDirDefault(t *testing.T) {
	// Unset env var to test default
	os.Unsetenv("STORAGE_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.StorageDir != "./storage" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "./storage")
	}
}

func TestConfig_StorageDirFromEnv(t *testing.T) {
	os.Setenv("STORAGE_DIR", "/custom/path")
	defer os.Unsetenv("STORAGE_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.StorageDir != "/custom/path" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "/custom/path")
	}
}
