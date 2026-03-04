package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds optional overrides from ~/.dotclaude/config.json
type Config struct {
	Projects map[string]string `json:"projects"` // alias → absolute path
}

func loadConfig() Config {
	cfg := Config{
		Projects: make(map[string]string),
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	configPath := filepath.Join(home, ".dotclaude", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	json.Unmarshal(data, &cfg)
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]string)
	}
	return cfg
}
