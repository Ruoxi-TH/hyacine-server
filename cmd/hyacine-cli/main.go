package main

import (
	"fmt"
	"hyacine-go-server/internal/auth"
	"hyacine-go-server/internal/store"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	dataDir := os.Getenv("HYACINE_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	db, err := store.New(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	command := os.Args[1]
	switch command {
	case "status":
		statusCmd(db)
	case "users":
		usersCmd(db)
	case "promote":
		promoteCmd(db)
	case "ban":
		banCmd(db)
	case "unban":
		unbanCmd(db)
	case "delete":
		deleteCmd(db)
	case "create-admin":
		createAdminCmd(db)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Hyacine Server CLI")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  status              Show system status")
	fmt.Println("  users               List all users")
	fmt.Println("  promote <user_id>   Promote user to admin")
	fmt.Println("  ban <user_id> [reason]   Ban user")
	fmt.Println("  unban <user_id>     Unban user")
	fmt.Println("  delete <user_id>    Delete user")
	fmt.Println("  create-admin <username> <email> <password>   Create admin account")
	fmt.Println("  help                Show this help")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  HYACINE_DATA_DIR    Data directory (default: ./data)")
}

func statusCmd(db *store.Store) {
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

func usersCmd(db *store.Store) {
	page := 1
	limit := 50
	offset := (page - 1) * limit

	users, total, err := db.ListUsers(offset, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list users: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Users (Total: %d) ===\n", total)
	fmt.Println()
	fmt.Printf("%-6s %-20s %-30s %-8s %-6s\n", "ID", "Username", "Email", "Role", "Banned")
	fmt.Println(strings.Repeat("-", 80))

	for _, u := range users {
		banned := "No"
		if u.Banned {
			banned = "Yes"
		}
		fmt.Printf("%-6d %-20s %-30s %-8s %-6s\n", u.ID, u.Username, u.Email, u.Role, banned)
	}

	if len(users) == limit && total > limit {
		fmt.Printf("\n... and %d more users\n", total-limit)
	}
}

func promoteCmd(db *store.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: promote <user_id>")
		os.Exit(1)
	}

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

func banCmd(db *store.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ban <user_id> [reason]")
		os.Exit(1)
	}

	userID, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid user ID: %v\n", err)
		os.Exit(1)
	}

	reason := "违反社区规范"
	if len(os.Args) > 3 {
		reason = strings.Join(os.Args[3:], " ")
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "User not found: %v\n", err)
		os.Exit(1)
	}

	if user.Role == "admin" {
		fmt.Fprintln(os.Stderr, "Cannot ban admin users")
		os.Exit(1)
	}

	if err := db.BanUser(userID, reason); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ban user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %s (ID: %d) banned: %s\n", user.Username, userID, reason)
}

func unbanCmd(db *store.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: unban <user_id>")
		os.Exit(1)
	}

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

	if err := db.UnbanUser(userID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unban user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %s (ID: %d) unbanned\n", user.Username, userID)
}

func deleteCmd(db *store.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: delete <user_id>")
		os.Exit(1)
	}

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
		fmt.Fprintln(os.Stderr, "Cannot delete admin users")
		os.Exit(1)
	}

	if err := db.DeleteUser(userID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %s (ID: %d) deleted\n", user.Username, userID)
}

func createAdminCmd(db *store.Store) {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: create-admin <username> <email> <password>")
		os.Exit(1)
	}

	username := strings.TrimSpace(os.Args[2])
	email := strings.ToLower(strings.TrimSpace(os.Args[3]))
	password := os.Args[4]

	if !auth.ValidateUsername(username) {
		fmt.Fprintln(os.Stderr, "Invalid username (2-20 chars, letters/numbers/underscores/Chinese)")
		os.Exit(1)
	}

	if !auth.ValidatePassword(password) {
		fmt.Fprintln(os.Stderr, "Invalid password (8+ chars with letters and numbers)")
		os.Exit(1)
	}

	if _, err := db.GetUserByEmail(email); err == nil {
		fmt.Fprintf(os.Stderr, "Email already registered: %s\n", email)
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

	user, err := db.CreateUser(username, email, hashedPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create user: %v\n", err)
		os.Exit(1)
	}

	if err := db.UpdateUserRole(user.ID, "admin"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to promote to admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin account created: %s (ID: %d, Email: %s)\n", username, user.ID, email)
}