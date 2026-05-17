package model

import (
	"net/http"
	"time"
)

const (
	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusDisabled  = "disabled"
	StatusArchived  = "archived"
	StatusDraft     = "draft"
	StatusEnded     = "ended"
	StatusExhausted = "exhausted"
	StatusUnused    = "unused"
	StatusClaimed   = "claimed"
	StatusFrozen    = "frozen"
	StatusInvalid   = "invalid"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusBlocked   = "blocked"
)

type User struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	Role            string    `json:"role,omitempty"`
	CommunityUserID string    `json:"communityUserId,omitempty"`
	Avatar          string    `json:"avatar,omitempty"`
	Nickname        string    `json:"nickname,omitempty"`
	Email           string    `json:"email,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       string    `json:"updatedAt,omitempty"`
}

type Admin = User

type SystemData struct {
	Projects      map[string]*Project          `json:"projects"`
	Campaigns     map[string]*Campaign         `json:"campaigns"`
	CDKs          map[string]*CDK              `json:"cdks"`
	Nodes         map[string]*DistributionNode `json:"nodes"`
	ClaimRecords  map[string]*ClaimRecord      `json:"claimRecords"`
	RiskRules     map[string]*RiskRule         `json:"riskRules"`
	Blacklist     map[string]*BlacklistItem    `json:"blacklist"`
	CaptchaConfig CaptchaConfig                `json:"captchaConfig"`
	SystemLogs    map[string]*SystemLog        `json:"systemLogs"`
	ExportTasks   map[string]*ExportTask       `json:"exportTasks"`
	Settings      Settings                     `json:"settings"`
	Admins        map[string]*Admin            `json:"admins"`
	Users         map[string]*User             `json:"users"`
}

type Project struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	CampaignIDs []string `json:"campaignIds"`
	NodeIDs     []string `json:"nodeIds"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`

	// Legacy activity fields. They are intentionally kept so old state.json and
	// the welfare extension can keep reading existing records during migration.
	ProjectCode      string        `json:"projectCode,omitempty"`
	CreatorID        string        `json:"creatorId,omitempty"`
	StartTime        string        `json:"startTime,omitempty"`
	EndTime          string        `json:"endTime,omitempty"`
	TotalStock       int           `json:"totalStock,omitempty"`
	ClaimedCount     int           `json:"claimedCount,omitempty"`
	PerUserLimit     int           `json:"perUserLimit,omitempty"`
	RewardType       string        `json:"rewardType,omitempty"`
	RewardContent    string        `json:"rewardContent,omitempty"`
	Enabled          bool          `json:"enabled,omitempty"`
	NeedLogin        bool          `json:"needLogin,omitempty"`
	NeedBindIdentity bool          `json:"needBindIdentity,omitempty"`
	Rules            string        `json:"rules,omitempty"`
	RewardItems      []RewardItem  `json:"rewardItems,omitempty"`
	ClaimRecords     []ClaimRecord `json:"claimRecords,omitempty"`
}

type ProjectOverview struct {
	ID           string `json:"id"`
	ProjectCode  string `json:"projectCode"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	TotalStock   int    `json:"totalStock"`
	ClaimedCount int    `json:"claimedCount"`
	Remaining    int    `json:"remaining"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	Enabled      bool   `json:"enabled"`
	ClaimURL     string `json:"claimUrl"`
	CreatedAt    string `json:"createdAt"`
	NodeCount    int    `json:"nodeCount"`
}

type ProjectDetail struct {
	Project
	Status    string `json:"status"`
	Remaining int    `json:"remaining"`
	ClaimURL  string `json:"claimUrl"`
}

type Campaign struct {
	ID                    string   `json:"id"`
	ProjectID             string   `json:"projectId"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Status                string   `json:"status"`
	TotalStock            int      `json:"totalStock"`
	ClaimedCount          int      `json:"claimedCount"`
	RemainingCount        int      `json:"remainingCount"`
	StartAt               string   `json:"startAt"`
	EndAt                 string   `json:"endAt"`
	AllowRepeat           bool     `json:"allowRepeat"`
	PerUserLimit          int      `json:"perUserLimit"`
	PerIPLimit            int      `json:"perIPLimit"`
	PerDeviceLimit        int      `json:"perDeviceLimit"`
	RequireCaptchaDefault bool     `json:"requireCaptchaDefault"`
	NodeIDs               []string `json:"nodeIds"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`

	// Legacy-compatible aliases used by old pages and public claim responses.
	ProjectCode string `json:"projectCode,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
	Enabled     bool   `json:"enabled"`
	Rules       string `json:"rules,omitempty"`
}

type RewardItem struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	ClaimedBy string `json:"claimedBy,omitempty"`
	ClaimedAt string `json:"claimedAt,omitempty"`
}

type CDK struct {
	ID                string `json:"id"`
	CampaignID        string `json:"campaignId"`
	Code              string `json:"code"`
	Status            string `json:"status"`
	ClaimedByRecordID string `json:"claimedByRecordId,omitempty"`
	ClaimedAt         string `json:"claimedAt,omitempty"`
	NodeID            string `json:"nodeId,omitempty"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type DistributionNode struct {
	ID             string `json:"id"`
	ProjectID      string `json:"projectId"`
	CampaignID     string `json:"campaignId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Status         string `json:"status"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	ButtonText     string `json:"buttonText"`
	RequireCaptcha bool   `json:"requireCaptcha"`
	ShowStock      bool   `json:"showStock"`
	ShowEndTime    bool   `json:"showEndTime"`
	Visits         int    `json:"visits"`
	UniqueVisitors int    `json:"uniqueVisitors"`
	Claims         int    `json:"claims"`
	FailedClaims   int    `json:"failedClaims"`
	LastVisitedAt  string `json:"lastVisitedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`

	// Legacy options retained for existing node records.
	Limit              int  `json:"limit,omitempty"`
	IPLimitEnabled     bool `json:"ipLimitEnabled,omitempty"`
	DeviceLimitEnabled bool `json:"deviceLimitEnabled,omitempty"`
}

type ClaimRecord struct {
	ID             string   `json:"id"`
	CampaignID     string   `json:"campaignId"`
	NodeID         string   `json:"nodeId"`
	ProjectID      string   `json:"projectId"`
	CDKID          string   `json:"cdkId"`
	Code           string   `json:"code"`
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	IP             string   `json:"ip"`
	UserAgent      string   `json:"userAgent"`
	Fingerprint    string   `json:"fingerprint"`
	IdempotencyKey string   `json:"idempotencyKey"`
	ClaimToken     string   `json:"claimToken"`
	HCaptchaPassed bool     `json:"hcaptchaPassed"`
	RiskHit        bool     `json:"riskHit"`
	RiskRuleIDs    []string `json:"riskRuleIds"`
	CreatedAt      string   `json:"createdAt"`

	// Legacy aliases.
	UserID        string `json:"userId,omitempty"`
	RewardItemID  string `json:"rewardItemId,omitempty"`
	RewardContent string `json:"rewardContent,omitempty"`
}

type RiskRule struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	Action    string                 `json:"action"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

type BlacklistItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	Reason    string `json:"reason"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CaptchaConfig struct {
	Provider                  string `json:"provider"`
	Enabled                   bool   `json:"enabled"`
	HCaptchaSiteKeyConfigured bool   `json:"hcaptchaSiteKeyConfigured"`
	HCaptchaSecretConfigured  bool   `json:"hcaptchaSecretConfigured"`
	LastTestStatus            string `json:"lastTestStatus"`
	LastTestMessage           string `json:"lastTestMessage"`
	LastTestAt                string `json:"lastTestAt"`
}

type SystemLog struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Level      string                 `json:"level"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Actor      string                 `json:"actor"`
	IP         string                 `json:"ip"`
	TargetType string                 `json:"targetType"`
	TargetID   string                 `json:"targetId"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  string                 `json:"createdAt"`
}

type ExportTask struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Filename   string                 `json:"filename"`
	FilePath   string                 `json:"filePath"`
	Filter     map[string]interface{} `json:"filter"`
	Error      string                 `json:"error"`
	CreatedAt  string                 `json:"createdAt"`
	FinishedAt string                 `json:"finishedAt"`
}

type Settings struct {
	SystemName       string `json:"systemName"`
	BrandName        string `json:"brandName"`
	BrandEnglishName string `json:"brandEnglishName"`
	LogoText         string `json:"logoText"`
	PublicBaseURL    string `json:"publicBaseURL"`
	StorageMode      string `json:"storageMode"`
	RedisEnabled     bool   `json:"redisEnabled"`
	RabbitMQEnabled  bool   `json:"rabbitmqEnabled"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type PageResult struct {
	Items    interface{} `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type ClaimPageInfo struct {
	NodeID            string    `json:"nodeId,omitempty"`
	NodeSlug          string    `json:"nodeSlug,omitempty"`
	ProjectCode       string    `json:"projectCode"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	StartTime         string    `json:"startTime"`
	EndTime           string    `json:"endTime"`
	StartAt           string    `json:"startAt,omitempty"`
	EndAt             string    `json:"endAt,omitempty"`
	TotalStock        int       `json:"totalStock"`
	ClaimedCount      int       `json:"claimedCount"`
	Remaining         int       `json:"remaining"`
	RemainingCount    int       `json:"remainingCount"`
	PerUserLimit      int       `json:"perUserLimit"`
	RewardType        string    `json:"rewardType"`
	NeedLogin         bool      `json:"needLogin"`
	Rules             string    `json:"rules"`
	Enabled           bool      `json:"enabled"`
	Status            string    `json:"status"`
	ButtonText        string    `json:"buttonText,omitempty"`
	ShowStock         bool      `json:"showStock"`
	ShowEndTime       bool      `json:"showEndTime"`
	RequireCaptcha    bool      `json:"requireCaptcha"`
	UserClaimed       bool      `json:"userClaimed"`
	UserRewardContent string    `json:"userRewardContent,omitempty"`
	UserClaimedAt     string    `json:"userClaimedAt,omitempty"`
	UserInfo          *UserInfo `json:"userInfo,omitempty"`
}

type ClaimRequest struct {
	Fingerprint string `json:"fingerprint"`
}

type PublicClaimRequest struct {
	CaptchaProvider string `json:"captchaProvider"`
	HCaptchaToken   string `json:"hcaptchaToken"`
	CaptchaToken    string `json:"captchaToken"`
	Fingerprint     string `json:"fingerprint"`
}

type ClaimResponse struct {
	Success       bool   `json:"success"`
	ClaimID       string `json:"claimId,omitempty"`
	ClaimToken    string `json:"claimToken,omitempty"`
	CampaignID    string `json:"campaignId,omitempty"`
	NodeID        string `json:"nodeId,omitempty"`
	Code          string `json:"code,omitempty"`
	RewardContent string `json:"rewardContent"`
	ClaimedAt     string `json:"claimedAt"`
	Message       string `json:"message"`
}

type UserInfo struct {
	UserID          string `json:"userId"`
	Username        string `json:"username"`
	Role            string `json:"role,omitempty"`
	Avatar          string `json:"avatar,omitempty"`
	Nickname        string `json:"nickname,omitempty"`
	Email           string `json:"email,omitempty"`
	CommunityUserID string `json:"communityUserId,omitempty"`
}

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type CreateCampaignRequest struct {
	ProjectID             string   `json:"projectId"`
	ProjectIDSnake        string   `json:"project_id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	StartAt               string   `json:"startAt"`
	EndAt                 string   `json:"endAt"`
	StartTime             string   `json:"startTime"`
	EndTime               string   `json:"endTime"`
	AllowRepeat           bool     `json:"allowRepeat"`
	PerUserLimit          int      `json:"perUserLimit"`
	PerIPLimit            int      `json:"perIPLimit"`
	PerDeviceLimit        int      `json:"perDeviceLimit"`
	RequireCaptchaDefault bool     `json:"requireCaptchaDefault"`
	Enabled               bool     `json:"enabled"`
	RewardList            []string `json:"rewardList"`
	Rules                 string   `json:"rules"`
}

type UpdateCampaignRequest = CreateCampaignRequest

type CreateNodeRequest struct {
	ProjectID          string `json:"projectId"`
	CampaignID         string `json:"campaignId"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Status             string `json:"status"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	ButtonText         string `json:"buttonText"`
	RequireCaptcha     bool   `json:"requireCaptcha"`
	ShowStock          bool   `json:"showStock"`
	ShowEndTime        bool   `json:"showEndTime"`
	Limit              int    `json:"limit"`
	IPLimitEnabled     bool   `json:"ipLimitEnabled"`
	DeviceLimitEnabled bool   `json:"deviceLimitEnabled"`
}

type UpdateNodeRequest struct {
	ProjectID          *string `json:"projectId,omitempty"`
	CampaignID         *string `json:"campaignId,omitempty"`
	Name               *string `json:"name,omitempty"`
	Slug               *string `json:"slug,omitempty"`
	Status             *string `json:"status,omitempty"`
	Title              *string `json:"title,omitempty"`
	Description        *string `json:"description,omitempty"`
	ButtonText         *string `json:"buttonText,omitempty"`
	RequireCaptcha     *bool   `json:"requireCaptcha,omitempty"`
	ShowStock          *bool   `json:"showStock,omitempty"`
	ShowEndTime        *bool   `json:"showEndTime,omitempty"`
	Limit              *int    `json:"limit,omitempty"`
	IPLimitEnabled     *bool   `json:"ipLimitEnabled,omitempty"`
	DeviceLimitEnabled *bool   `json:"deviceLimitEnabled,omitempty"`
}

type ImportCDKRequest struct {
	ProjectID      string   `json:"projectId"`
	ProjectIDSnake string   `json:"project_id"`
	CampaignID     string   `json:"campaignId"`
	Codes          []string `json:"codes"`
	Items          []string `json:"items"`
}

type ImportCDKResponse struct {
	Imported       int      `json:"imported"`
	Success        int      `json:"successCount"`
	Duplicates     int      `json:"duplicates"`
	UsedElsewhere  int      `json:"usedElsewhere"`
	Invalid        int      `json:"invalid"`
	Failed         int      `json:"failedCount"`
	Preview        []string `json:"preview"`
	RejectedSample []string `json:"rejectedSample,omitempty"`
}

type OnboardingStatus struct {
	HasProject            bool                   `json:"hasProject"`
	ProjectCount          int                    `json:"projectCount"`
	HasCampaign           bool                   `json:"hasCampaign"`
	CampaignCount         int                    `json:"campaignCount"`
	HasCdkStock           bool                   `json:"hasCdkStock"`
	CdkCount              int                    `json:"cdkCount"`
	HasDistributionNode   bool                   `json:"hasDistributionNode"`
	NodeCount             int                    `json:"nodeCount"`
	HasRiskConfig         bool                   `json:"hasRiskConfig"`
	HasPublicLink         bool                   `json:"hasPublicLink"`
	CurrentStep           int                    `json:"currentStep"`
	RecommendedProjectID  string                 `json:"recommendedProjectId,omitempty"`
	RecommendedCampaignID string                 `json:"recommendedCampaignId,omitempty"`
	RecommendedNodeID     string                 `json:"recommendedNodeId,omitempty"`
	RecommendedPublicLink string                 `json:"recommendedPublicLink,omitempty"`
	LatestProjectID       string                 `json:"latestProjectId,omitempty"`
	LatestCampaignID      string                 `json:"latestCampaignId,omitempty"`
	LatestStockCampaignID string                 `json:"latestStockCampaignId,omitempty"`
	LatestNodeID          string                 `json:"latestNodeId,omitempty"`
	LatestNodeLink        string                 `json:"latestNodeLink,omitempty"`
	NextActionHint        string                 `json:"nextActionHint,omitempty"`
	StepPrerequisites     map[string]bool        `json:"stepPrerequisites,omitempty"`
	RecommendedCampaign   map[string]interface{} `json:"recommendedCampaign,omitempty"`
	RecommendedProject    map[string]interface{} `json:"recommendedProject,omitempty"`
	RecommendedNode       map[string]interface{} `json:"recommendedNode,omitempty"`
}

type BatchIDsRequest struct {
	IDs []string `json:"ids"`
}

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

type ClaimMessage struct {
	ProjectID   string `json:"projectId"`
	NodeID      string `json:"nodeId,omitempty"`
	UserID      string `json:"userId"`
	Fingerprint string `json:"fingerprint"`
	IP          string `json:"ip"`
	ClaimedAt   string `json:"claimedAt"`
}

var (
	ErrNotFound                  = NewAppError(http.StatusNotFound, "NOT_FOUND", "资源不存在")
	ErrBadRequest                = NewAppError(http.StatusBadRequest, "BAD_REQUEST", "请求参数错误")
	ErrProjectClosed             = NewAppError(http.StatusForbidden, "PROJECT_DISABLED", "该项目已关闭")
	ErrNotStarted                = NewAppError(http.StatusBadRequest, "NOT_STARTED", "活动尚未开始")
	ErrEnded                     = NewAppError(http.StatusBadRequest, "ENDED", "活动已结束")
	ErrSoldOut                   = NewAppError(http.StatusConflict, "SOLD_OUT", "已领完")
	ErrAlreadyClaimed            = NewAppError(http.StatusConflict, "ALREADY_CLAIMED", "您已领取过")
	ErrLoginRequired             = NewAppError(http.StatusUnauthorized, "LOGIN_REQUIRED", "请先登录")
	ErrForbidden                 = NewAppError(http.StatusForbidden, "FORBIDDEN", "无权执行此操作")
	ErrCaptchaRequired           = NewAppError(http.StatusBadRequest, "CAPTCHA_REQUIRED", "请先完成人机验证")
	ErrHCaptchaNotConfigured     = NewAppError(http.StatusInternalServerError, "HCAPTCHA_NOT_CONFIGURED", "验证码服务未配置")
	ErrCaptchaInvalid            = NewAppError(http.StatusBadRequest, "CAPTCHA_INVALID", "人机验证失败，请刷新后重试")
	ErrCaptchaServiceUnavailable = NewAppError(http.StatusServiceUnavailable, "CAPTCHA_SERVICE_UNAVAILABLE", "验证码服务暂时不可用，请稍后再试")
	ErrCampaignCDKLocked         = NewAppError(http.StatusConflict, "CAMPAIGN_CDK_LOCKED", "该活动的 CDK 池已锁定，不允许重复导入")
	ErrCDKCodeUsedElsewhere      = NewAppError(http.StatusConflict, "CDK_CODE_USED_ELSEWHERE", "该 CDK 已被其他活动占用")
)
