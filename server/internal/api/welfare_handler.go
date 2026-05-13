package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"cdk-airdrop-station/server/internal/model"
	"cdk-airdrop-station/server/internal/store"
)

// WelfareHandler 福利 API 处理器
type WelfareHandler struct {
	store *store.WelfareStore
}

// NewWelfareHandler 创建福利处理器
func NewWelfareHandler(store *store.WelfareStore) *WelfareHandler {
	return &WelfareHandler{store: store}
}

// RegisterRoutes 注册路由
func (h *WelfareHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/welfare/create", h.handleCreate)
	mux.HandleFunc("GET /api/welfare/list", h.handleList)
	mux.HandleFunc("GET /api/welfare/detail/{id}", h.handleDetail)
	mux.HandleFunc("GET /api/welfare/page/{code}", h.handlePageInfo)
	mux.HandleFunc("POST /api/welfare/claim/{code}", h.handleClaim)
	mux.HandleFunc("POST /api/welfare/disable/{id}", h.handleDisable)
	mux.HandleFunc("GET /api/welfare/records/{id}", h.handleRecords)
}

// handleCreate 创建福利
func (h *WelfareHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req model.CreateWelfareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.NewAppError(http.StatusBadRequest, "invalid_request", "请求格式错误"))
		return
	}

	// 从 JWT 获取用户 ID
	userID := getUserIDFromContext(r)

	project, err := h.store.CreateWelfare(req, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// handleList 列出所有福利
func (h *WelfareHandler) handleList(w http.ResponseWriter, r *http.Request) {
	welfares := h.store.ListWelfares()
	writeJSON(w, http.StatusOK, welfares)
}

// handleDetail 获取福利详情
func (h *WelfareHandler) handleDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "missing_id", "缺少项目 ID"))
		return
	}

	detail, err := h.store.GetWelfareDetail(id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// handlePageInfo 获取福利页面信息
func (h *WelfareHandler) handlePageInfo(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "missing_code", "缺少项目代码"))
		return
	}

	userID := getUserIDFromContext(r)
	fingerprint := r.URL.Query().Get("fingerprint")

	info, err := h.store.GetWelfarePageInfo(code, userID, fingerprint)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// handleClaim 领取福利
func (h *WelfareHandler) handleClaim(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "missing_code", "缺少项目代码"))
		return
	}

	var req model.WelfareClaimRequest
	// 如果没有 body，使用空请求
	json.NewDecoder(r.Body).Decode(&req)

	userID := getUserIDFromContext(r)
	fingerprint := req.Fingerprint
	if fingerprint == "" {
		fingerprint = r.Header.Get("X-Fingerprint")
	}

	ip := getClientIP(r)
	ua := r.UserAgent()

	resp, err := h.store.ExecuteClaim(r.Context(), code, userID, fingerprint, ip, ua)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleDisable 停用福利
func (h *WelfareHandler) handleDisable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "missing_id", "缺少项目 ID"))
		return
	}

	userID := getUserIDFromContext(r)

	if err := h.store.DisableWelfare(id, userID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "已停用"})
}

// handleRecords 获取领取记录
func (h *WelfareHandler) handleRecords(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, model.NewAppError(http.StatusBadRequest, "missing_id", "缺少项目 ID"))
		return
	}

	records, err := h.store.GetClaimRecords(id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// ─── 辅助函数 ────────────────────────────────────────────

func getUserIDFromContext(r *http.Request) string {
	// 从 JWT 或 session 中获取用户 ID
	// 这里简化处理，实际项目中应该从认证中间件中获取
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return ""
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
