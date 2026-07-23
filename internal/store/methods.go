package store

import (
	"time"
)

func (s *Store) CreateUser(username, email, hashedPassword string) (*User, error) {
	result, err := s.db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password, role, banned, COALESCE(ban_reason, ''), created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.Banned, &u.BanReason, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password, role, banned, COALESCE(ban_reason, ''), created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.Banned, &u.BanReason, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password, role, banned, COALESCE(ban_reason, ''), created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.Banned, &u.BanReason, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) ListUsers(offset, limit int) ([]User, int, error) {
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(
		"SELECT id, username, email, password, role, banned, COALESCE(ban_reason, ''), created_at, updated_at FROM users ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u := User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.Banned, &u.BanReason, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (s *Store) UpdateUserRole(id int64, role string) error {
	_, err := s.db.Exec("UPDATE users SET role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", role, id)
	return err
}

func (s *Store) BanUser(id int64, reason string) error {
	_, err := s.db.Exec("UPDATE users SET banned = 1, ban_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", reason, id)
	return err
}

func (s *Store) UnbanUser(id int64) error {
	_, err := s.db.Exec("UPDATE users SET banned = 0, ban_reason = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *Store) CreateEmailCode(email, code string, expiresAt time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO email_codes (email, code, expires_at) VALUES (?, ?, ?)",
		email, code, expiresAt,
	)
	return err
}

func (s *Store) GetValidEmailCode(email, code string) (*EmailCode, error) {
	ec := &EmailCode{}
	err := s.db.QueryRow(
		"SELECT id, email, code, expires_at, used, created_at FROM email_codes WHERE email = ? AND code = ? AND used = 0 AND expires_at > ?",
		email, code, time.Now(),
	).Scan(&ec.ID, &ec.Email, &ec.Code, &ec.ExpiresAt, &ec.Used, &ec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func (s *Store) MarkEmailCodeUsed(id int64) error {
	_, err := s.db.Exec("UPDATE email_codes SET used = 1 WHERE id = ?", id)
	return err
}

func (s *Store) CountRecentEmailCodes(email string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM email_codes WHERE email = ? AND created_at > ?",
		email, since,
	).Scan(&count)
	return count, err
}

func (s *Store) CleanupExpired() error {
	now := time.Now()
	_, err := s.db.Exec("DELETE FROM email_codes WHERE expires_at < ?", now)
	return err
}

func (s *Store) Stats() (map[string]int64, error) {
	stats := make(map[string]int64)
	var userCount, adminCount, bannedCount int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return nil, err
	}
	stats["users"] = userCount
	err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount)
	if err != nil {
		return nil, err
	}
	stats["admins"] = adminCount
	err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE banned = 1").Scan(&bannedCount)
	if err != nil {
		return nil, err
	}
	stats["banned"] = bannedCount
	return stats, nil
}
