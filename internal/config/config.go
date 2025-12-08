package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Theme       string `json:"theme"`
	Model       string `json:"model"`
	Style       string `json:"style"`
	LineNumbers *bool  `json:"line_numbers,omitempty"`
	CacheSizeMB int    `json:"cache_size_mb,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Theme:       "auto",
		Model:       "sonnet",
		Style:       "",
		LineNumbers: nil,
		CacheSizeMB: 5,
	}
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hnk")
}

func configPath() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config.json")
}

func Load() *Config {
	cfg := DefaultConfig()
	dir := configDir()
	if dir == "" {
		return cfg
	}

	oldPath := dir
	newPath := filepath.Join(dir, "config.json")

	if info, err := os.Stat(oldPath); err == nil && !info.IsDir() {
		data, err := os.ReadFile(oldPath)
		if err == nil {
			os.MkdirAll(dir+".tmp", 0755)
			os.Rename(dir+".tmp", dir)
			os.MkdirAll(dir, 0755)
			os.WriteFile(newPath, data, 0644)
			os.Remove(oldPath)
		}
	}

	os.MkdirAll(dir, 0755)

	data, err := os.ReadFile(newPath)
	if err != nil {
		return cfg
	}

	json.Unmarshal(data, cfg)
	if cfg.CacheSizeMB <= 0 {
		cfg.CacheSizeMB = 5
	}
	return cfg
}

func (c *Config) Save() error {
	path := configPath()
	if path == "" {
		return nil
	}

	os.MkdirAll(filepath.Dir(path), 0755)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) CacheSizeBytes() int {
	if c.CacheSizeMB <= 0 {
		return 5 * 1024 * 1024
	}
	return c.CacheSizeMB * 1024 * 1024
}
