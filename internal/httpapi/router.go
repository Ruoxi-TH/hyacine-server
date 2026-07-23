package httpapi

import (
	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/email"
	"hyacine-go-server/internal/music/netease"
	"hyacine-go-server/internal/store"
	"hyacine-go-server/internal/stream"
	"log"
	"net/http"
	"time"
)

type App struct {
	cfg           config.Config
	store         Store
	authHandler   *AuthHandler
	adminHandler  *AdminHandler
	netease       netease.Client
	directNetease *netease.DirectClient
	client        *http.Client
	streams       *stream.Store
}

type Store interface {
	CreateUser(username, email, hashedPassword string) (*store.User, error)
	GetUserByID(id int64) (*store.User, error)
	GetUserByEmail(email string) (*store.User, error)
	GetUserByUsername(username string) (*store.User, error)
	ListUsers(offset, limit int) ([]store.User, int, error)
	UpdateUserRole(id int64, role string) error
	BanUser(id int64, reason string) error
	UnbanUser(id int64) error
	DeleteUser(id int64) error
	CreateEmailCode(email, code string, expiresAt time.Time) error
	GetValidEmailCode(email, code string) (*store.EmailCode, error)
	MarkEmailCodeUsed(id int64) error
	CountRecentEmailCodes(email string, since time.Time) (int, error)
	CleanupExpired() error
	Stats() (map[string]int64, error)
}

func ListenAndServe(cfg config.Config, store Store, smtpCfg email.SMTPConfig, jwtSecret string) error {
	log.Printf("Hyacine Go server listening on :%s", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, NewRouter(cfg, store, smtpCfg, jwtSecret))
}

func NewRouter(cfg config.Config, store Store, smtpCfg email.SMTPConfig, jwtSecret string) http.Handler {
	app := &App{
		cfg:          cfg,
		store:        store,
		client:       &http.Client{Timeout: 20 * time.Second},
		streams:      stream.NewStore(15 * time.Minute),
		authHandler:  NewAuthHandler(store, smtpCfg, jwtSecret),
		adminHandler: NewAdminHandler(store, nil),
	}
	
	if cfg.NeteaseAPIBase == "" {
		app.directNetease = netease.NewDirectClient(15 * time.Second)
	} else {
		app.netease = netease.NewHTTPClient(cfg.NeteaseAPIBase, 10*time.Second)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", app.health)

	mux.HandleFunc("/api/v1/auth/send-code", app.authHandler.SendCode)
	mux.HandleFunc("/api/v1/auth/register", app.authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", app.authHandler.Login)
	mux.HandleFunc("/api/v1/auth/me", app.authHandler.Me)

	mux.HandleFunc("/api/v1/admin/users", app.adminHandler.ListUsers)
	mux.HandleFunc("/api/v1/admin/users/ban", app.adminHandler.BanUser)
	mux.HandleFunc("/api/v1/admin/users/unban", app.adminHandler.UnbanUser)
	mux.HandleFunc("/api/v1/admin/users/delete", app.adminHandler.DeleteUser)
	mux.HandleFunc("/api/v1/admin/users/promote", app.adminHandler.PromoteUser)
	mux.HandleFunc("/api/v1/admin/stats", app.adminHandler.Stats)

	mux.HandleFunc("/api/v1/music-sources/netease/qr", app.neteaseQR)
	mux.HandleFunc("/api/v1/music-sources/netease/qr/", app.neteaseQRPoll)
	mux.HandleFunc("/api/v1/music-sources/netease/profile", app.neteaseProfile)
	mux.HandleFunc("/api/v1/music-sources/netease/recommendations", app.neteaseRecommendations)
	mux.HandleFunc("/api/v1/music-sources/netease/daily-songs", app.neteaseDailySongs)
	mux.HandleFunc("/api/v1/music-sources/netease/playlists", app.neteasePlaylists)
	mux.HandleFunc("/api/v1/music-sources/netease/playlists/detail", app.neteasePlaylistDetail)
	mux.HandleFunc("/api/v1/music-sources/netease/playlists/create", app.neteaseCreatePlaylist)
	mux.HandleFunc("/api/v1/music-sources/netease/playlists/delete", app.neteaseDeletePlaylist)
	mux.HandleFunc("/api/v1/music-sources/netease/favorites/toggle", app.neteaseToggleFavorite)
	mux.HandleFunc("/api/v1/music-sources/netease/search", app.neteaseSearch)
	mux.HandleFunc("/api/v1/music-sources/netease/play-url", app.neteasePlayURL)
	mux.HandleFunc("/api/v1/music-sources/netease/lyrics", app.neteaseLyrics)
	mux.HandleFunc("/api/v1/music-sources/netease/comments", app.neteaseComments)
	mux.HandleFunc("/api/v1/music-sources/netease/stream/", app.neteaseStream)
	mux.HandleFunc("/api/v1/music-sources/bilibili/validate-cookie", app.bilibiliValidateCookie)
	mux.HandleFunc("/api/v1/music-sources/bilibili/search", app.bilibiliSearch)
	mux.HandleFunc("/api/v1/music-sources/bilibili/play-url", app.bilibiliPlayURL)
	mux.HandleFunc("/api/v1/music-sources/bilibili/stream/", app.bilibiliStream)

	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("request panic recovered: %v", recovered)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal server error"})
			}
		}()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
