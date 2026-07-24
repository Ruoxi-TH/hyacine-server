package main

import (
	"encoding/json"
	"fmt"
	"hyacine-go-server/internal/auth"
	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/email"
	"hyacine-go-server/internal/httpapi"
	"hyacine-go-server/internal/store"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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

func main() {
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "create-admin":
			cliCreateAdmin()
			return
		case "promote":
			cliPromote()
			return
		case "status":
			cliStatus()
			return
		case "help", "--help", "-h":
			printCLIUsage()
			return
		}
	}

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
		Host:       fileCfg.SMTP.Host,
		Port:       fileCfg.SMTP.Port,
		Username:   fileCfg.SMTP.User,
		Password:   fileCfg.SMTP.Password,
		FromName:   fileCfg.SMTP.From,
		Encryption: email.EncryptionMode(fileCfg.SMTP.Encryption),
	}

	jwtSecret := fileCfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "change-this-secret-in-production"
	}

	log.Fatal(httpapi.ListenAndServe(cfg, db, smtpCfg, jwtSecret))
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

func getCLIDB() *store.Store {
	dataDir := os.Getenv("HYACINE_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	db, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	return db
}

func printCLIUsage() {
	fmt.Println("Hyacine Server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hyacine-server                       Start the server")
	fmt.Println("  hyacine-server create-admin <username> <email> <password>   Create admin")
	fmt.Println("  hyacine-server promote <user_id>     Promote user to admin")
	fmt.Println("  hyacine-server status                Show system status")
	fmt.Println("  hyacine-server help                  Show this help")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  HYACINE_DATA_DIR    Data directory (default: ./data)")
}

func cliCreateAdmin() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: hyacine-server create-admin <username> <email> <password>")
		os.Exit(1)
	}

	db := getCLIDB()
	defer db.Close()

	username := strings.TrimSpace(os.Args[2])
	emailAddr := strings.ToLower(strings.TrimSpace(os.Args[3]))
	password := os.Args[4]

	if !auth.ValidateUsername(username) {
		fmt.Fprintln(os.Stderr, "Invalid username (2-20 chars, letters/numbers/underscores/Chinese)")
		os.Exit(1)
	}

	if !auth.ValidatePassword(password) {
		fmt.Fprintln(os.Stderr, "Invalid password (8+ chars with letters and numbers)")
		os.Exit(1)
	}

	if _, err := db.GetUserByEmail(emailAddr); err == nil {
		fmt.Fprintf(os.Stderr, "Email already registered: %s\n", emailAddr)
		os.Exit(1)
	}

	if _, err := db.GetUserByUsername(username); err == nil {
		fmt.Fprintf(os.Stderr, "Username already taken: %s\n", username)
		os.Exit(1)
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	user, err := db.CreateUser(username, emailAddr, hashedPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create user: %v\n", err)
		os.Exit(1)
	}

	if err := db.UpdateUserRole(user.ID, "admin"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to promote to admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin account created: %s (ID: %d, Email: %s)\n", username, user.ID, emailAddr)
}

func cliPromote() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: hyacine-server promote <user_id>")
		os.Exit(1)
	}

	db := getCLIDB()
	defer db.Close()

	userID, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid user ID: %v\n", err)
		os.Exit(1)
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "User not found: %v\n", err)
		os.Exit(1)
	}

	if user.Role == "admin" {
		fmt.Printf("User %s is already an admin\n", user.Username)
		return
	}

	if err := db.UpdateUserRole(userID, "admin"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to promote user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %s (ID: %d) promoted to admin\n", user.Username, userID)
}

func cliStatus() {
	db := getCLIDB()
	defer db.Close()

	stats, err := db.Stats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Hyacine Server Status ===")
	fmt.Printf("Total Users:  %d\n", stats["users"])
	fmt.Printf("Admins:       %d\n", stats["admins"])
	fmt.Printf("Banned:       %d\n", stats["banned"])
	fmt.Println()
	fmt.Printf("Server Time:  %s\n", time.Now().Format(time.RFC3339))
}
