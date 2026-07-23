package httpapi

import (
	"encoding/json"
	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/store"
	"net/http"
	"strconv"
	"strings"
)

type AdminHandler struct {
	store *store.Store
	email EmailSender
}

func NewAdminHandler(s *store.Store, email EmailSender) *AdminHandler {
	return &AdminHandler{store: s, email: email}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	users, total, err := h.store.ListUsers(offset, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to list users"})
		return
	}

	var sanitized []UserResponse
	for _, u := range users {
		sanitized = append(sanitized, sanitizeUser(&u))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": sanitized,
		"total": total,
		"page":  page,
	})
}

type BanRequest struct {
	UserID int64  `json:"user_id"`
	Reason string `json:"reason"`
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req BanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "user_id is required"})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "违反社区规范"
	}

	user, err := h.store.GetUserByID(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		return
	}

	if user.Role == "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "cannot ban admin users"})
		return
	}

	if err := h.store.BanUser(req.UserID, reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to ban user"})
		return
	}

	h.email.SendBanNotification(user.Email, reason)

	writeJSON(w, http.StatusOK, map[string]string{"message": "user banned successfully"})
}

func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "user_id is required"})
		return
	}

	user, err := h.store.GetUserByID(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		return
	}

	if err := h.store.UnbanUser(req.UserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to unban user"})
		return
	}

	h.email.SendUnbanNotification(user.Email)

	writeJSON(w, http.StatusOK, map[string]string{"message": "user unbanned successfully"})
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "user_id is required"})
		return
	}

	user, err := h.store.GetUserByID(req.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		return
	}

	if user.Role == "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "cannot delete admin users"})
		return
	}

	if err := h.store.DeleteUser(req.UserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to delete user"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted successfully"})
}

type PromoteRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (h *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req PromoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "user_id is required"})
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "role must be 'user' or 'admin'"})
		return
	}

	if _, err := h.store.GetUserByID(req.UserID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		return
	}

	if err := h.store.UpdateUserRole(req.UserID, req.Role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to update role"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user role updated successfully"})
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	stats, err := h.store.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	cfg, err := config.LoadFileConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to load config"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"smtp": map[string]interface{}{
			"host":     cfg.SMTP.Host,
			"port":     cfg.SMTP.Port,
			"user":     cfg.SMTP.User,
			"from":     cfg.SMTP.From,
			"has_password": cfg.SMTP.Password != "",
		},
	})
}

func (h *AdminHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req struct {
		SMTP struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			From     string `json:"from"`
		} `json:"smtp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	cfg, err := config.LoadFileConfig()
	if err != nil {
		cfg = &config.FileConfig{
			Port:     "3000",
			LogLevel: "info",
		}
	}

	cfg.SMTP.Host = req.SMTP.Host
	cfg.SMTP.Port = req.SMTP.Port
	cfg.SMTP.User = req.SMTP.User
	cfg.SMTP.From = req.SMTP.From
	if req.SMTP.Password != "" {
		cfg.SMTP.Password = req.SMTP.Password
	}

	if err := config.SaveFileConfig(cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to save config"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "config updated"})
}

func IsAdmin(user *store.User) bool {
	return user != nil && user.Role == "admin"
}
