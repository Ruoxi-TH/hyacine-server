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

type FileConfig struct {
	Port           string       `json:"port"`
	NeteaseAPIBase string       `json:"netease_api_base"`
	LogLevel       string       `json:"log_level"`
	CORS           CORSConfig   `json:"cors"`
	Stream         StreamConfig `json:"stream"`
}

func Load() (Config, error) {
	// 先尝试从配置文件加载
	fileCfg, err := loadFromFile()
	if err == nil {
		cfg := Config{
			Port:           fileCfg.Port,
			NeteaseAPIBase: strings.TrimRight(fileCfg.NeteaseAPIBase, "/"),
		}
		// 环境变量优先级更高
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

	// 回退到环境变量
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
	// 按优先级查找配置文件
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

// LoadFileConfig 加载完整配置（包含 CORS、Stream 等高级选项）
func LoadFileConfig() (*FileConfig, error) {
	return loadFromFile()
}