package app

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cdk-airdrop-station/server/internal/api"
	"cdk-airdrop-station/server/internal/config"
	"cdk-airdrop-station/server/internal/hcaptcha"
	"cdk-airdrop-station/server/internal/store"
)

func New(cfg config.Config, stateStore *store.Store, logger *slog.Logger) *http.Server {
	handler := api.New(stateStore, logger)
	handler.SetHCaptchaVerifier(hcaptcha.New(cfg.HCaptchaSecret, cfg.HCaptchaVerifyURL, cfg.HCaptchaSiteKey))
	handler.SetCommunityConfig(cfg.CommunityURL, cfg.CommunityClientID)
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/api/admin/health", handler.AdminHealth)
	mux.HandleFunc("/api/admin/onboarding/status", handler.AdminOnboardingStatus)
	mux.HandleFunc("/api/admin/dashboard", handler.AdminDashboard)

	// 管理端 API
	mux.HandleFunc("/api/admin/projects", handler.AdminProjectsRouter)
	mux.HandleFunc("/api/admin/projects/", handler.AdminProjectDetailRouter)
	mux.HandleFunc("/api/admin/campaigns", handler.AdminCampaignsRouter)
	mux.HandleFunc("/api/admin/campaigns/", handler.AdminCampaignDetailRouter)
	mux.HandleFunc("/api/admin/cdks", handler.AdminCDKsRouter)
	mux.HandleFunc("/api/admin/cdks/", handler.AdminCDKDetailRouter)
	mux.HandleFunc("/api/admin/nodes", handler.AdminNodesRouter)
	mux.HandleFunc("/api/admin/nodes/", handler.AdminNodeDetailRouter)
	mux.HandleFunc("/api/admin/claims", handler.AdminClaimsRouter)
	mux.HandleFunc("/api/admin/claims/", handler.AdminClaimsRouter)
	mux.HandleFunc("/api/admin/analytics/overview", handler.AdminAnalyticsRouter)
	mux.HandleFunc("/api/admin/analytics/", handler.AdminAnalyticsRouter)
	mux.HandleFunc("/api/admin/logs", handler.AdminLogsRouter)
	mux.HandleFunc("/api/admin/logs/", handler.AdminLogsRouter)
	mux.HandleFunc("/api/admin/risk/", handler.AdminRiskRouter)
	mux.HandleFunc("/api/admin/captcha/", handler.AdminCaptchaRouter)
	mux.HandleFunc("/api/admin/settings", handler.AdminSettingsRouter)
	mux.HandleFunc("/api/admin/export-tasks", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/backup", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/restore", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/admins", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/admins/", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/exports/", handler.AdminSystemRouter)
	mux.HandleFunc("/api/admin/backups/", handler.AdminSystemRouter)

	// 公开领取 API
	mux.HandleFunc("/api/claim", handler.ClaimRouter)
	mux.HandleFunc("/api/public/claim/", handler.PublicClaimRouter)

	// 用户与认证 API
	mux.HandleFunc("/api/me", handler.GetMe)
	mux.HandleFunc("/api/auth/login", handler.AuthLogin)
	mux.HandleFunc("/api/auth/register", handler.AuthRegister)
	mux.HandleFunc("/api/auth/community-shortcut", handler.AuthCommunityShortcut)
	mux.HandleFunc("/api/auth/community-login", handler.AuthCommunityLogin)
	mux.HandleFunc("/api/auth/community-config", handler.AuthCommunityConfig)

	// UGC 福利系统 API
	welfareStore, err := store.NewWelfareStore(cfg.DataFile, cfg.RedisURL, cfg.RabbitMQURL)
	if err == nil {
		welfareHandler := api.NewWelfareHandler(welfareStore)
		welfareHandler.RegisterRoutes(mux)
	}

	// 前端 SPA
	if stat, err := os.Stat(cfg.PublicDir); err == nil && stat.IsDir() {
		mux.Handle("/", spaHandler(cfg.PublicDir))
	}

	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      withMiddleware(mux, cfg, logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func withMiddleware(next http.Handler, cfg config.Config, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			applyCORS(w, r, cfg.AllowedOrigins)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		requestID := fmt.Sprintf("%d-%06d", time.Now().UnixNano(), rand.Intn(999999))
		start := time.Now()

		applySecurityHeaders(w, cfg)
		applyCORS(w, r, cfg.AllowedOrigins)
		w.Header().Set("X-Request-ID", requestID)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		logger.Info("request completed",
			"requestId", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"durationMs", time.Since(start).Milliseconds(),
		)
	})
}

func applySecurityHeaders(w http.ResponseWriter, cfg config.Config) {
	if cfg.HSTSEnabled {
		maxAge := int64(cfg.HSTSDuration.Seconds())
		w.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains; preload", maxAge))
	}
	if cfg.ContentSecurityPolicy != "" {
		w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
	}
	if cfg.ReferrerPolicy != "" {
		w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
	}
	w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
	w.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
	w.Header().Set("X-XSS-Protection", cfg.XXSSProtection)
}

func applyCORS(w http.ResponseWriter, r *http.Request, allowedOrigins []string) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			return
		}
	}
}

func spaHandler(publicDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(publicDir))
	indexPath := filepath.Join(publicDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			http.NotFound(w, r)
			return
		}
		requestPath := filepath.Join(publicDir, filepath.Clean(strings.TrimPrefix(r.URL.Path, "/")))
		if info, err := os.Stat(requestPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
