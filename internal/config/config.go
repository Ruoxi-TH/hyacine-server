package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port           string `json:"port"`
	NeteaseAPIBase string `json:"netease_api_base"`
	LogLevel       string `json:"log_level"`
}

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
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type JWTConfig struct {
	Secret      string `json:"secret"`
	AccessTTL   string `json:"access_ttl"`
	RefreshTTL  string `json:"refresh_ttl"`
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

func Load() (Config, error) {
	fileCfg, err := loadFromFile()
	if err == nil {
		cfg := Config{
			Port:           fileCfg.Port,
			NeteaseAPIBase: strings.TrimRight(fileCfg.NeteaseAPIBase, "/"),
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
		return cfg, nil
	}

	cfg := Config{
		Port:           strings.TrimSpace(os.Getenv("PORT")),
		NeteaseAPIBase: strings.TrimRight(strings.TrimSpace(os.Getenv("NETEASE_API_BASE")), "/"),
	}
	if cfg.Port == "" {
		cfg.Port = "3000"
	}
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

func LoadFileConfig() (*FileConfig, error) {
	return loadFromFile()
}
