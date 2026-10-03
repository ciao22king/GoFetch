package app

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigDir points os.UserConfigDir at a temporary directory so tests
// never touch the developer's real configuration.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestLoadConfigMissing(t *testing.T) {
	withConfigDir(t)
	cfg := LoadConfig()
	if cfg != (Config{}) {
		t.Fatalf("LoadConfig() with no file = %+v, want zero Config", cfg)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := withConfigDir(t)
	want := Config{DefaultDir: "~/Code", Depth: 1, NoBuild: true}
	if err := SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "gofetch", "config.json")); err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if got := LoadConfig(); got != want {
		t.Fatalf("LoadConfig() = %+v, want %+v", got, want)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := withConfigDir(t)
	configDir := filepath.Join(dir, "gofetch")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg := LoadConfig(); cfg != (Config{}) {
		t.Fatalf("LoadConfig() with invalid JSON = %+v, want zero Config", cfg)
	}
}

func TestConfigSanitized(t *testing.T) {
	cfg := Config{DefaultDir: "  ~/Code  ", Depth: -3}.sanitized()
	if cfg.Depth != 0 {
		t.Errorf("negative depth should become 0, got %d", cfg.Depth)
	}
	if cfg.DefaultDir != "~/Code" {
		t.Errorf("DefaultDir should be trimmed, got %q", cfg.DefaultDir)
	}
}
