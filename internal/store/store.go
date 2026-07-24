package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type User struct {
	ID        int64
	Username  string
	Email     string
	Password  string
	Role      string
	Banned    bool
	BanReason string
	CreatedAt string
	UpdatedAt string
}

type EmailCode struct {
	ID        int64
	Email     string
	Code      string
	ExpiresAt string
	Used      bool
	CreatedAt string
}

type LoginLog struct {
	ID        int64
	UserID    int64
	IP        string
	UserAgent string
	CreatedAt string
}

type Store struct {
	db *sql.DB
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "hyacine.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateLoginLog(userID int64, ip, userAgent string) error {
	_, err := s.db.Exec(
		"INSERT INTO login_logs (user_id, ip, user_agent) VALUES (?, ?, ?)",
		userID, ip, userAgent,
	)
	return err
}

func (s *Store) ListLoginLogs(userID int64, limit int) ([]LoginLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		"SELECT id, user_id, ip, user_agent, created_at FROM login_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []LoginLog
	for rows.Next() {
		var l LoginLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.IP, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *Store) UpdateUserLastLogin(id int64) error {
	_, err := s.db.Exec("UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			banned INTEGER NOT NULL DEFAULT 0,
			ban_reason TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

		CREATE TABLE IF NOT EXISTS email_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			code TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_email_codes_email ON email_codes(email);

		CREATE TABLE IF NOT EXISTS login_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE INDEX IF NOT EXISTS idx_login_logs_user_id ON login_logs(user_id);
		CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs(created_at);
	`)
	return err
}
