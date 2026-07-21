package main

import (
	"encoding/json"
	"log"
	"os"

	"hyacine-go-server/internal/config"
)

var version = "dev"

// 默认配置
var defaultConfig = config.FileConfig{
	Port:           "3000",
	NeteaseAPIBase: "",
	LogLevel:       "info",
	CORS: config.CORSConfig{
		Enabled: true,
		Origins: []string{"*"},
	},
	Stream: config.StreamConfig{
		BufferSize: 32768,
		Timeout:    30,
	},
}

func ensureConfig() {
	configPaths := []string{"./config.json", "/etc/hyacine/config.json"}
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return // 配置文件已存在
		}
	}

	// 创建默认配置文件
	path := "./config.json"
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal default config: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Warning: failed to create default config: %v", err)
		return
	}

	log.Printf("Created default config file: %s", path)
}

func main() {
	log.Printf("Hyacine Server %s", version)
	
	// 确保配置文件存在
	ensureConfig()
	
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(httpapi.ListenAndServe(cfg))
}
