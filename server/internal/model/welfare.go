package model

import "time"

// ─── UGC 福利系统数据模型 ──────────────────────────────────

// WelfareProject 扩展 Project，增加论坛联动字段
type WelfareProject struct {
	Project

	// 主帖绑定
	ForumPostID    string `json:"forumPostId"`    // 关联的论坛帖子 ID
	RequireReply   bool   `json:"requireReply"`   // 是否要求回复后才能领
	RequireLike    bool   `json:"requireLike"`    // 是否要求点赞后才能领
	RequireFollow  bool   `json:"requireFollow"`  // 是否要求关注发布者
	MinReplyLength int    `json:"minReplyLength"` // 最低回复字数（0=不限）

	// 发布者信用
	PublisherCredit float64 `json:"publisherCredit"` // 发布者信用分
	ReportCount     int     `json:"reportCount"`     // 被举报次数

	// 设备/IP 限制
	IPLimitPerProject  int `json:"ipLimitPerProject"`  // 同 IP 本项目限领
	DeviceLimitPerProject int `json:"deviceLimitPerProject"` // 同设备本项目限领
}

// ClaimEligibility 领取资格校验结果
type ClaimEligibility struct {
	Eligible   bool   `json:"eligible"`
	Reason     string `json:"reason,omitempty"`
	NeedReply  bool   `json:"needReply,omitempty"`
	NeedLike   bool   `json:"needLike,omitempty"`
	NeedFollow bool   `json:"needFollow,omitempty"`
}

// UserBehavior 用户行为记录（用于风控）
type UserBehavior struct {
	UserID      string    `json:"userId"`
	IP          string    `json:"ip"`
	Fingerprint string    `json:"fingerprint"`
	Action      string    `json:"action"` // reply / like / claim / report
	ForumPostID string    `json:"forumPostId"`
	Timestamp   time.Time `json:"timestamp"`
	UserAgent   string    `json:"userAgent"`
}

// WelfareReport 福利举报
type WelfareReport struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"projectId"`
	ReporterID string    `json:"reporterId"`
	Reason     string    `json:"reason"` // invalid_cdk / scam / inappropriate
	Status     string    `json:"status"` // pending / resolved / dismissed
	CreatedAt  time.Time `json:"createdAt"`
}

// RiskAssessment 风险评估结果
type RiskAssessment struct {
	Score   float64  `json:"score"`   // 0-100，越高越危险
	Level   string   `json:"level"`   // low / medium / high / blocked
	Reasons []string `json:"reasons"`
}

// CreateWelfareRequest 创建福利请求
type CreateWelfareRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	StartTime            string   `json:"startTime"`
	EndTime              string   `json:"endTime"`
	PerUserLimit         int      `json:"perUserLimit"`
	Enabled              bool     `json:"enabled"`
	NeedLogin            bool     `json:"needLogin"`
	Rules                string   `json:"rules"`
	RewardList           []string `json:"rewardList"`           // CDK 列表
	ForumPostID          string   `json:"forumPostId"`          // 关联帖子 ID
	RequireReply         bool     `json:"requireReply"`
	RequireLike          bool     `json:"requireLike"`
	RequireFollow        bool     `json:"requireFollow"`
	MinReplyLength       int      `json:"minReplyLength"`
	IPLimitPerProject    int      `json:"ipLimitPerProject"`
	DeviceLimitPerProject int    `json:"deviceLimitPerProject"`
}

// WelfareClaimRequest 领取请求
type WelfareClaimRequest struct {
	ProjectCode string `json:"projectCode"`
	Fingerprint string `json:"fingerprint"`
}

// WelfareClaimResponse 领取响应
type WelfareClaimResponse struct {
	Success       bool   `json:"success"`
	RewardContent string `json:"rewardContent,omitempty"`
	ClaimedAt     string `json:"claimedAt,omitempty"`
	Message       string `json:"message"`
	Remaining     int    `json:"remaining"`
}

// WelfarePageInfo 福利页面公开信息
type WelfarePageInfo struct {
	ProjectCode      string   `json:"projectCode"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	StartTime        string   `json:"startTime"`
	EndTime          string   `json:"endTime"`
	TotalStock       int      `json:"totalStock"`
	ClaimedCount     int      `json:"claimedCount"`
	Remaining        int      `json:"remaining"`
	PerUserLimit     int      `json:"perUserLimit"`
	NeedLogin        bool     `json:"needLogin"`
	Rules            string   `json:"rules"`
	Enabled          bool     `json:"enabled"`
	Status           string   `json:"status"`
	ForumPostID      string   `json:"forumPostId"`
	RequireReply     bool     `json:"requireReply"`
	RequireLike      bool     `json:"requireLike"`
	RequireFollow    bool     `json:"requireFollow"`
	MinReplyLength   int      `json:"minReplyLength"`
	// 当前用户的领取状态
	UserClaimed       bool   `json:"userClaimed"`
	UserRewardContent string `json:"userRewardContent,omitempty"`
	UserClaimedAt     string `json:"userClaimedAt,omitempty"`
	// 领取资格
	Eligibility *ClaimEligibility `json:"eligibility,omitempty"`
}

// WelfareStats 福利统计
type WelfareStats struct {
	TotalWelfares   int     `json:"totalWelfares"`
	ActiveWelfares  int     `json:"activeWelfares"`
	TotalClaims     int     `json:"totalClaims"`
	TotalReports    int     `json:"totalReports"`
	AvgClaimSpeed   float64 `json:"avgClaimSpeed"`   // 平均领取速度（分钟/个）
	SuccessRate     float64 `json:"successRate"`      // CDK 有效率
}
