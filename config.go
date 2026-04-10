package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ─────────────────────────────────────────────
//  Domain types
// ─────────────────────────────────────────────

type Project struct {
	Name        string   `json:"name"`
	Port        int      `json:"port"`
	AllowedMACs []string `json:"allowed_macs,omitempty"`
	Command     string   `json:"command,omitempty"`
	Dir         string   `json:"dir,omitempty"`
}

type Config struct {
	Projects  []Project `json:"projects"`
	ProxyPort int       `json:"proxy_port"`
	AdminPort int       `json:"admin_port"`
	LogFile   string    `json:"log_file"`
}

// ─────────────────────────────────────────────
//  Load / save
// ─────────────────────────────────────────────

func defaultConfig() Config {
	return Config{
		ProxyPort: 80,
		AdminPort: 9090,
		Projects: []Project{
			{Name: "myapp", Port: 3000, Command: "npm run dev"},
			{Name: "api", Port: 8080, Command: "go run main.go"},
		},
	}
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid config JSON: %w", err)
	}
	if cfg.ProxyPort == 0 {
		cfg.ProxyPort = 80
	}
	if cfg.AdminPort == 0 {
		cfg.AdminPort = 9090
	}
	return cfg, nil
}

func saveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
