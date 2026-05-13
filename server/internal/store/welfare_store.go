package store

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cdk-airdrop-station/server/internal/model"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// WelfareStore 福利业务存储
type WelfareStore struct {
	path        string
	mu          sync.RWMutex
	data        model.SystemData
	redisClient *redis.Client
	amqpConn    *amqp.Connection
	amqpChannel *amqp.Channel
	claimer     *RedisClaimer
	riskEngine  *RiskEngine
}

// NewWelfareStore 创建福利存储实例
func NewWelfareStore(path, redisURL, rabbitmqURL string) (*WelfareStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	store := &WelfareStore{path: path}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store.data = model.SystemData{
				Projects: make(map[string]*model.Project),
				Users:    make(map[string]*model.User),
			}
			return store, store.saveLocked()
		}
		return nil, err
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		store.data = model.SystemData{
			Projects: make(map[string]*model.Project),
			Users:    make(map[string]*model.User),
		}
		return store, store.saveLocked()
	}

	if err := json.Unmarshal(content, &store.data); err != nil {
		store.data = model.SystemData{
			Projects: make(map[string]*model.Project),
			Users:    make(map[string]*model.User),
		}
		return store, store.saveLocked()
	}

	if store.data.Projects == nil {
		store.data.Projects = make(map[string]*model.Project)
	}
	if store.data.Users == nil {
		store.data.Users = make(map[string]*model.User)
	}

	// 初始化 Redis
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			store.redisClient = redis.NewClient(opt)
			store.claimer = NewRedisClaimer(store.redisClient)
			store.riskEngine = NewRiskEngine(store.redisClient)

			// 同步所有 active 项目的库存到 Redis
			for _, p := range store.data.Projects {
				if p.Enabled {
					store.claimer.SyncStockToRedis(context.Background(), p.ProjectCode, p.TotalStock-p.ClaimedCount)
				}
			}
		}
	}

	// 初始化 RabbitMQ
	if rabbitmqURL != "" {
		conn, err := amqp.Dial(rabbitmqURL)
		if err == nil {
			ch, err := conn.Channel()
			if err == nil {
				store.amqpConn = conn
				store.amqpChannel = ch
				_, _ = ch.QueueDeclare("zenspulse_claims", true, false, false, false, nil)
			}
		}
	}

	return store, nil
}

// CreateWelfare 创建福利项目
func (s *WelfareStore) CreateWelfare(req model.CreateWelfareRequest, creatorID string) (*model.WelfareProject, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.NewAppError(http.StatusBadRequest, "name_required", "福利名称不能为空")
	}

	if len(req.RewardList) == 0 {
		return nil, model.NewAppError(http.StatusBadRequest, "reward_required", "请提供至少一个兑换码")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	id := MakeRandomString(12)
	code := MakeRandomString(8)

	perUserLimit := req.PerUserLimit
	if perUserLimit <= 0 {
		perUserLimit = 1
	}

	// 处理 CDK 列表，去重
	rewardItems := make([]model.RewardItem, 0, len(req.RewardList))
	seen := make(map[string]bool)
	for _, content := range req.RewardList {
		content = strings.TrimSpace(content)
		if content == "" || seen[content] {
			continue
		}
		seen[content] = true
		rewardItems = append(rewardItems, model.RewardItem{
			ID:      MakeRandomString(8),
			Content: content,
			Status:  "unused",
		})
	}

	project := &model.WelfareProject{
		Project: model.Project{
			ID:            id,
			ProjectCode:   code,
			CreatorID:     creatorID,
			Name:          strings.TrimSpace(req.Name),
			Description:   req.Description,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			TotalStock:    len(rewardItems),
			ClaimedCount:  0,
			PerUserLimit:  perUserLimit,
			RewardType:    "cdk_list",
			Enabled:       req.Enabled,
			NeedLogin:     req.NeedLogin,
			Rules:         req.Rules,
			CreatedAt:     now,
			UpdatedAt:     now,
			RewardItems:   rewardItems,
			ClaimRecords:  []model.ClaimRecord{},
		},
		ForumPostID:          req.ForumPostID,
		RequireReply:         req.RequireReply,
		RequireLike:          req.RequireLike,
		RequireFollow:        req.RequireFollow,
		MinReplyLength:       req.MinReplyLength,
		PublisherCredit:      70, // 初始信用分
		IPLimitPerProject:    req.IPLimitPerProject,
		DeviceLimitPerProject: req.DeviceLimitPerProject,
	}

	// 存储
	s.data.Projects[id] = &project.Project
	if err := s.saveLocked(); err != nil {
		delete(s.data.Projects, id)
		return nil, err
	}

	// 同步到 Redis
	if s.claimer != nil {
		s.claimer.SyncStockToRedis(context.Background(), code, len(rewardItems))
	}

	return project, nil
}

// GetWelfarePageInfo 获取福利页面信息
func (s *WelfareStore) GetWelfarePageInfo(code, userID, fingerprint string) (*model.WelfarePageInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := s.findByCode(code)
	if p == nil {
		return nil, model.ErrNotFound
	}

	info := &model.WelfarePageInfo{
		ProjectCode:  p.ProjectCode,
		Name:         p.Name,
		Description:  p.Description,
		StartTime:    p.StartTime,
		EndTime:      p.EndTime,
		TotalStock:   p.TotalStock,
		ClaimedCount: p.ClaimedCount,
		Remaining:    s.calcRemaining(p),
		PerUserLimit: p.PerUserLimit,
		NeedLogin:    p.NeedLogin,
		Rules:        p.Rules,
		Enabled:      p.Enabled,
		Status:       s.calcStatus(p),
	}

	// 检查当前用户是否已领取
	if userID != "" || fingerprint != "" {
		for _, record := range p.ClaimRecords {
			if (userID != "" && record.UserID == userID) ||
				(fingerprint != "" && record.Fingerprint == fingerprint) {
				info.UserClaimed = true
				info.UserRewardContent = record.RewardContent
				info.UserClaimedAt = record.CreatedAt
				break
			}
		}
	}

	return info, nil
}

// ExecuteClaim 执行领取
func (s *WelfareStore) ExecuteClaim(
	ctx context.Context,
	code string,
	userID string,
	fingerprint string,
	ip string,
	ua string,
) (*model.WelfareClaimResponse, error) {

	// 如果有 Redis，使用高并发流程
	if s.claimer != nil {
		return s.claimWithRedis(ctx, code, userID, fingerprint, ip, ua)
	}

	// 否则使用互斥锁
	return s.claimWithMutex(code, userID, fingerprint, ip, ua)
}

// claimWithRedis 使用 Redis 高并发领取
func (s *WelfareStore) claimWithRedis(
	ctx context.Context,
	code string,
	userID string,
	fingerprint string,
	ip string,
	ua string,
) (*model.WelfareClaimResponse, error) {

	// 1. 基础检查
	s.mu.RLock()
	p := s.findByCode(code)
	if p == nil {
		s.mu.RUnlock()
		return nil, model.ErrNotFound
	}

	if !p.Enabled {
		s.mu.RUnlock()
		return nil, model.ErrProjectClosed
	}

	status := s.calcStatus(p)
	if status == "upcoming" {
		s.mu.RUnlock()
		return nil, model.ErrNotStarted
	}
	if status == "ended" {
		s.mu.RUnlock()
		return nil, model.ErrEnded
	}
	if status == "disabled" {
		s.mu.RUnlock()
		return nil, model.ErrProjectClosed
	}
	if p.NeedLogin && userID == "" {
		s.mu.RUnlock()
		return nil, model.ErrLoginRequired
	}
	s.mu.RUnlock()

	// 2. 风控检查
	if s.riskEngine != nil {
		assessment := s.riskEngine.EvaluateRisk(ctx, nil, userID, fingerprint, code)
		if assessment.Level == "blocked" {
			return &model.WelfareClaimResponse{
				Success: false,
				Message: "领取请求被拒绝，请稍后再试",
			}, nil
		}
	}

	// 3. Redis 原子领取
	result, err := s.claimer.ExecuteClaim(ctx, code, userID, ip, fingerprint, 0, 0)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return &model.WelfareClaimResponse{
			Success: false,
			Message: result.Error,
		}, nil
	}

	// 4. 获取奖励内容
	s.mu.Lock()
	p = s.findByCode(code)
	if p == nil {
		s.mu.Unlock()
		return nil, model.ErrNotFound
	}

	var rewardContent string
	var rewardItemID string

	// 从 CDK 池中取一个未使用的
	for i, item := range p.RewardItems {
		if item.Status == "unused" {
			rewardContent = item.Content
			rewardItemID = item.ID
			p.RewardItems[i].Status = "claimed"
			p.RewardItems[i].ClaimedBy = userID
			if userID == "" {
				p.RewardItems[i].ClaimedBy = fingerprint
			}
			p.RewardItems[i].ClaimedAt = time.Now().Format(time.RFC3339)
			break
		}
	}

	if rewardContent == "" {
		s.mu.Unlock()
		return &model.WelfareClaimResponse{
			Success: false,
			Message: "奖励已发完",
		}, nil
	}

	// 记录领取
	record := model.ClaimRecord{
		ID:            MakeRandomString(12),
		ProjectID:     p.ID,
		UserID:        userID,
		Fingerprint:   fingerprint,
		RewardItemID:  rewardItemID,
		RewardContent: rewardContent,
		IP:            ip,
		UserAgent:     ua,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	p.ClaimRecords = append(p.ClaimRecords, record)
	p.ClaimedCount++
	s.saveLocked()
	s.mu.Unlock()

	return &model.WelfareClaimResponse{
		Success:       true,
		RewardContent: rewardContent,
		ClaimedAt:     record.CreatedAt,
		Message:       "领取成功",
		Remaining:     result.Remaining,
	}, nil
}

// claimWithMutex 使用互斥锁领取
func (s *WelfareStore) claimWithMutex(
	code string,
	userID string,
	fingerprint string,
	ip string,
	ua string,
) (*model.WelfareClaimResponse, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.findByCode(code)
	if p == nil {
		return nil, model.ErrNotFound
	}

	if !p.Enabled {
		return nil, model.ErrProjectClosed
	}

	status := s.calcStatus(p)
	switch status {
	case "upcoming":
		return nil, model.ErrNotStarted
	case "ended":
		return nil, model.ErrEnded
	case "soldout":
		return nil, model.ErrSoldOut
	case "disabled":
		return nil, model.ErrProjectClosed
	}

	if p.NeedLogin && userID == "" {
		return nil, model.ErrLoginRequired
	}

	// 检查领取次数
	claimCount := 0
	for _, record := range p.ClaimRecords {
		if (userID != "" && record.UserID == userID) ||
			(fingerprint != "" && record.Fingerprint == fingerprint) {
			claimCount++
		}
	}
	if claimCount >= p.PerUserLimit {
		return nil, model.ErrAlreadyClaimed
	}

	// 检查库存
	remaining := s.calcRemaining(p)
	if remaining <= 0 {
		return nil, model.ErrSoldOut
	}

	// 从 CDK 池中取一个未使用的
	var rewardContent string
	var rewardItemID string

	for i, item := range p.RewardItems {
		if item.Status == "unused" {
			rewardContent = item.Content
			rewardItemID = item.ID
			p.RewardItems[i].Status = "claimed"
			p.RewardItems[i].ClaimedBy = userID
			if userID == "" {
				p.RewardItems[i].ClaimedBy = fingerprint
			}
			p.RewardItems[i].ClaimedAt = time.Now().Format(time.RFC3339)
			break
		}
	}

	if rewardContent == "" {
		return nil, model.ErrSoldOut
	}

	// 创建领取记录
	record := model.ClaimRecord{
		ID:            MakeRandomString(12),
		ProjectID:     p.ID,
		UserID:        userID,
		Fingerprint:   fingerprint,
		RewardItemID:  rewardItemID,
		RewardContent: rewardContent,
		IP:            ip,
		UserAgent:     ua,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	p.ClaimRecords = append(p.ClaimRecords, record)
	p.ClaimedCount++

	if err := s.saveLocked(); err != nil {
		p.ClaimRecords = p.ClaimRecords[:len(p.ClaimRecords)-1]
		p.ClaimedCount--
		return nil, model.NewAppError(http.StatusInternalServerError, "save_failed", "保存领取记录失败")
	}

	return &model.WelfareClaimResponse{
		Success:       true,
		RewardContent: rewardContent,
		ClaimedAt:     record.CreatedAt,
		Message:       "领取成功",
		Remaining:     remaining - 1,
	}, nil
}

// ListWelfares 列出所有福利
func (s *WelfareStore) ListWelfares() []model.ProjectOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]model.ProjectOverview, 0, len(s.data.Projects))
	for _, p := range s.data.Projects {
		list = append(list, model.ProjectOverview{
			ID:           p.ID,
			ProjectCode:  p.ProjectCode,
			Name:         p.Name,
			Status:       s.calcStatus(p),
			TotalStock:   p.TotalStock,
			ClaimedCount: p.ClaimedCount,
			Remaining:    s.calcRemaining(p),
			StartTime:    p.StartTime,
			EndTime:      p.EndTime,
			Enabled:      p.Enabled,
			ClaimURL:     "/welfare/" + p.ProjectCode,
			CreatedAt:    p.CreatedAt,
		})
	}
	return list
}

// GetWelfareDetail 获取福利详情
func (s *WelfareStore) GetWelfareDetail(id string) (*model.ProjectDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.data.Projects[id]
	if !ok {
		return nil, model.ErrNotFound
	}

	return &model.ProjectDetail{
		Project:   *p,
		Status:    s.calcStatus(p),
		Remaining: s.calcRemaining(p),
		ClaimURL:  "/welfare/" + p.ProjectCode,
	}, nil
}

// DisableWelfare 停用福利
func (s *WelfareStore) DisableWelfare(id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.data.Projects[id]
	if !ok {
		return model.ErrNotFound
	}

	if p.CreatorID != "" && userID != "" && p.CreatorID != userID {
		return model.ErrForbidden
	}

	p.Enabled = false
	p.UpdatedAt = time.Now().Format(time.RFC3339)

	if s.claimer != nil {
		s.claimer.ResetProject(context.Background(), p.ProjectCode)
	}

	return s.saveLocked()
}

// GetClaimRecords 获取领取记录
func (s *WelfareStore) GetClaimRecords(id string) ([]model.ClaimRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.data.Projects[id]
	if !ok {
		return nil, model.ErrNotFound
	}

	records := make([]model.ClaimRecord, len(p.ClaimRecords))
	copy(records, p.ClaimRecords)

	// 反转，最新的在前
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

// ─── 辅助方法 ────────────────────────────────────────────

func (s *WelfareStore) findByCode(code string) *model.Project {
	for _, p := range s.data.Projects {
		if p.ProjectCode == code {
			return p
		}
	}
	return nil
}

func (s *WelfareStore) calcStatus(p *model.Project) string {
	if !p.Enabled {
		return "disabled"
	}

	now := time.Now()

	if p.StartTime != "" {
		start, err := time.Parse(time.RFC3339, p.StartTime)
		if err == nil && now.Before(start) {
			return "upcoming"
		}
	}

	if p.EndTime != "" {
		end, err := time.Parse(time.RFC3339, p.EndTime)
		if err == nil && now.After(end) {
			return "ended"
		}
	}

	if s.calcRemaining(p) <= 0 {
		return "soldout"
	}

	return "active"
}

func (s *WelfareStore) calcRemaining(p *model.Project) int {
	if p.RewardType == "cdk_list" {
		count := 0
		for _, item := range p.RewardItems {
			if item.Status == "unused" {
				count++
			}
		}
		return count
	}
	remaining := p.TotalStock - p.ClaimedCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *WelfareStore) saveLocked() error {
	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, payload, 0o644)
}

// Shutdown 关闭存储
func (s *WelfareStore) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.saveLocked()
}
