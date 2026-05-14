package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Addr              string
	DataFile          string
	PublicDir         string
	AllowedOrigins    []string
	ShutdownTimeout   time.Duration
	RedisURL          string
	RabbitMQURL       string
	HCaptchaSecret    string
	HCaptchaVerifyURL string
	HCaptchaSiteKey   string

	// PostgreSQL
	PostgresDSN string

	// MySQL
	MySQLDSN string

	// 社区 SSO 配置
	CommunityURL      string
	CommunityClientID string

	// 安全配置
	HSTSEnabled           bool
	HSTSDuration          time.Duration
	CSPEnabled            bool
	CSPReportOnly         bool
	CSPDefaultSrc         string
	CSPStyleSrc           string
	CSPScriptSrc          string
	CSPImgSrc             string
	CSPFontSrc            string
	CSPConnectSrc         string
	CSPFrameSrc           string
	CSPFormActionSrc      string
	CSPObjectSrc          string
	CSPBaseURI            string
	CSPManifestSrc        string
	CSPFrameAncestors     string
	CSPReportURI          string
	ReferrerPolicy        string
	PermissionsPolicy     string
	XFrameOptions         string
	XContentTypeOptions   string
	XXSSProtection        string
	XDNSPrefetchControl   string
	XPerfPolicy           string
	ContentSecurityPolicy string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:              envOrDefault("CDK_AIRDROP_ADDR", ":8088"),
		DataFile:          envOrDefault("CDK_AIRDROP_DATA_FILE", filepath.Clean("./data/state.json")),
		PublicDir:         envOrDefault("CDK_AIRDROP_PUBLIC_DIR", filepath.Clean("../web/dist")),
		AllowedOrigins:    splitCSV(envOrDefault("CDK_AIRDROP_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		ShutdownTimeout:   10 * time.Second,
		RedisURL:          envOrDefault("CDK_AIRDROP_REDIS_URL", ""),    // e.g., redis://localhost:6379/0
		RabbitMQURL:       envOrDefault("CDK_AIRDROP_RABBITMQ_URL", ""), // e.g., amqp://guest:guest@localhost:5672/
		HCaptchaSecret:    envOrDefault("HCAPTCHA_SECRET", ""),
		HCaptchaVerifyURL: envOrDefault("HCAPTCHA_VERIFY_URL", "https://api.hcaptcha.com/siteverify"),
		HCaptchaSiteKey:   envOrDefault("HCAPTCHA_SITE_KEY", ""),

		// SQL storage
		PostgresDSN: firstEnv("CDK_POSTGRES_DSN", "DATABASE_URL"),
		MySQLDSN:    envOrDefault("CDK_MYSQL_DSN", ""),

		// 社区 SSO
		CommunityURL:      envOrDefault("CDK_COMMUNITY_URL", "http://localhost:5173"),
		CommunityClientID: envOrDefault("CDK_COMMUNITY_CLIENT_ID", "cdk-airdrop"),

		// 安全配置默认值
		HSTSEnabled:         envOrDefault("CDK_AIRDROP_HSTS_ENABLED", "true") == "true",
		HSTSDuration:        31536000 * time.Second, // 1 year
		CSPEnabled:          envOrDefault("CDK_AIRDROP_CSP_ENABLED", "true") == "true",
		CSPReportOnly:       envOrDefault("CDK_AIRDROP_CSP_REPORT_ONLY", "false") == "true",
		CSPDefaultSrc:       envOrDefault("CDK_AIRDROP_CSP_DEFAULT_SRC", "'self'"),
		CSPStyleSrc:         envOrDefault("CDK_AIRDROP_CSP_STYLE_SRC", "'self' 'unsafe-inline'"),
		CSPScriptSrc:        envOrDefault("CDK_AIRDROP_CSP_SCRIPT_SRC", "'self' https://hcaptcha.com https://*.hcaptcha.com"),
		CSPImgSrc:           envOrDefault("CDK_AIRDROP_CSP_IMG_SRC", "'self' data:"),
		CSPFontSrc:          envOrDefault("CDK_AIRDROP_CSP_FONT_SRC", "'self'"),
		CSPConnectSrc:       envOrDefault("CDK_AIRDROP_CSP_CONNECT_SRC", "'self' https://hcaptcha.com https://*.hcaptcha.com"),
		CSPFrameSrc:         envOrDefault("CDK_AIRDROP_CSP_FRAME_SRC", "https://hcaptcha.com https://*.hcaptcha.com"),
		CSPFormActionSrc:    envOrDefault("CDK_AIRDROP_CSP_FORM_ACTION_SRC", "'self'"),
		CSPObjectSrc:        envOrDefault("CDK_AIRDROP_CSP_OBJECT_SRC", "'none'"),
		CSPBaseURI:          envOrDefault("CDK_AIRDROP_CSP_BASE_URI", "'self'"),
		CSPManifestSrc:      envOrDefault("CDK_AIRDROP_CSP_MANIFEST_SRC", "'self'"),
		CSPFrameAncestors:   envOrDefault("CDK_AIRDROP_CSP_FRAME_ANCESTORS", "'none'"),
		CSPReportURI:        envOrDefault("CDK_AIRDROP_CSP_REPORT_URI", ""),
		ReferrerPolicy:      envOrDefault("CDK_AIRDROP_REFERRER_POLICY", "strict-origin-when-cross-origin"),
		PermissionsPolicy:   envOrDefault("CDK_AIRDROP_PERMISSIONS_POLICY", ""),
		XFrameOptions:       envOrDefault("CDK_AIRDROP_X_FRAME_OPTIONS", "SAMEORIGIN"),
		XContentTypeOptions: envOrDefault("CDK_AIRDROP_X_CONTENT_TYPE_OPTIONS", "nosniff"),
		XXSSProtection:      envOrDefault("CDK_AIRDROP_X_XSS_PROTECTION", "1; mode=block"),
		XDNSPrefetchControl: envOrDefault("CDK_AIRDROP_X_DNS_PREFETCH_CONTROL", "off"),
		XPerfPolicy:         envOrDefault("CDK_AIRDROP_X_PERF_POLICY", "power-efficient"),
	}

	if cfg.Addr == "" {
		return Config{}, fmt.Errorf("CDK_AIRDROP_ADDR cannot be empty")
	}
	if cfg.DataFile == "" {
		return Config{}, fmt.Errorf("CDK_AIRDROP_DATA_FILE cannot be empty")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
