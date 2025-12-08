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
}

func DefaultConfig() *Config {
	return &Config{
		Theme:       "auto",
		Model:       "sonnet",
		Style:       "",
		LineNumbers: nil,
	}
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hnk")
}

func Load() *Config {
	cfg := DefaultConfig()
	path := configPath()
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	json.Unmarshal(data, cfg)
	return cfg
}

func (c *Config) Save() error {
	path := configPath()
	if path == "" {
		return nil
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
