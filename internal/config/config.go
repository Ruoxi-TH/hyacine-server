package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type CORSConfig struct {
	Enabled bool     `json:"enabled"`
	Origins []string `json:"origins"`
}

type StreamConfig struct {
	BufferSize int `json:"buffer_size"`
	Timeout    int `json:"timeout"`
}

type DatabaseConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type SMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	From       string `json:"from"`
	Encryption string `json:"encryption"`
}

type JWTConfig struct {
	Secret     string `json:"secret"`
	AccessTTL  string `json:"access_ttl"`
	RefreshTTL string `json:"refresh_ttl"`
}

type FileConfig struct {
	Port           string         `json:"port"`
	NeteaseAPIBase string         `json:"netease_api_base"`
	LogLevel       string         `json:"log_level"`
	CORS           CORSConfig     `json:"cors"`
	Stream         StreamConfig   `json:"stream"`
	Database       DatabaseConfig `json:"database"`
	SMTP           SMTPConfig     `json:"smtp"`
	JWT            JWTConfig      `json:"jwt"`
}

func Load() (*FileConfig, error) {
	cfg, err := loadFromFile()
	if err != nil {
		cfg = &FileConfig{
			Port:           "3000",
			NeteaseAPIBase: strings.TrimRight(strings.TrimSpace(os.Getenv("NETEASE_API_BASE")), "/"),
			LogLevel:       "info",
			CORS:           CORSConfig{Enabled: true, Origins: []string{"*"}},
			Stream:         StreamConfig{BufferSize: 32768, Timeout: 30},
			Database:       DatabaseConfig{Type: "sqlite", Path: "./data/hyacine.db"},
			SMTP:           SMTPConfig{Port: 587, From: "风堇音乐 <noreply@example.com>"},
			JWT:            JWTConfig{Secret: "change-this-secret-in-production", AccessTTL: "168h", RefreshTTL: "720h"},
		}
	}

	if envPort := strings.TrimSpace(os.Getenv("PORT")); envPort != "" {
		cfg.Port = envPort
	}
	if envAPI := strings.TrimSpace(os.Getenv("NETEASE_API_BASE")); envAPI != "" {
		cfg.NeteaseAPIBase = strings.TrimRight(envAPI, "/")
	}
	if cfg.Port == "" {
		cfg.Port = "3000"
	}
	cfg.NeteaseAPIBase = strings.TrimRight(cfg.NeteaseAPIBase, "/")

	return cfg, nil
}

func loadFromFile() (*FileConfig, error) {
	configPaths := []string{
		"./config.json",
		"/etc/hyacine/config.json",
	}

	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	configPaths = append(configPaths,
		filepath.Join(execDir, "config.json"),
	)

	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var cfg FileConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}

		return &cfg, nil
	}

	return nil, os.ErrNotExist
}
