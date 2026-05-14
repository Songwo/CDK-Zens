package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"cdk-airdrop-station/server/internal/hcaptcha"
	"cdk-airdrop-station/server/internal/model"
	"cdk-airdrop-station/server/internal/store"

	"github.com/golang-jwt/jwt/v5"
)

type Handler struct {
	store             *store.Store
	logger            *slog.Logger
	hcaptcha          hcaptcha.Verifier
	communityURL      string
	communityClientID string
}

func New(store *store.Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) SetCommunityConfig(communityURL, communityClientID string) {
	h.communityURL = communityURL
	h.communityClientID = communityClientID
}

func (h *Handler) SetHCaptchaVerifier(verifier hcaptcha.Verifier) { h.hcaptcha = verifier }

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	s := h.store.Settings()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "ok",
		"version":          "5.0.0",
		"systemName":       s.SystemName,
		"brandName":        s.BrandName,
		"brandEnglishName": s.BrandEnglishName,
		"logoText":         s.LogoText,
	})
}

func (h *Handler) AdminHealth(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, h.store.Health())
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	writeJSON(w, http.StatusOK, h.store.DashboardForUser(user))
}

func (h *Handler) AdminOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, methodErr())
		return
	}
	writeJSON(w, http.StatusOK, h.store.OnboardingStatusForUser(user))
}

func (h *Handler) AdminProjectsRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	if r.URL.Path != "/api/admin/projects" {
		h.AdminProjectDetailRouter(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListProjectsPageForUser(queryMap(r), user))
	case http.MethodPost:
		var req model.CreateProjectRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.CreateProject(req, user.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminProjectDetailRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	id, action := parseResourcePath(r.URL.Path, "/api/admin/projects/")
	if id == "" {
		writeError(w, model.ErrBadRequest)
		return
	}
	switch {
	case action == "" && r.Method == http.MethodGet:
		res, err := h.store.GetProject(id)
		respond(w, res, err)
	case action == "" && r.Method == http.MethodPut:
		var req model.UpdateProjectRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.UpdateProject(id, req, actor)
		respond(w, res, err)
	case action == "" && r.Method == http.MethodDelete:
		respondOK(w, h.store.DeleteProject(id, actor))
	case action == "archive" && r.Method == http.MethodPost:
		respondOK(w, h.store.ArchiveProject(id, actor))
	case action == "bind-campaign" && r.Method == http.MethodPost:
		var req struct {
			CampaignID string `json:"campaignId"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		respondOK(w, h.store.BindCampaign(id, req.CampaignID, actor))
	case action == "unbind-campaign" && r.Method == http.MethodPost:
		var req struct {
			CampaignID string `json:"campaignId"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		respondOK(w, h.store.UnbindCampaign(id, req.CampaignID, actor))
	case action == "stats" && r.Method == http.MethodGet:
		res, err := h.store.ProjectStats(id)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminCampaignsRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	if r.URL.Path != "/api/admin/campaigns" {
		h.AdminCampaignDetailRouter(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListCampaignsForUser(queryMap(r), user))
	case http.MethodPost:
		var req model.CreateCampaignRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.CreateCampaign(req, actor)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminCampaignDetailRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	id, action := parseResourcePath(r.URL.Path, "/api/admin/campaigns/")
	if id == "" {
		writeError(w, model.ErrBadRequest)
		return
	}
	switch {
	case action == "" && r.Method == http.MethodGet:
		res, err := h.store.GetCampaign(id)
		respond(w, res, err)
	case action == "" && r.Method == http.MethodPut:
		var req model.UpdateCampaignRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.UpdateCampaign(id, req, actor)
		respond(w, res, err)
	case action == "" && r.Method == http.MethodDelete:
		respondOK(w, h.store.DeleteCampaign(id, actor))
	case action == "pause" && r.Method == http.MethodPost:
		respondOK(w, h.store.SetCampaignStatus(id, model.StatusPaused, actor))
	case action == "resume" && r.Method == http.MethodPost:
		respondOK(w, h.store.SetCampaignStatus(id, model.StatusActive, actor))
	case action == "end" && r.Method == http.MethodPost:
		respondOK(w, h.store.SetCampaignStatus(id, model.StatusEnded, actor))
	case action == "cdks" && r.Method == http.MethodGet:
		q := queryMap(r)
		q["campaignId"] = id
		writeJSON(w, http.StatusOK, h.store.ListCDKsForUser(q, user))
	case action == "nodes" && r.Method == http.MethodGet:
		q := queryMap(r)
		q["campaignId"] = id
		writeJSON(w, http.StatusOK, h.store.ListNodesForUser(q, user))
	case action == "claims" || action == "records":
		q := queryMap(r)
		q["campaignId"] = id
		writeJSON(w, http.StatusOK, h.store.ListClaimsForUser(q, user))
	case action == "stats" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.AnalyticsForUser(map[string]string{"campaignId": id, "range": "30d"}, user))
	case action == "cdks/import" && r.Method == http.MethodPost:
		var req model.ImportCDKRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		codes := req.Codes
		if len(codes) == 0 {
			codes = req.Items
		}
		res, err := h.store.ImportCDKsForProject(firstNonEmpty(req.ProjectID, req.ProjectIDSnake), id, codes, actor)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminCDKsRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	if r.URL.Path != "/api/admin/cdks" {
		h.AdminCDKDetailRouter(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, methodErr())
		return
	}
	writeJSON(w, http.StatusOK, h.store.ListCDKsForUser(queryMap(r), user))
}

func (h *Handler) AdminCDKDetailRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/cdks/"), "/")
	switch {
	case rest == "import" && r.Method == http.MethodPost:
		var req model.ImportCDKRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		codes := req.Codes
		if len(codes) == 0 {
			codes = req.Items
		}
		res, err := h.store.ImportCDKsForProject(firstNonEmpty(req.ProjectID, req.ProjectIDSnake), req.CampaignID, codes, actor)
		respond(w, res, err)
	case rest == "batch-freeze" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchSetCDKStatus(req.IDs, model.StatusFrozen, actor)
		respond(w, map[string]int{"count": n}, err)
	case rest == "batch-invalidate" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchSetCDKStatus(req.IDs, model.StatusInvalid, actor)
		respond(w, map[string]int{"count": n}, err)
	case rest == "export" && r.Method == http.MethodPost:
		res, err := h.store.ExportCSV("cdks", queryMap(r), actor)
		respond(w, res, err)
	default:
		id, action := parseResourcePath(r.URL.Path, "/api/admin/cdks/")
		if id == "" {
			writeError(w, model.ErrBadRequest)
			return
		}
		switch {
		case action == "freeze" && r.Method == http.MethodPost:
			respondOK(w, h.store.SetCDKStatus(id, model.StatusFrozen, actor))
		case action == "unfreeze" && r.Method == http.MethodPost:
			respondOK(w, h.store.SetCDKStatus(id, model.StatusUnused, actor))
		case action == "invalidate" && r.Method == http.MethodPost:
			respondOK(w, h.store.SetCDKStatus(id, model.StatusInvalid, actor))
		case action == "" && r.Method == http.MethodDelete:
			respondOK(w, h.store.DeleteCDK(id, actor))
		default:
			writeError(w, methodErr())
		}
	}
}

func (h *Handler) AdminNodesRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	if r.URL.Path != "/api/admin/nodes" {
		h.AdminNodeDetailRouter(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListNodesForUser(queryMap(r), user))
	case http.MethodPost:
		var req model.CreateNodeRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.CreateNode(req, actor)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminNodeDetailRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/nodes/"), "/")
	switch {
	case rest == "batch-enable-captcha" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchNodeCaptcha(req.IDs, true, actor)
		respond(w, map[string]int{"count": n}, err)
	case rest == "batch-disable-captcha" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchNodeCaptcha(req.IDs, false, actor)
		respond(w, map[string]int{"count": n}, err)
	default:
		id, action := parseResourcePath(r.URL.Path, "/api/admin/nodes/")
		if id == "" {
			writeError(w, model.ErrBadRequest)
			return
		}
		switch {
		case action == "" && r.Method == http.MethodGet:
			res, err := h.store.GetNode(id)
			respond(w, res, err)
		case action == "" && r.Method == http.MethodPut:
			var req model.UpdateNodeRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, err)
				return
			}
			res, err := h.store.UpdateNode(id, req, actor)
			respond(w, res, err)
		case action == "" && r.Method == http.MethodDelete:
			respondOK(w, h.store.DeleteNode(id, actor))
		case action == "pause" && r.Method == http.MethodPost:
			respondOK(w, h.store.SetNodeStatus(id, model.StatusPaused, actor))
		case action == "resume" && r.Method == http.MethodPost:
			respondOK(w, h.store.SetNodeStatus(id, model.StatusActive, actor))
		case action == "stats" && r.Method == http.MethodGet:
			res, err := h.store.GetNode(id)
			respond(w, res, err)
		case action == "claims" && r.Method == http.MethodGet:
			q := queryMap(r)
			q["nodeId"] = id
			writeJSON(w, http.StatusOK, h.store.ListClaimsForUser(q, user))
		default:
			writeError(w, methodErr())
		}
	}
}

func (h *Handler) PublicClaimRouter(w http.ResponseWriter, r *http.Request) {
	slug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/public/claim/"), "/")
	if slug == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "MISSING_SLUG", "缺少节点标识"))
		return
	}
	if strings.HasSuffix(slug, "/result") {
		slug = strings.TrimSuffix(slug, "/result")
		token := r.URL.Query().Get("claimToken")
		res, err := h.store.GetClaimResultByToken(slug, token)
		respond(w, res, err)
		return
	}
	// 强制登录才能领取
	userID, _ := h.parseJWTUser(r)
	if userID == "" {
		writeError(w, model.ErrLoginRequired)
		return
	}
	switch r.Method {
	case http.MethodGet:
		res, err := h.store.GetPublicNodeClaim(slug, userID, r.URL.Query().Get("fingerprint"))
		respond(w, res, err)
	case http.MethodPost:
		var req model.PublicClaimRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		node, err := h.store.GetNodeBySlug(slug)
		if err != nil {
			writeError(w, err)
			return
		}
		captchaPassed := false
		if node["requireCaptcha"] == true {
			token := firstNonEmpty(req.HCaptchaToken, req.CaptchaToken)
			if token == "" {
				writeError(w, model.ErrCaptchaRequired)
				return
			}
			if err := h.hcaptcha.VerifyHCaptcha(r.Context(), token, clientIP(r), ""); err != nil {
				writeError(w, err)
				return
			}
			captchaPassed = true
		}
		res, err := h.store.ClaimNodeReward(slug, userID, req.Fingerprint, clientIP(r), r.UserAgent(), captchaPassed)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) ClaimRouter(w http.ResponseWriter, r *http.Request) {
	// 强制登录才能领取
	userID, _ := h.parseJWTUser(r)
	if userID == "" {
		writeError(w, model.ErrLoginRequired)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "MISSING_CODE", "缺少项目代码"))
		return
	}
	r.URL.Path = "/api/public/claim/" + code
	h.PublicClaimRouter(w, r)
}

func (h *Handler) AdminClaimsRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	actor := user.ID
	if r.URL.Path == "/api/admin/claims" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, h.store.ListClaimsForUser(queryMap(r), user))
		return
	}
	if r.URL.Path == "/api/admin/claims/export" && r.Method == http.MethodPost {
		res, err := h.store.ExportCSV("claims", queryMap(r), actor)
		respond(w, res, err)
		return
	}
	id, action := parseResourcePath(r.URL.Path, "/api/admin/claims/")
	switch {
	case id != "" && action == "" && r.Method == http.MethodGet:
		res, err := h.store.GetClaim(id)
		respond(w, res, err)
	case id != "" && action == "mark-risk" && r.Method == http.MethodPost:
		respondOK(w, h.store.MarkClaimRisk(id, actor))
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminAnalyticsRouter(w http.ResponseWriter, r *http.Request) {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return
	}
	data := h.store.AnalyticsForUser(queryMap(r), user)
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/analytics/")
	switch path {
	case "visits-trend":
		writeJSON(w, http.StatusOK, data["visitsTrend"])
	case "claims-trend":
		writeJSON(w, http.StatusOK, data["claimsTrend"])
	case "conversion-ranking":
		writeJSON(w, http.StatusOK, data["conversionRanking"])
	case "campaign-ranking":
		writeJSON(w, http.StatusOK, data["campaignRanking"])
	case "node-ranking":
		writeJSON(w, http.StatusOK, data["nodeRanking"])
	case "failure-reasons":
		writeJSON(w, http.StatusOK, data["failureReasons"])
	default:
		writeJSON(w, http.StatusOK, data)
	}
}

func (h *Handler) AdminLogsRouter(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	if r.URL.Path == "/api/admin/logs" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, h.store.ListLogs(queryMap(r)))
		return
	}
	if r.URL.Path == "/api/admin/logs/export" && r.Method == http.MethodPost {
		actor, _ := h.parseJWTUser(r)
		res, err := h.store.ExportCSV("logs", queryMap(r), actor)
		respond(w, res, err)
		return
	}
	if r.URL.Path == "/api/admin/logs/cleanup" && r.Method == http.MethodPost {
		actor, _ := h.parseJWTUser(r)
		var req struct {
			Days int `json:"days"`
		}
		_ = decodeJSON(r, &req)
		n, err := h.store.CleanupLogs(req.Days, actor)
		respond(w, map[string]int{"count": n}, err)
		return
	}
	id, _ := parseResourcePath(r.URL.Path, "/api/admin/logs/")
	switch r.Method {
	case http.MethodGet:
		res, err := h.store.GetLog(id)
		respond(w, res, err)
	case http.MethodDelete:
		respondOK(w, h.store.DeleteLog(id))
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminRiskRouter(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	actor, _ := h.parseJWTUser(r)
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/risk/"), "/")
	switch {
	case path == "overview" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.RiskOverview())
	case path == "rules" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListRiskRules())
	case path == "rules" && r.Method == http.MethodPost:
		var req model.RiskRule
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.SaveRiskRule("", req, actor)
		respond(w, res, err)
	case strings.HasPrefix(path, "rules/"):
		id, action := parseResourcePath(r.URL.Path, "/api/admin/risk/rules/")
		if action == "" && r.Method == http.MethodPut {
			var req model.RiskRule
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, err)
				return
			}
			res, err := h.store.SaveRiskRule(id, req, actor)
			respond(w, res, err)
			return
		}
		if action == "" && r.Method == http.MethodDelete {
			respondOK(w, h.store.DeleteRiskRule(id, actor))
			return
		}
		if action == "enable" {
			respondOK(w, h.store.EnableRiskRule(id, true, actor))
			return
		}
		if action == "disable" {
			respondOK(w, h.store.EnableRiskRule(id, false, actor))
			return
		}
		writeError(w, methodErr())
	case path == "blacklist" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListBlacklist())
	case path == "blacklist" && r.Method == http.MethodPost:
		var req model.BlacklistItem
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.SaveBlacklist("", req, actor)
		respond(w, res, err)
	case strings.HasPrefix(path, "blacklist/"):
		id, action := parseResourcePath(r.URL.Path, "/api/admin/risk/blacklist/")
		if action == "" && r.Method == http.MethodPut {
			var req model.BlacklistItem
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, err)
				return
			}
			res, err := h.store.SaveBlacklist(id, req, actor)
			respond(w, res, err)
			return
		}
		if action == "" && r.Method == http.MethodDelete {
			respondOK(w, h.store.DeleteBlacklist(id, actor))
			return
		}
		if action == "enable" {
			respondOK(w, h.store.EnableBlacklist(id, true, actor))
			return
		}
		if action == "disable" {
			respondOK(w, h.store.EnableBlacklist(id, false, actor))
			return
		}
		writeError(w, methodErr())
	case path == "hits" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.RiskHits())
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminCaptchaRouter(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	actor, _ := h.parseJWTUser(r)
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/captcha/"), "/")
	siteConfigured := strings.TrimSpace(os.Getenv("HCAPTCHA_SITE_KEY")) != "" || strings.TrimSpace(os.Getenv("VITE_HCAPTCHA_SITE_KEY")) != ""
	secretConfigured := strings.TrimSpace(h.hcaptcha.Secret) != ""
	switch {
	case path == "config" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.CaptchaConfig(siteConfigured, secretConfigured))
	case path == "config" && r.Method == http.MethodPut:
		var req model.CaptchaConfig
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.UpdateCaptchaConfig(req, actor)
		res.HCaptchaSiteKeyConfigured = siteConfigured
		res.HCaptchaSecretConfigured = secretConfigured
		respond(w, res, err)
	case path == "test" && r.Method == http.MethodPost:
		status, msg := "ok", "hCaptcha Secret 已配置，siteverify 地址可用性将在真实 token 校验时确认"
		if !secretConfigured {
			status = "failed"
			msg = "未配置 HCAPTCHA_SECRET"
		}
		_ = h.store.SetCaptchaTest(status, msg)
		writeJSON(w, http.StatusOK, h.store.CaptchaConfig(siteConfigured, secretConfigured))
	case path == "nodes" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListNodes(queryMap(r)))
	case path == "nodes/batch-enable" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchNodeCaptcha(req.IDs, true, actor)
		respond(w, map[string]int{"count": n}, err)
	case path == "nodes/batch-disable" && r.Method == http.MethodPost:
		var req model.BatchIDsRequest
		_ = decodeJSON(r, &req)
		n, err := h.store.BatchNodeCaptcha(req.IDs, false, actor)
		respond(w, map[string]int{"count": n}, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminSettingsRouter(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	actor, _ := h.parseJWTUser(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.Settings())
	case http.MethodPut:
		var req model.Settings
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.UpdateSettings(req, actor)
		respond(w, res, err)
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) AdminSystemRouter(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRole(w, r) {
		return
	}
	actor, _ := h.parseJWTUser(r)
	switch {
	case r.URL.Path == "/api/admin/export-tasks" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListExportTasks(queryMap(r)))
	case r.URL.Path == "/api/admin/backup" && r.Method == http.MethodPost:
		res, err := h.store.Backup(actor)
		respond(w, res, err)
	case r.URL.Path == "/api/admin/restore" && r.Method == http.MethodPost:
		var filename string
		var payload []byte
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			file, header, err := r.FormFile("file")
			if err == nil {
				defer file.Close()
				payload, _ = io.ReadAll(file)
				filename = header.Filename
			}
		} else {
			var req struct {
				Filename string          `json:"filename"`
				Content  json.RawMessage `json:"content"`
			}
			_ = decodeJSON(r, &req)
			filename = req.Filename
			payload = req.Content
		}
		respondOK(w, h.store.Restore(filename, payload, actor))
	case r.URL.Path == "/api/admin/admins" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.ListAdmins())
	case r.URL.Path == "/api/admin/admins" && r.Method == http.MethodPost:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		res, err := h.store.CreateUser(req.Username, req.Password)
		respond(w, res, err)
	case strings.HasPrefix(r.URL.Path, "/api/admin/admins/"):
		id, _ := parseResourcePath(r.URL.Path, "/api/admin/admins/")
		if r.Method == http.MethodPut {
			var req model.Admin
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, err)
				return
			}
			res, err := h.store.UpdateAdmin(id, req, actor)
			respond(w, res, err)
			return
		}
		if r.Method == http.MethodDelete {
			respondOK(w, h.store.DeleteAdmin(id, actor))
			return
		}
		writeError(w, methodErr())
	case strings.HasPrefix(r.URL.Path, "/api/admin/exports/"):
		h.store.ServeDataFile(w, r, "exports")
	case strings.HasPrefix(r.URL.Path, "/api/admin/backups/"):
		h.store.ServeDataFile(w, r, "backups")
	default:
		writeError(w, methodErr())
	}
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, username := h.parseJWTUser(r)
	if userID == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"loggedIn": false})
		return
	}
	info := model.UserInfo{UserID: userID, Username: username, Role: "user"}
	if u := h.store.FindUserByID(userID); u != nil {
		if u.Role != "" {
			info.Role = u.Role
		}
		info.Avatar = u.Avatar
		info.Nickname = u.Nickname
		info.Email = u.Email
		info.CommunityUserID = u.CommunityUserID
	} else if username != "" {
		info.Role = "admin"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"loggedIn": true, "user": info, "username": username, "role": info.Role})
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	// 本地登录已禁用，仅支持社区 SSO 登录
	writeError(w, model.NewAppError(http.StatusForbidden, "LOCAL_LOGIN_DISABLED", "本地登录已禁用，请使用社区账号登录"))
}

func (h *Handler) AuthRegister(w http.ResponseWriter, r *http.Request) {
	// 本地注册已禁用，仅支持社区 SSO 登录
	writeError(w, model.NewAppError(http.StatusForbidden, "LOCAL_REGISTER_DISABLED", "本地注册已禁用，请使用社区账号登录"))
}

func (h *Handler) AuthCommunityShortcut(w http.ResponseWriter, r *http.Request) {
	// 兼容旧路由，转发到新处理逻辑
	h.AuthCommunityLogin(w, r)
}

func (h *Handler) AuthCommunityLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, methodErr())
		return
	}
	var req struct {
		SsoToken string `json:"ssoToken"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.SsoToken == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "MISSING_TOKEN", "缺少 ssoToken"))
		return
	}

	// 解析社区 SSO Token（使用相同密钥）
	secretStr := os.Getenv("CDK_COMMUNITY_JWT_SECRET")
	if secretStr == "" {
		secretStr = "zf3sg4ikVorzR1S/cel4+o/VkQH+4LZxU8RkX0c0Ne6BFPf9NnPL09AoHSK9Zg95"
	}
	secret := []byte(secretStr)
	token, err := jwt.Parse(req.SsoToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "INVALID_SSO_TOKEN", "SSO Token 无效或已过期"))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "INVALID_CLAIMS", "Token claims 解析失败"))
		return
	}

	// 验证 SSO 标志位
	if ssoFlag, _ := claims["sso"].(bool); !ssoFlag {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "NOT_SSO_TOKEN", "该 Token 不是 SSO 授权 Token"))
		return
	}

	// 提取用户信息
	communityUserID, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	nickname, _ := claims["nickname"].(string)
	avatar, _ := claims["avatar"].(string)
	email, _ := claims["email"].(string)

	// 根据社区角色映射 CDK 角色
	role := "user"
	if rolesRaw, ok := claims["roles"]; ok {
		if rolesArr, ok := rolesRaw.([]interface{}); ok {
			for _, r := range rolesArr {
				if rs, ok := r.(string); ok {
					if rs == "ROLE_ADMIN" || rs == "ROLE_SUPER_ADMIN" {
						role = "admin"
						break
					}
				}
			}
		}
	}

	if communityUserID == "" || username == "" {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "INCOMPLETE_CLAIMS", "Token 中缺少用户信息"))
		return
	}

	// 在 CDK 系统中创建/更新用户
	user, err := h.store.CreateOrUpdateCommunityUser(communityUserID, username, avatar, nickname, email, role)
	if err != nil {
		writeError(w, model.NewAppError(http.StatusInternalServerError, "USER_SYNC_FAILED", "用户同步失败"))
		return
	}

	// 签发 CDK 本地 JWT
	cdkToken, _ := h.generateJWT(user.ID, user.Username)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": cdkToken,
		"user": model.UserInfo{
			UserID:          user.ID,
			Username:        user.Username,
			Role:            user.Role,
			Avatar:          user.Avatar,
			Nickname:        user.Nickname,
			Email:           user.Email,
			CommunityUserID: user.CommunityUserID,
		},
	})
}

func (h *Handler) AuthCommunityConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"communityUrl": h.communityURL,
		"clientId":     h.communityClientID,
	})
}

func (h *Handler) parseJWTUser(r *http.Request) (string, string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", ""
	}
	tokenString := authHeader[7:]
	secret := []byte("zf3sg4ikVorzR1S/cel4+o/VkQH+4LZxU8RkX0c0Ne6BFPf9NnPL09AoHSK9Zg95")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", ""
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ""
	}
	sub, _ := claims["sub"].(string)
	name, _ := claims["username"].(string)
	return sub, name
}

// getAuthUser returns the full user object from JWT, or nil if not logged in.
func (h *Handler) getAuthUser(r *http.Request) *model.User {
	userID, _ := h.parseJWTUser(r)
	if userID == "" {
		return nil
	}
	u := h.store.FindUserByID(userID)
	if u != nil {
		return u
	}
	// fallback for legacy users - create a minimal user object
	_, username := h.parseJWTUser(r)
	return &model.User{ID: userID, Username: username, Role: "admin"}
}

// requireAdminRole checks that the user is logged in AND has admin role.
func (h *Handler) requireAdminRole(w http.ResponseWriter, r *http.Request) bool {
	user := h.getAuthUser(r)
	if user == nil {
		writeError(w, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "请先登录"))
		return false
	}
	if user.Role != "admin" {
		writeError(w, model.NewAppError(http.StatusForbidden, "FORBIDDEN", "仅管理员可访问此功能"))
		return false
	}
	return true
}

func (h *Handler) generateJWT(userID, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID, "username": username, "exp": time.Now().Add(24 * time.Hour).Unix()})
	secret := []byte("zf3sg4ikVorzR1S/cel4+o/VkQH+4LZxU8RkX0c0Ne6BFPf9NnPL09AoHSK9Zg95")
	return token.SignedString(secret)
}

func decodeJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return model.NewAppError(http.StatusBadRequest, "INVALID_JSON", "请求体格式不正确")
	}
	return nil
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func parseResourcePath(path, prefix string) (id, action string) {
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.SplitN(rest, "/", 2)
	id = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action
}

func queryMap(r *http.Request) map[string]string {
	out := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func respond(w http.ResponseWriter, data interface{}, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func respondOK(w http.ResponseWriter, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "操作成功"})
}

func methodErr() error {
	return model.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "不支持的请求方法")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
