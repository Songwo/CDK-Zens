const JSON_HEADERS = { "Content-Type": "application/json" };

function getAuthToken() {
  return localStorage.getItem("cdk_token") || localStorage.getItem("token") || localStorage.getItem("access_token") || "";
}

export function setAuthToken(token) {
  localStorage.setItem("cdk_token", token);
}

export function clearAuthToken() {
  localStorage.removeItem("cdk_token");
}

export function isLoggedIn() {
  return !!getAuthToken();
}

function qs(params = {}) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "" && value !== "all") search.set(key, value);
  });
  const text = search.toString();
  return text ? `?${text}` : "";
}

async function request(path, options = {}) {
  const token = getAuthToken();
  const headers = { ...(options.body instanceof FormData ? {} : JSON_HEADERS), ...(options.headers || {}) };
  if (token) headers.Authorization = `Bearer ${token}`;
  const response = await fetch(path, { ...options, headers });
  const payload = await response.json().catch(() => null);
  if (!response.ok || payload?.success === false) {
    const msg = payload?.message || payload?.error?.message || "请求失败";
    const err = new Error(msg);
    err.code = payload?.code || payload?.error?.code || "UNKNOWN";
    err.status = response.status;
    if (response.status === 401 && !window.location.pathname.startsWith("/login")) {
      clearAuthToken();
      window.location.href = `/login?returnUrl=${encodeURIComponent(window.location.pathname)}`;
    }
    throw err;
  }
  return payload?.data ?? payload;
}

function post(path, body) {
  return request(path, { method: "POST", body: body === undefined ? "{}" : JSON.stringify(body) });
}

function put(path, body) {
  return request(path, { method: "PUT", body: JSON.stringify(body) });
}

function del(path) {
  return request(path, { method: "DELETE" });
}

export const adminApi = {
  dashboard: () => request("/api/admin/dashboard"),
  health: () => request("/api/admin/health"),
  onboardingStatus: () => request("/api/admin/onboarding/status"),

  listProjects: (params) => request(`/api/admin/projects${qs(params)}`),
  createProject: (body) => post("/api/admin/projects", body),
  getProject: (id) => request(`/api/admin/projects/${id}`),
  updateProject: (id, body) => put(`/api/admin/projects/${id}`, body),
  deleteProject: (id) => del(`/api/admin/projects/${id}`),
  archiveProject: (id) => post(`/api/admin/projects/${id}/archive`),
  bindCampaign: (id, campaignId) => post(`/api/admin/projects/${id}/bind-campaign`, { campaignId }),
  unbindCampaign: (id, campaignId) => post(`/api/admin/projects/${id}/unbind-campaign`, { campaignId }),
  projectStats: (id) => request(`/api/admin/projects/${id}/stats`),

  listCampaigns: (params) => request(`/api/admin/campaigns${qs(params)}`),
  createCampaign: (body) => post("/api/admin/campaigns", body),
  getCampaign: (id) => request(`/api/admin/campaigns/${id}`),
  updateCampaign: (id, body) => put(`/api/admin/campaigns/${id}`, body),
  deleteCampaign: (id) => del(`/api/admin/campaigns/${id}`),
  pauseCampaign: (id) => post(`/api/admin/campaigns/${id}/pause`),
  resumeCampaign: (id) => post(`/api/admin/campaigns/${id}/resume`),
  endCampaign: (id) => post(`/api/admin/campaigns/${id}/end`),
  campaignCDKs: (id, params) => request(`/api/admin/campaigns/${id}/cdks${qs(params)}`),
  campaignNodes: (id, params) => request(`/api/admin/campaigns/${id}/nodes${qs(params)}`),
  campaignClaims: (id, params) => request(`/api/admin/campaigns/${id}/claims${qs(params)}`),
  importCampaignCDKs: (id, codes) => post(`/api/admin/campaigns/${id}/cdks/import`, { codes }),

  listCDKs: (params) => request(`/api/admin/cdks${qs(params)}`),
  importCDKs: (projectId, campaignId, codes) => post("/api/admin/cdks/import", { projectId, campaignId, codes }),
  freezeCDK: (id) => post(`/api/admin/cdks/${id}/freeze`),
  unfreezeCDK: (id) => post(`/api/admin/cdks/${id}/unfreeze`),
  invalidateCDK: (id) => post(`/api/admin/cdks/${id}/invalidate`),
  deleteCDK: (id) => del(`/api/admin/cdks/${id}`),
  batchFreezeCDKs: (ids) => post("/api/admin/cdks/batch-freeze", { ids }),
  batchInvalidateCDKs: (ids) => post("/api/admin/cdks/batch-invalidate", { ids }),
  exportCDKs: (filter) => post(`/api/admin/cdks/export${qs(filter)}`),

  listNodes: (params) => request(`/api/admin/nodes${qs(params)}`),
  createNode: (body) => post("/api/admin/nodes", body),
  getNode: (id) => request(`/api/admin/nodes/${id}`),
  updateNode: (id, body) => put(`/api/admin/nodes/${id}`, body),
  deleteNode: (id) => del(`/api/admin/nodes/${id}`),
  pauseNode: (id) => post(`/api/admin/nodes/${id}/pause`),
  resumeNode: (id) => post(`/api/admin/nodes/${id}/resume`),
  nodeClaims: (id, params) => request(`/api/admin/nodes/${id}/claims${qs(params)}`),
  batchEnableNodeCaptcha: (ids) => post("/api/admin/nodes/batch-enable-captcha", { ids }),
  batchDisableNodeCaptcha: (ids) => post("/api/admin/nodes/batch-disable-captcha", { ids }),

  listClaims: (params) => request(`/api/admin/claims${qs(params)}`),
  getClaim: (id) => request(`/api/admin/claims/${id}`),
  markClaimRisk: (id) => post(`/api/admin/claims/${id}/mark-risk`),
  exportClaims: (filter) => post(`/api/admin/claims/export${qs(filter)}`),

  analyticsOverview: (params) => request(`/api/admin/analytics/overview${qs(params)}`),
  logs: (params) => request(`/api/admin/logs${qs(params)}`),
  getLog: (id) => request(`/api/admin/logs/${id}`),
  deleteLog: (id) => del(`/api/admin/logs/${id}`),
  cleanupLogs: (days) => post("/api/admin/logs/cleanup", { days }),
  exportLogs: (filter) => post(`/api/admin/logs/export${qs(filter)}`),
  exportTasks: (params) => request(`/api/admin/export-tasks${qs(params)}`),

  riskOverview: () => request("/api/admin/risk/overview"),
  riskRules: () => request("/api/admin/risk/rules"),
  saveRiskRule: (body, id) => (id ? put(`/api/admin/risk/rules/${id}`, body) : post("/api/admin/risk/rules", body)),
  deleteRiskRule: (id) => del(`/api/admin/risk/rules/${id}`),
  enableRiskRule: (id) => post(`/api/admin/risk/rules/${id}/enable`),
  disableRiskRule: (id) => post(`/api/admin/risk/rules/${id}/disable`),
  blacklist: () => request("/api/admin/risk/blacklist"),
  saveBlacklist: (body, id) => (id ? put(`/api/admin/risk/blacklist/${id}`, body) : post("/api/admin/risk/blacklist", body)),
  deleteBlacklist: (id) => del(`/api/admin/risk/blacklist/${id}`),
  enableBlacklist: (id) => post(`/api/admin/risk/blacklist/${id}/enable`),
  disableBlacklist: (id) => post(`/api/admin/risk/blacklist/${id}/disable`),
  riskHits: () => request("/api/admin/risk/hits"),

  captchaConfig: () => request("/api/admin/captcha/config"),
  updateCaptchaConfig: (body) => put("/api/admin/captcha/config", body),
  testCaptcha: () => post("/api/admin/captcha/test"),
  captchaNodes: (params) => request(`/api/admin/captcha/nodes${qs(params)}`),
  batchEnableCaptcha: (ids) => post("/api/admin/captcha/nodes/batch-enable", { ids }),
  batchDisableCaptcha: (ids) => post("/api/admin/captcha/nodes/batch-disable", { ids }),

  settings: () => request("/api/admin/settings"),
  updateSettings: (body) => put("/api/admin/settings", body),
  backup: () => post("/api/admin/backup"),
  restore: (body) => post("/api/admin/restore", body),
  admins: () => request("/api/admin/admins"),
  createAdmin: (body) => post("/api/admin/admins", body),
  updateAdmin: (id, body) => put(`/api/admin/admins/${id}`, body),
  deleteAdmin: (id) => del(`/api/admin/admins/${id}`),

  disableProject(id) {
    return this.pauseCampaign(id);
  },
  getRecords(id) {
    return this.campaignClaims(id);
  },
};

export const claimApi = {
  getProject(code, fingerprint) {
    return this.getNode(code, fingerprint);
  },
  submit(code, body) {
    return this.submitNode(code, body);
  },
  getNode(slug, fingerprint) {
    return request(`/api/public/claim/${slug}${qs({ fingerprint })}`);
  },
  submitNode(slug, body) {
    return post(`/api/public/claim/${slug}`, body);
  },
  getResult(slug, claimToken) {
    return request(`/api/public/claim/${slug}/result${qs({ claimToken })}`);
  },
};

export const userApi = {
  getMe: () => request("/api/me"),
};

// ─── 社区 SSO 接口 ─────────────────────────────────────

export const communityApi = {
  /** 获取社区 SSO 配置（社区地址、clientId） */
  config: () => request("/api/auth/community-config"),

  /** 使用社区 SSO Token 登录 */
  login: (ssoToken) => post("/api/auth/community-login", { ssoToken }),
};

// ─── 公开品牌信息 ────────────────────────────────────────

export const publicApi = {
  brand: () => request("/health"),
};
