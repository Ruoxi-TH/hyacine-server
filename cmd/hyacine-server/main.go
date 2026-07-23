package main

import (
	"encoding/json"
	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/email"
	"hyacine-go-server/internal/httpapi"
	"hyacine-go-server/internal/store"
	"log"
	"os"
)

var version = "dev"

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
	Database: config.DatabaseConfig{
		Type: "sqlite",
		Path: "./data/hyacine.db",
	},
	SMTP: config.SMTPConfig{
		Host:     "",
		Port:     587,
		User:     "",
		Password: "",
		From:     "风堇音乐 <noreply@example.com>",
	},
	JWT: config.JWTConfig{
		Secret:      "change-this-secret-in-production",
		AccessTTL:   "168h",
		RefreshTTL:  "720h",
	},
}

func ensureConfig() {
	configPaths := []string{"./config.json", "/etc/hyacine/config.json"}
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return
		}
	}

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

	ensureConfig()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	fileCfg, err := config.LoadFileConfig()
	if err != nil {
		log.Printf("Warning: failed to load file config: %v, using defaults", err)
		fileCfg = &defaultConfig
	}

	dataDir := "./data"
	if fileCfg.Database.Path != "" {
		dataDir = fileCfg.Database.Path
	}

	db, err := store.New(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at %s", dataDir)

	smtpCfg := email.SMTPConfig{
		Host:     fileCfg.SMTP.Host,
		Port:     fileCfg.SMTP.Port,
		User:     fileCfg.SMTP.User,
		Password: fileCfg.SMTP.Password,
		From:     fileCfg.SMTP.From,
	}

	jwtSecret := fileCfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "change-this-secret-in-production"
	}

	log.Fatal(httpapi.ListenAndServe(cfg, db, smtpCfg, jwtSecret))
}