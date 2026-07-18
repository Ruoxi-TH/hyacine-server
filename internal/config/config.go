package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	NeteaseAPIBase string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           strings.TrimSpace(os.Getenv("PORT")),
		NeteaseAPIBase: strings.TrimRight(strings.TrimSpace(os.Getenv("NETEASE_API_BASE")), "/"),
	}
	if cfg.Port == "" {
		cfg.Port = "3000"
	}
	return cfg, nil
}
