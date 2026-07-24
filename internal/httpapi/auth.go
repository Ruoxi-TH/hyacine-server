package httpapi

import (
	"encoding/json"
	"errors"
	"hyacine-go-server/internal/auth"
	"hyacine-go-server/internal/email"
	"hyacine-go-server/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

type EmailSender interface {
	SendVerificationCode(to, code string) error
	SendBanNotification(to, reason string) error
	SendUnbanNotification(to string) error
}

type AuthHandler struct {
	store       Store
	emailSender EmailSender
	jwtSecret   string
}

func NewAuthHandler(s Store, emailCfg email.SMTPConfig, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		store:       s,
		emailSender: email.NewSender(emailCfg),
		jwtSecret:   jwtSecret,
	}
}

func (h *AuthHandler) EmailSender() EmailSender {
	return h.emailSender
}

type SendCodeRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req SendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	email := auth.NormalizeEmail(req.Email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "email is required"})
		return
	}

	count, err := h.store.CountRecentEmailCodes(email, time.Now().Add(-24*time.Hour))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to check rate limit"})
		return
	}
	if count >= 10 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "too many requests, please try again tomorrow"})
		return
	}

	count, err = h.store.CountRecentEmailCodes(email, time.Now().Add(-60*time.Second))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to check rate limit"})
		return
	}
	if count >= 1 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "please wait 60 seconds before requesting another code"})
		return
	}

	code, err := auth.GenerateCode(6)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to generate code"})
		return
	}

	if err := h.store.CreateEmailCode(email, code, time.Now().Add(5*time.Minute)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to store code"})
		return
	}

	if err := h.emailSender.SendVerificationCode(email, code); err != nil {
		log.Printf("[SMTP] Failed to send verification code to %s: %v", email, err)
		if delErr := h.store.DeleteEmailCode(email, code); delErr != nil {
			log.Printf("[SMTP] Failed to rollback email code for %s: %v", email, delErr)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to send email"})
		return
	}

	log.Printf("[SMTP] Verification code sent to %s", email)
	writeJSON(w, http.StatusOK, map[string]string{"message": "verification code sent"})
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	username := strings.TrimSpace(req.Username)
	email := auth.NormalizeEmail(req.Email)
	password := req.Password
	code := req.Code

	if !auth.ValidateUsername(username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "username must be 2-20 characters, containing only letters, numbers, underscores, or Chinese characters"})
		return
	}

	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "email is required"})
		return
	}

	if !auth.ValidatePassword(password) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "password must be at least 8 characters and contain both letters and numbers"})
		return
	}

	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "verification code is required"})
		return
	}

	emailCode, err := h.store.GetValidEmailCode(email, code)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid or expired verification code"})
		return
	}

	if _, err := h.store.GetUserByEmail(email); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "email already registered"})
		return
	}

	if _, err := h.store.GetUserByUsername(username); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "username already taken"})
		return
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to hash password"})
		return
	}

	user, err := h.store.CreateUser(username, email, hashedPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to create user"})
		return
	}

	h.store.MarkEmailCodeUsed(emailCode.ID)

	token, err := auth.GenerateAccessToken(user.ID, user.Username, user.Role, h.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":  "registration successful",
		"user":     sanitizeUser(user),
		"token":    token,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	email := auth.NormalizeEmail(req.Email)
	password := req.Password

	if email == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "email and password are required"})
		return
	}

	user, err := h.store.GetUserByEmail(email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "invalid email or password"})
		return
	}

	if user.Banned {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "account is banned: " + user.BanReason})
		return
	}

	if !auth.CheckPassword(password, user.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "invalid email or password"})
		return
	}

	token, err := auth.GenerateAccessToken(user.ID, user.Username, user.Role, h.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "login successful",
		"user":    sanitizeUser(user),
		"token":   token,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
		return
	}

	user, err := h.getUserFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, sanitizeUser(user))
}

func (h *AuthHandler) getUserFromRequest(r *http.Request) (*store.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("no authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.New("invalid authorization header format")
	}

	claims, err := auth.ParseToken(parts[1], h.jwtSecret)
	if err != nil {
		return nil, err
	}

	return h.store.GetUserByID(claims.UserID)
}

type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Banned    bool   `json:"banned"`
	BanReason string `json:"ban_reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

func sanitizeUser(u *store.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Banned:    u.Banned,
		BanReason: u.BanReason,
		CreatedAt: u.CreatedAt,
	}
}
