package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config holds user preferences stored next to the history file. Every field
// is optional: a zero value means "fall back to the built-in default", so a
// partial or missing config file is always valid.
type Config struct {
	// DefaultDir is the parent directory suggested for new clones. The -dir
	// flag overrides it.
	DefaultDir string `json:"default_dir"`
	// Depth is the default shallow-clone depth (0 = full clone). The -depth
	// flag overrides it.
	Depth int `json:"depth"`
	// NoBuild disables the automatic build after cloning unless -no-build is
	// passed explicitly.
	NoBuild bool `json:"no_build"`
}

func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gofetch", "config.json"), nil
}

// LoadConfig reads the user's config file. Any problem (missing file, invalid
// JSON, unknown location) degrades to the zero Config instead of failing: the
// app must keep working with defaults.
func LoadConfig() Config {
	filePath, err := configPath()
	if err != nil {
		return Config{}
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}
		}
		return Config{}
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil {
		return Config{}
	}
	return cfg.sanitized()
}

// sanitized normalizes values that would otherwise cause confusing behavior:
// a negative depth is treated as "full clone" and the directory is expanded.
func (c Config) sanitized() Config {
	if c.Depth < 0 {
		c.Depth = 0
	}
	c.DefaultDir = strings.TrimSpace(c.DefaultDir)
	return c
}

// SaveConfig writes the config atomically, reusing the same temp-file +
// rename strategy as the history so a crash cannot corrupt preferences.
func SaveConfig(cfg Config) error {
	filePath, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg.sanitized(), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, filePath)
}
