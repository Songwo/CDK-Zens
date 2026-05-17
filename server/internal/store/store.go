package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cdk-airdrop-station/server/internal/model"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	timeLayout              = time.RFC3339
	defaultSystemName       = "Zens-CDK"
	defaultBrandName        = "Zens-CDK"
	defaultBrandEnglishName = "Zens CDK Airdrop Hub"
	defaultLogoText         = "ZC"
	legacySystemName        = "缪盒空投台"
	legacyBrandEnglishName  = "MiuBox Airdrop Hub"
	legacyLogoText          = "MB"
)

var slugPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Store struct {
	path        string
	mu          sync.RWMutex
	data        model.SystemData
	db          *sql.DB
	dbDialect   string
	redisClient *redis.Client
	amqpConn    *amqp.Connection
	amqpChannel *amqp.Channel
}

func New(path, redisURL, rabbitmqURL string) (*Store, error) {
	return NewWithMySQL(path, "", redisURL, rabbitmqURL)
}

func NewWithMySQL(path, mysqlDSN, redisURL, rabbitmqURL string) (*Store, error) {
	return NewWithDatabase(path, "", mysqlDSN, redisURL, rabbitmqURL)
}

func NewWithDatabase(path, postgresDSN, mysqlDSN, redisURL, rabbitmqURL string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: path}

	// SQL mode: PostgreSQL is preferred for Docker one-click deployment.
	if postgresDSN != "" {
		db, err := InitPostgres(postgresDSN)
		if err != nil {
			return nil, fmt.Errorf("postgres init: %w", err)
		}
		s.db = db
		s.dbDialect = sqlDialectPostgres
	} else if mysqlDSN != "" {
		db, err := InitMySQL(mysqlDSN)
		if err != nil {
			return nil, fmt.Errorf("mysql init: %w", err)
		}
		s.db = db
		s.dbDialect = sqlDialectMySQL
	}

	// JSON fallback: load from file when no SQL database is configured.
	if s.db == nil {
		content, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if len(strings.TrimSpace(string(content))) > 0 {
			if err := json.Unmarshal(content, &s.data); err != nil {
				return nil, err
			}
		}
	}

	if redisURL != "" {
		if opt, err := redis.ParseURL(redisURL); err == nil {
			s.redisClient = redis.NewClient(opt)
		}
	}
	if rabbitmqURL != "" {
		if conn, err := amqp.Dial(rabbitmqURL); err == nil {
			s.amqpConn = conn
			if ch, err := conn.Channel(); err == nil {
				s.amqpChannel = ch
			}
		}
	}
	s.mu.Lock()
	s.migrateLocked()
	// Load from SQL after migration inits all maps
	if s.db != nil {
		s.loadFromSQL()
		if s.normalizeSettingsLocked(nowISO()) {
			_ = s.saveLocked()
		}
	}
	if s.db == nil {
		s.saveLocked()
	}
	s.mu.Unlock()
	return s, nil
}

func (s *Store) migrateLocked() {
	now := nowISO()
	if s.data.Projects == nil {
		s.data.Projects = map[string]*model.Project{}
	}
	if s.data.Campaigns == nil {
		s.data.Campaigns = map[string]*model.Campaign{}
	}
	if s.data.CDKs == nil {
		s.data.CDKs = map[string]*model.CDK{}
	}
	if s.data.Nodes == nil {
		s.data.Nodes = map[string]*model.DistributionNode{}
	}
	if s.data.ClaimRecords == nil {
		s.data.ClaimRecords = map[string]*model.ClaimRecord{}
	}
	if s.data.RiskRules == nil {
		s.data.RiskRules = map[string]*model.RiskRule{}
	}
	if s.data.Blacklist == nil {
		s.data.Blacklist = map[string]*model.BlacklistItem{}
	}
	if s.data.SystemLogs == nil {
		s.data.SystemLogs = map[string]*model.SystemLog{}
	}
	if s.data.ExportTasks == nil {
		s.data.ExportTasks = map[string]*model.ExportTask{}
	}
	if s.data.Users == nil {
		s.data.Users = map[string]*model.User{}
	}
	if s.data.Admins == nil {
		s.data.Admins = map[string]*model.Admin{}
	}
	for id, u := range s.data.Users {
		if _, ok := s.data.Admins[id]; !ok {
			cp := *u
			if cp.Role == "" {
				cp.Role = "admin"
			}
			s.data.Admins[id] = &cp
		}
	}
	s.normalizeSettingsLocked(now)
	if s.data.CaptchaConfig.Provider == "" {
		s.data.CaptchaConfig = model.CaptchaConfig{Provider: "hcaptcha", Enabled: true}
	}
	if len(s.data.RiskRules) == 0 {
		s.addDefaultRiskRulesLocked(now)
	}
	defaultProjectID := "default_project"
	if len(s.data.Projects) == 0 && len(s.data.Campaigns) > 0 {
		s.data.Projects[defaultProjectID] = &model.Project{ID: defaultProjectID, Name: "默认项目", Description: "旧活动自动迁移项目", Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
	}
	if _, ok := s.data.Projects[defaultProjectID]; !ok && len(s.data.Projects) > 0 {
		// Keep real project records, but create a neutral project for legacy campaigns.
		s.data.Projects[defaultProjectID] = &model.Project{ID: defaultProjectID, Name: "默认项目", Description: "旧活动自动迁移项目", Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
	}
	for id, p := range s.data.Projects {
		if p.ID == "" {
			p.ID = id
		}
		if p.CreatedAt == "" {
			p.CreatedAt = now
		}
		if p.UpdatedAt == "" {
			p.UpdatedAt = p.CreatedAt
		}
		if p.Status == "" {
			p.Status = model.StatusActive
		}
		if p.ProjectCode == "" && p.TotalStock == 0 && p.RewardType == "" {
			continue
		}
		if _, ok := s.data.Campaigns[p.ID]; !ok {
			c := &model.Campaign{
				ID: p.ID, ProjectID: defaultProjectID, Name: p.Name, Description: p.Description,
				Status: legacyCampaignStatus(p), TotalStock: p.TotalStock, ClaimedCount: p.ClaimedCount,
				StartAt: firstNonEmpty(p.StartTime, p.CreatedAt), EndAt: p.EndTime, AllowRepeat: false,
				PerUserLimit: maxInt(p.PerUserLimit, 1), PerIPLimit: 0, PerDeviceLimit: 1,
				RequireCaptchaDefault: false, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
				ProjectCode: p.ProjectCode, StartTime: p.StartTime, EndTime: p.EndTime, Enabled: p.Enabled, Rules: p.Rules,
			}
			s.data.Campaigns[c.ID] = c
		}
		for _, item := range p.RewardItems {
			if item.ID == "" {
				item.ID = MakeRandomString(8)
			}
			if _, ok := s.data.CDKs[item.ID]; !ok {
				status := item.Status
				if status == "" {
					status = model.StatusUnused
				}
				s.data.CDKs[item.ID] = &model.CDK{ID: item.ID, CampaignID: p.ID, Code: item.Content, Status: status, ClaimedAt: item.ClaimedAt, CreatedAt: p.CreatedAt, UpdatedAt: firstNonEmpty(item.ClaimedAt, p.UpdatedAt)}
			}
		}
		for _, old := range p.ClaimRecords {
			if old.ID == "" {
				old.ID = MakeRandomString(12)
			}
			if _, ok := s.data.ClaimRecords[old.ID]; !ok {
				rec := old
				rec.CampaignID = firstNonEmpty(rec.CampaignID, rec.ProjectID, p.ID)
				rec.ProjectID = defaultProjectID
				rec.CDKID = firstNonEmpty(rec.CDKID, rec.RewardItemID)
				rec.Code = firstNonEmpty(rec.Code, rec.RewardContent)
				rec.Status = firstNonEmpty(rec.Status, model.StatusSuccess)
				rec.RewardContent = firstNonEmpty(rec.RewardContent, rec.Code)
				rec.CreatedAt = firstNonEmpty(rec.CreatedAt, now)
				s.data.ClaimRecords[rec.ID] = &rec
				if rec.CDKID != "" {
					if cdk := s.data.CDKs[rec.CDKID]; cdk != nil {
						cdk.Status = model.StatusClaimed
						cdk.ClaimedAt = rec.CreatedAt
						cdk.ClaimedByRecordID = rec.ID
						cdk.NodeID = rec.NodeID
					}
				}
			}
		}
	}
	for id, n := range s.data.Nodes {
		if n.ID == "" {
			n.ID = id
		}
		if n.ProjectID == "" {
			n.ProjectID = defaultProjectID
		}
		if n.Status == "" {
			n.Status = model.StatusActive
		}
		if n.CreatedAt == "" {
			n.CreatedAt = now
		}
		if n.UpdatedAt == "" {
			n.UpdatedAt = n.CreatedAt
		}
		if n.ButtonText == "" {
			n.ButtonText = "立即领取"
		}
		if n.CampaignID != "" {
			if c := s.data.Campaigns[n.CampaignID]; c != nil {
				c.NodeIDs = appendUnique(c.NodeIDs, n.ID)
				n.ProjectID = firstNonEmpty(n.ProjectID, c.ProjectID)
			}
		}
	}
	for id, c := range s.data.Campaigns {
		if c.ID == "" {
			c.ID = id
		}
		if c.ProjectID == "" {
			c.ProjectID = defaultProjectID
		}
		if c.Status == "" {
			c.Status = model.StatusActive
		}
		if c.CreatedAt == "" {
			c.CreatedAt = now
		}
		if c.UpdatedAt == "" {
			c.UpdatedAt = c.CreatedAt
		}
		if c.StartAt == "" {
			c.StartAt = c.StartTime
		}
		if c.EndAt == "" {
			c.EndAt = c.EndTime
		}
		c.StartTime = c.StartAt
		c.EndTime = c.EndAt
		c.Enabled = c.Status != model.StatusPaused && c.Status != model.StatusEnded && c.Status != model.StatusArchived
		s.recalcCampaignLocked(c)
		if p := s.data.Projects[c.ProjectID]; p != nil {
			p.CampaignIDs = appendUnique(p.CampaignIDs, c.ID)
		}
	}
	for _, n := range s.data.Nodes {
		if p := s.data.Projects[n.ProjectID]; p != nil {
			p.NodeIDs = appendUnique(p.NodeIDs, n.ID)
		}
	}
}

func (s *Store) normalizeSettingsLocked(now string) bool {
	changed := false
	if s.data.Settings.SystemName == "" {
		s.data.Settings = model.Settings{
			SystemName:       defaultSystemName,
			BrandName:        defaultBrandName,
			BrandEnglishName: defaultBrandEnglishName,
			LogoText:         defaultLogoText,
			PublicBaseURL:    "http://127.0.0.1:5173",
			StorageMode:      s.defaultStorageMode(),
			RedisEnabled:     s.redisClient != nil,
			RabbitMQEnabled:  s.amqpChannel != nil,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		return true
	}
	if s.data.Settings.SystemName == legacySystemName {
		s.data.Settings.SystemName = defaultSystemName
		changed = true
	}
	if s.data.Settings.BrandName == "" || s.data.Settings.BrandName == legacySystemName {
		s.data.Settings.BrandName = defaultBrandName
		changed = true
	}
	if s.data.Settings.BrandEnglishName == "" || s.data.Settings.BrandEnglishName == legacyBrandEnglishName {
		s.data.Settings.BrandEnglishName = defaultBrandEnglishName
		changed = true
	}
	if s.data.Settings.LogoText == "" || s.data.Settings.LogoText == legacyLogoText {
		s.data.Settings.LogoText = defaultLogoText
		changed = true
	}
	if s.data.Settings.StorageMode == "" {
		s.data.Settings.StorageMode = s.defaultStorageMode()
		changed = true
	}
	if s.data.Settings.CreatedAt == "" {
		s.data.Settings.CreatedAt = now
		changed = true
	}
	if changed {
		s.data.Settings.UpdatedAt = now
	}
	return changed
}

func (s *Store) defaultStorageMode() string {
	if s.dbDialect != "" {
		return s.dbDialect
	}
	return "json"
}

func (s *Store) addDefaultRiskRulesLocked(now string) {
	s.data.RiskRules["rule_ip_minute"] = &model.RiskRule{ID: "rule_ip_minute", Name: "单 IP 每分钟限制", Type: "rate_limit", Enabled: true, Action: "block", Config: map[string]interface{}{"limit": float64(10), "windowSeconds": float64(60)}, CreatedAt: now, UpdatedAt: now}
	s.data.RiskRules["rule_device_campaign"] = &model.RiskRule{ID: "rule_device_campaign", Name: "单设备同活动限制", Type: "device_limit", Enabled: true, Action: "block", Config: map[string]interface{}{"limit": float64(1)}, CreatedAt: now, UpdatedAt: now}
}

func (s *Store) FindUserByUsername(username string) *model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.data.Admins {
		if strings.EqualFold(u.Username, username) {
			cp := *u
			return &cp
		}
	}
	for _, u := range s.data.Users {
		if strings.EqualFold(u.Username, username) {
			cp := *u
			return &cp
		}
	}
	return nil
}

func (s *Store) FindUserByID(id string) *model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.data.Admins[id]; ok {
		cp := *u
		return &cp
	}
	if u, ok := s.data.Users[id]; ok {
		cp := *u
		return &cp
	}
	return nil
}

func (s *Store) CreateUser(username, password string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Admins {
		if strings.EqualFold(u.Username, username) {
			return nil, model.NewAppError(http.StatusConflict, "USER_EXISTS", "用户名已存在")
		}
	}
	now := time.Now()
	u := &model.User{ID: MakeRandomString(12), Username: strings.TrimSpace(username), Password: password, Role: "admin", CreatedAt: now, UpdatedAt: now.Format(timeLayout)}
	s.data.Users[u.ID] = u
	s.data.Admins[u.ID] = u
	s.logLocked("operation", "info", "创建管理员", "创建管理员 "+u.Username, u.Username, "", "admin", u.ID, nil)
	return u, s.saveLocked()
}

func (s *Store) FindUserByCommunityID(communityUserID string) *model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.data.Admins {
		if u.CommunityUserID == communityUserID {
			cp := *u
			return &cp
		}
	}
	return nil
}

func (s *Store) CreateOrUpdateCommunityUser(communityUserID, username, avatar, nickname, email, role string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找已有用户
	for _, u := range s.data.Admins {
		if u.CommunityUserID == communityUserID {
			// 更新用户信息
			u.Username = username
			u.Avatar = avatar
			u.Nickname = nickname
			u.Email = email
			if role != "" {
				u.Role = role
			}
			u.UpdatedAt = time.Now().Format(timeLayout)
			s.logLocked("operation", "info", "社区SSO同步", "同步社区用户 "+username, username, "", "admin", u.ID, nil)
			return u, s.saveLocked()
		}
	}

	// 创建新用户
	now := time.Now()
	// 如果未传递有效角色，默认使用 user，因为 admin 权限很高
	finalRole := role
	if finalRole == "" {
		finalRole = "user"
	}
	u := &model.User{
		ID:              MakeRandomString(12),
		Username:        username,
		Password:        "sso_community",
		Role:            finalRole,
		CommunityUserID: communityUserID,
		Avatar:          avatar,
		Nickname:        nickname,
		Email:           email,
		CreatedAt:       now,
		UpdatedAt:       now.Format(timeLayout),
	}
	s.data.Users[u.ID] = u
	s.data.Admins[u.ID] = u
	s.logLocked("operation", "info", "社区SSO创建", "社区用户首次登录 "+username, username, "", "admin", u.ID, nil)
	return u, s.saveLocked()
}

func (s *Store) Dashboard() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dashboardLocked(nil)
}

// DashboardForUser returns dashboard data scoped to the user's projects.
// Admin users see all data; regular users see only their own projects.
func (s *Store) DashboardForUser(user *model.User) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user.Role == "admin" {
		return s.dashboardLocked(nil)
	}
	// Build set of project IDs owned by this user
	ownedProjects := map[string]bool{}
	for _, p := range s.data.Projects {
		if p.CreatorID == user.ID {
			ownedProjects[p.ID] = true
		}
	}
	return s.dashboardLocked(ownedProjects)
}

func (s *Store) OnboardingStatusForUser(user *model.User) model.OnboardingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	isAdmin := user.Role == "admin"
	canUseProject := func(p *model.Project) bool {
		return p != nil && (isAdmin || p.CreatorID == user.ID)
	}

	status := model.OnboardingStatus{
		StepPrerequisites: map[string]bool{},
	}
	latestProjectAt := ""
	latestCampaignAt := ""
	latestStockAt := ""
	latestNodeAt := ""

	for _, p := range s.data.Projects {
		if !canUseProject(p) {
			continue
		}
		status.ProjectCount++
		if p.CreatedAt >= latestProjectAt {
			latestProjectAt = p.CreatedAt
			status.LatestProjectID = p.ID
			status.RecommendedProjectID = p.ID
			status.RecommendedProject = map[string]interface{}{"id": p.ID, "name": p.Name, "createdAt": p.CreatedAt}
		}
	}
	status.HasProject = status.ProjectCount > 0

	for _, c := range s.data.Campaigns {
		p := s.data.Projects[c.ProjectID]
		if !canUseProject(p) {
			continue
		}
		status.CampaignCount++
		if c.CreatedAt >= latestCampaignAt {
			latestCampaignAt = c.CreatedAt
			status.LatestCampaignID = c.ID
			status.RecommendedCampaignID = c.ID
			status.RecommendedCampaign = s.campaignViewLocked(c)
		}
		if c.TotalStock > 0 && c.CreatedAt >= latestStockAt {
			latestStockAt = c.CreatedAt
			status.LatestStockCampaignID = c.ID
		}
	}
	status.HasCampaign = status.CampaignCount > 0

	for _, cdk := range s.data.CDKs {
		c := s.data.Campaigns[cdk.CampaignID]
		if c == nil || !canUseProject(s.data.Projects[c.ProjectID]) {
			continue
		}
		status.CdkCount++
	}
	status.HasCdkStock = status.CdkCount > 0

	for _, n := range s.data.Nodes {
		if !canUseProject(s.data.Projects[n.ProjectID]) {
			continue
		}
		status.NodeCount++
		if n.CreatedAt >= latestNodeAt {
			latestNodeAt = n.CreatedAt
			status.LatestNodeID = n.ID
			status.RecommendedNodeID = n.ID
			status.LatestNodeLink = "/claim/" + n.Slug
			status.RecommendedPublicLink = status.LatestNodeLink
			status.RecommendedNode = s.nodeViewLocked(n)
		}
		if n.Status == model.StatusActive && n.Slug != "" {
			status.HasPublicLink = true
		}
	}
	status.HasDistributionNode = status.NodeCount > 0

	for _, r := range s.data.RiskRules {
		if r.Enabled {
			status.HasRiskConfig = true
			break
		}
	}
	if !status.HasRiskConfig {
		status.HasRiskConfig = s.data.CaptchaConfig.Enabled
	}

	switch {
	case !status.HasProject:
		status.CurrentStep = 1
		status.NextActionHint = "请先创建项目"
	case !status.HasCampaign:
		status.CurrentStep = 2
		status.NextActionHint = "请创建活动"
	case !status.HasCdkStock:
		status.CurrentStep = 3
		status.NextActionHint = "请导入 CDK"
	case !status.HasDistributionNode:
		status.CurrentStep = 4
		status.NextActionHint = "请创建分发节点"
	case !status.HasRiskConfig:
		status.CurrentStep = 5
		status.NextActionHint = "请配置验证码或风控"
	case !status.HasPublicLink:
		status.CurrentStep = 6
		status.NextActionHint = "请复制链接并测试"
	default:
		status.CurrentStep = 7
		status.NextActionHint = "流程已完成"
	}
	status.StepPrerequisites["campaign"] = status.HasProject
	status.StepPrerequisites["cdk"] = status.HasCampaign
	status.StepPrerequisites["node"] = status.HasCampaign && status.HasCdkStock
	status.StepPrerequisites["link"] = status.HasDistributionNode
	return status
}

func (s *Store) dashboardLocked(scopeProjectIDs map[string]bool) map[string]interface{} {
	stats := map[string]interface{}{"activeCampaigns": 0, "totalCampaigns": 0, "totalStock": 0, "claimedCount": 0, "remainingCount": 0, "todayClaims": 0, "totalNodes": 0, "abnormalNodes": 0, "successRate": 0.0, "failedClaims": 0}
	campaigns := []map[string]interface{}{}
	nodes := []map[string]interface{}{}
	claims := []model.ClaimRecord{}
	alerts := []map[string]string{}
	for _, c := range s.data.Campaigns {
		if scopeProjectIDs != nil && !scopeProjectIDs[c.ProjectID] {
			continue
		}
		cc := s.campaignViewLocked(c)
		stats["totalCampaigns"] = stats["totalCampaigns"].(int) + 1
		if cc["status"] == model.StatusActive {
			stats["activeCampaigns"] = stats["activeCampaigns"].(int) + 1
		}
		stats["totalStock"] = stats["totalStock"].(int) + c.TotalStock
		stats["claimedCount"] = stats["claimedCount"].(int) + c.ClaimedCount
		stats["remainingCount"] = stats["remainingCount"].(int) + c.RemainingCount
		campaigns = append(campaigns, cc)
		if c.TotalStock > 0 && c.RemainingCount <= int(math.Ceil(float64(c.TotalStock)*0.1)) {
			alerts = append(alerts, map[string]string{"type": "stock_low", "level": "warning", "title": "库存不足", "message": c.Name + " 剩余库存偏低"})
		}
	}
	for _, n := range s.data.Nodes {
		if scopeProjectIDs != nil && !scopeProjectIDs[n.ProjectID] {
			continue
		}
		stats["totalNodes"] = stats["totalNodes"].(int) + 1
		nodes = append(nodes, s.nodeViewLocked(n))
		if n.Status != model.StatusActive {
			stats["abnormalNodes"] = stats["abnormalNodes"].(int) + 1
		}
	}
	for _, r := range s.data.ClaimRecords {
		if scopeProjectIDs != nil && !scopeProjectIDs[r.ProjectID] {
			continue
		}
		claims = append(claims, *r)
		if isToday(r.CreatedAt) && r.Status == model.StatusSuccess {
			stats["todayClaims"] = stats["todayClaims"].(int) + 1
		}
		if r.Status == model.StatusFailed || r.Status == model.StatusBlocked {
			stats["failedClaims"] = stats["failedClaims"].(int) + 1
		}
	}
	if stats["totalStock"].(int) > 0 {
		stats["successRate"] = float64(stats["claimedCount"].(int)) / float64(stats["totalStock"].(int)) * 100
	}
	sort.Slice(campaigns, func(i, j int) bool {
		return fmt.Sprint(campaigns[i]["createdAt"]) > fmt.Sprint(campaigns[j]["createdAt"])
	})
	sort.Slice(nodes, func(i, j int) bool { return fmt.Sprint(nodes[i]["createdAt"]) > fmt.Sprint(nodes[j]["createdAt"]) })
	sort.Slice(claims, func(i, j int) bool { return claims[i].CreatedAt > claims[j].CreatedAt })
	return map[string]interface{}{"stats": stats, "campaigns": limitMaps(campaigns, 10), "nodes": limitMaps(nodes, 10), "recentClaims": limitClaims(claims, 10), "recentRecords": limitClaims(claims, 10), "alerts": alerts, "health": s.healthLocked()}
}

func (s *Store) Health() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthLocked()
}

func (s *Store) healthLocked() map[string]interface{} {
	info := map[string]interface{}{"app": "ok", "jsonStore": "ok", "mode": "json-compatible", "storageMode": "json", "stateFile": s.path, "redis": "disabled", "rabbitmq": "disabled", "brand": s.data.Settings.BrandEnglishName}
	if s.redisClient != nil {
		info["redis"] = "enabled"
	}
	if s.amqpChannel != nil {
		info["rabbitmq"] = "enabled"
	}
	return info
}

func (s *Store) ListProjectsPage(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	for _, p := range s.data.Projects {
		if p.ID == "default_project" && len(s.data.Projects) > 1 && kw == "" && q["status"] == "" { /* still show default */
		}
		if q["status"] != "" && q["status"] != "all" && p.Status != q["status"] {
			continue
		}
		if kw != "" && !containsAny(kw, p.Name, p.Description, p.ID) {
			continue
		}
		rows = append(rows, map[string]interface{}{"id": p.ID, "name": p.Name, "description": p.Description, "status": p.Status, "campaignIds": p.CampaignIDs, "nodeIds": p.NodeIDs, "campaignCount": len(p.CampaignIDs), "nodeCount": len(p.NodeIDs), "createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt, "creatorId": p.CreatorID})
	}
	return paginateMaps(rows, q)
}

// ListProjectsPageForUser returns projects scoped to the user.
// Admin users see all projects; regular users see only their own.
func (s *Store) ListProjectsPageForUser(q map[string]string, user *model.User) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	for _, p := range s.data.Projects {
		// Regular users can only see their own projects
		if user.Role != "admin" && p.CreatorID != user.ID {
			continue
		}
		if p.ID == "default_project" && len(s.data.Projects) > 1 && kw == "" && q["status"] == "" { /* still show default */
		}
		if q["status"] != "" && q["status"] != "all" && p.Status != q["status"] {
			continue
		}
		if kw != "" && !containsAny(kw, p.Name, p.Description, p.ID) {
			continue
		}
		rows = append(rows, map[string]interface{}{"id": p.ID, "name": p.Name, "description": p.Description, "status": p.Status, "campaignIds": p.CampaignIDs, "nodeIds": p.NodeIDs, "campaignCount": len(p.CampaignIDs), "nodeCount": len(p.NodeIDs), "createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt, "creatorId": p.CreatorID})
	}
	return paginateMaps(rows, q)
}

func (s *Store) CreateProject(req model.CreateProjectRequest, actor string) (*model.Project, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.NewAppError(http.StatusBadRequest, "NAME_REQUIRED", "项目名称不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowISO()
	p := &model.Project{ID: MakeRandomString(12), Name: strings.TrimSpace(req.Name), Description: req.Description, Status: model.StatusActive, CampaignIDs: []string{}, NodeIDs: []string{}, CreatedAt: now, UpdatedAt: now, CreatorID: actor}
	s.data.Projects[p.ID] = p
	s.logLocked("operation", "info", "创建项目", "创建项目 "+p.Name, actor, "", "project", p.ID, nil)
	return p, s.saveLocked()
}

func (s *Store) GetProject(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.data.Projects[id]
	if p == nil {
		return nil, model.ErrNotFound
	}
	return s.projectDetailLocked(p), nil
}

func (s *Store) UpdateProject(id string, req model.UpdateProjectRequest, actor string) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.data.Projects[id]
	if p == nil {
		return nil, model.ErrNotFound
	}
	if req.Name != nil {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	p.UpdatedAt = nowISO()
	s.logLocked("operation", "info", "编辑项目", "编辑项目 "+p.Name, actor, "", "project", p.ID, nil)
	return s.projectDetailLocked(p), s.saveLocked()
}

func (s *Store) DeleteProject(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.data.Projects[id]
	if p == nil {
		return model.ErrNotFound
	}
	if len(p.CampaignIDs) > 0 || len(p.NodeIDs) > 0 {
		return model.NewAppError(http.StatusConflict, "PROJECT_NOT_EMPTY", "请先解绑项目下活动和节点")
	}
	delete(s.data.Projects, id)
	s.logLocked("operation", "warn", "删除项目", "删除项目 "+p.Name, actor, "", "project", id, nil)
	return s.saveLocked()
}

func (s *Store) ArchiveProject(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.data.Projects[id]
	if p == nil {
		return model.ErrNotFound
	}
	p.Status = model.StatusArchived
	p.UpdatedAt = nowISO()
	s.logLocked("operation", "info", "归档项目", "归档项目 "+p.Name, actor, "", "project", id, nil)
	return s.saveLocked()
}

func (s *Store) BindCampaign(projectID, campaignID, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.data.Projects[projectID]
	c := s.data.Campaigns[campaignID]
	if p == nil || c == nil {
		return model.ErrNotFound
	}
	if old := s.data.Projects[c.ProjectID]; old != nil {
		old.CampaignIDs = removeString(old.CampaignIDs, campaignID)
	}
	c.ProjectID = projectID
	c.UpdatedAt = nowISO()
	p.CampaignIDs = appendUnique(p.CampaignIDs, campaignID)
	s.logLocked("operation", "info", "绑定活动", "项目绑定活动", actor, "", "project", projectID, map[string]interface{}{"campaignId": campaignID})
	return s.saveLocked()
}

func (s *Store) UnbindCampaign(projectID, campaignID, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.data.Projects[projectID]
	c := s.data.Campaigns[campaignID]
	if p == nil || c == nil {
		return model.ErrNotFound
	}
	p.CampaignIDs = removeString(p.CampaignIDs, campaignID)
	c.ProjectID = "default_project"
	c.UpdatedAt = nowISO()
	s.logLocked("operation", "info", "解绑活动", "项目解绑活动", actor, "", "project", projectID, map[string]interface{}{"campaignId": campaignID})
	return s.saveLocked()
}

func (s *Store) ProjectStats(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.data.Projects[id]
	if p == nil {
		return nil, model.ErrNotFound
	}
	stats := map[string]int{"campaigns": 0, "nodes": 0, "totalStock": 0, "claimed": 0, "remaining": 0}
	for _, cid := range p.CampaignIDs {
		if c := s.data.Campaigns[cid]; c != nil {
			stats["campaigns"]++
			stats["totalStock"] += c.TotalStock
			stats["claimed"] += c.ClaimedCount
			stats["remaining"] += c.RemainingCount
		}
	}
	for _, nid := range p.NodeIDs {
		if s.data.Nodes[nid] != nil {
			stats["nodes"]++
		}
	}
	return map[string]interface{}{"project": p, "stats": stats}, nil
}

func (s *Store) ListCampaigns(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	projectFilter := firstNonEmpty(q["projectId"], q["project_id"])
	for _, c := range s.data.Campaigns {
		view := s.campaignViewLocked(c)
		projectName := fmt.Sprint(view["projectName"])
		if q["status"] != "" && q["status"] != "all" && fmt.Sprint(view["status"]) != q["status"] {
			continue
		}
		if projectFilter != "" && c.ProjectID != projectFilter {
			continue
		}
		if kw != "" && !containsAny(kw, c.Name, c.Description, c.ID, c.ProjectCode, projectName) {
			continue
		}
		rows = append(rows, view)
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) ListCampaignsForUser(q map[string]string, user *model.User) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	projectFilter := firstNonEmpty(q["projectId"], q["project_id"])
	isAdmin := user.Role == "admin"
	for _, c := range s.data.Campaigns {
		p := s.data.Projects[c.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		view := s.campaignViewLocked(c)
		if q["status"] != "" && q["status"] != "all" && fmt.Sprint(view["status"]) != q["status"] {
			continue
		}
		if projectFilter != "" && c.ProjectID != projectFilter {
			continue
		}
		if kw != "" && !containsAny(kw, c.Name, c.Description, c.ID, c.ProjectCode, safeProjectName(p)) {
			continue
		}
		rows = append(rows, view)
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) CreateCampaign(req model.CreateCampaignRequest, actor string) (map[string]interface{}, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.NewAppError(http.StatusBadRequest, "NAME_REQUIRED", "活动名称不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	projectID := campaignRequestProjectID(req)
	if projectID == "" {
		if len(s.data.Projects) == 0 {
			return nil, model.NewAppError(http.StatusBadRequest, "PROJECT_REQUIRED", "请先创建项目")
		}
		return nil, model.NewAppError(http.StatusBadRequest, "PROJECT_REQUIRED", "请选择所属项目")
	}
	if s.data.Projects[projectID] == nil {
		return nil, model.NewAppError(http.StatusNotFound, "PROJECT_NOT_FOUND", "项目不存在，请先创建项目")
	}
	now := nowISO()
	status := model.StatusActive
	if !req.Enabled {
		status = model.StatusPaused
	}
	c := &model.Campaign{ID: MakeRandomString(12), ProjectID: projectID, Name: strings.TrimSpace(req.Name), Description: req.Description, Status: status, StartAt: firstNonEmpty(req.StartAt, req.StartTime), EndAt: firstNonEmpty(req.EndAt, req.EndTime), AllowRepeat: req.AllowRepeat, PerUserLimit: maxInt(req.PerUserLimit, 1), PerIPLimit: req.PerIPLimit, PerDeviceLimit: req.PerDeviceLimit, RequireCaptchaDefault: req.RequireCaptchaDefault, CreatedAt: now, UpdatedAt: now, ProjectCode: MakeRandomString(8), Enabled: status == model.StatusActive, Rules: req.Rules}
	c.StartTime = c.StartAt
	c.EndTime = c.EndAt
	s.data.Campaigns[c.ID] = c
	s.data.Projects[projectID].CampaignIDs = appendUnique(s.data.Projects[projectID].CampaignIDs, c.ID)
	if len(req.RewardList) > 0 {
		// 一活动一池：创建期写入的 CDK 同样做全局去重，避免与其他活动撞码。
		globalUsed := map[string]string{}
		for _, cdk := range s.data.CDKs {
			globalUsed[cdk.Code] = cdk.CampaignID
		}
		seen := map[string]bool{}
		for _, raw := range req.RewardList {
			code := strings.TrimSpace(raw)
			if code == "" || seen[code] {
				continue
			}
			if _, exists := globalUsed[code]; exists {
				continue
			}
			seen[code] = true
			s.addCDKLocked(c.ID, code, now)
			globalUsed[code] = c.ID
		}
	}
	s.recalcCampaignLocked(c)
	s.logLocked("operation", "info", "创建活动", "创建活动 "+c.Name, actor, "", "campaign", c.ID, nil)
	return s.campaignViewLocked(c), s.saveLocked()
}

func (s *Store) GetCampaign(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.data.Campaigns[id]
	if c == nil {
		return nil, model.ErrNotFound
	}
	return s.campaignViewLocked(c), nil
}

func (s *Store) UpdateCampaign(id string, req model.UpdateCampaignRequest, actor string) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.data.Campaigns[id]
	if c == nil {
		return nil, model.ErrNotFound
	}
	if strings.TrimSpace(req.Name) != "" {
		c.Name = strings.TrimSpace(req.Name)
	}
	c.Description = req.Description
	projectID := campaignRequestProjectID(req)
	if projectID != "" && projectID != c.ProjectID {
		if s.data.Projects[projectID] == nil {
			return nil, model.NewAppError(http.StatusNotFound, "PROJECT_NOT_FOUND", "项目不存在，请先创建项目")
		}
		if old := s.data.Projects[c.ProjectID]; old != nil {
			old.CampaignIDs = removeString(old.CampaignIDs, c.ID)
		}
		c.ProjectID = projectID
		s.data.Projects[c.ProjectID].CampaignIDs = appendUnique(s.data.Projects[c.ProjectID].CampaignIDs, c.ID)
	}
	if req.StartAt != "" || req.StartTime != "" {
		c.StartAt = firstNonEmpty(req.StartAt, req.StartTime)
		c.StartTime = c.StartAt
	}
	if req.EndAt != "" || req.EndTime != "" {
		c.EndAt = firstNonEmpty(req.EndAt, req.EndTime)
		c.EndTime = c.EndAt
	}
	c.AllowRepeat = req.AllowRepeat
	c.PerUserLimit = maxInt(req.PerUserLimit, 1)
	c.PerIPLimit = req.PerIPLimit
	c.PerDeviceLimit = req.PerDeviceLimit
	c.RequireCaptchaDefault = req.RequireCaptchaDefault
	c.Rules = req.Rules
	c.UpdatedAt = nowISO()
	s.recalcCampaignLocked(c)
	s.logLocked("operation", "info", "编辑活动", "编辑活动 "+c.Name, actor, "", "campaign", c.ID, nil)
	return s.campaignViewLocked(c), s.saveLocked()
}

func (s *Store) SetCampaignStatus(id, status, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.data.Campaigns[id]
	if c == nil {
		return model.ErrNotFound
	}
	c.Status = status
	c.Enabled = status == model.StatusActive
	c.UpdatedAt = nowISO()
	title := map[string]string{model.StatusPaused: "暂停活动", model.StatusActive: "恢复活动", model.StatusEnded: "结束活动"}[status]
	if title == "" {
		title = "修改活动状态"
	}
	s.logLocked("operation", "info", title, title+" "+c.Name, actor, "", "campaign", id, nil)
	return s.saveLocked()
}

func (s *Store) DeleteCampaign(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.data.Campaigns[id]
	if c == nil {
		return model.ErrNotFound
	}
	for _, cdk := range s.data.CDKs {
		if cdk.CampaignID == id && cdk.Status == model.StatusClaimed {
			return model.NewAppError(http.StatusConflict, "CAMPAIGN_HAS_CLAIMS", "活动已有领取记录，不能删除")
		}
	}
	delete(s.data.Campaigns, id)
	if p := s.data.Projects[c.ProjectID]; p != nil {
		p.CampaignIDs = removeString(p.CampaignIDs, id)
	}
	for _, n := range s.data.Nodes {
		if n.CampaignID == id {
			n.CampaignID = ""
			n.UpdatedAt = nowISO()
		}
	}
	for cid, cdk := range s.data.CDKs {
		if cdk.CampaignID == id {
			delete(s.data.CDKs, cid)
		}
	}
	s.logLocked("operation", "warn", "删除活动", "删除活动 "+c.Name, actor, "", "campaign", id, nil)
	return s.saveLocked()
}

func (s *Store) ListCDKs(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	projectFilter := firstNonEmpty(q["projectId"], q["project_id"])
	for _, cdk := range s.data.CDKs {
		if q["campaignId"] != "" && cdk.CampaignID != q["campaignId"] {
			continue
		}
		if q["status"] != "" && q["status"] != "all" && cdk.Status != q["status"] {
			continue
		}
		c := s.data.Campaigns[cdk.CampaignID]
		projectName := ""
		projectID := ""
		if c != nil {
			projectID = c.ProjectID
			projectName = safeProjectName(s.data.Projects[c.ProjectID])
		}
		if projectFilter != "" && projectID != projectFilter {
			continue
		}
		if kw != "" && !containsAny(kw, cdk.Code, cdk.ID, safeCampaignName(c), projectName) {
			continue
		}
		rows = append(rows, map[string]interface{}{"id": cdk.ID, "projectId": projectID, "projectName": projectName, "campaignId": cdk.CampaignID, "campaignName": safeCampaignName(c), "code": cdk.Code, "status": cdk.Status, "claimedByRecordId": cdk.ClaimedByRecordID, "claimedAt": cdk.ClaimedAt, "nodeId": cdk.NodeID, "createdAt": cdk.CreatedAt, "updatedAt": cdk.UpdatedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) ListCDKsForUser(q map[string]string, user *model.User) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	projectFilter := firstNonEmpty(q["projectId"], q["project_id"])
	isAdmin := user.Role == "admin"
	for _, cdk := range s.data.CDKs {
		c := s.data.Campaigns[cdk.CampaignID]
		projectName := ""
		projectID := ""
		if c != nil {
			projectID = c.ProjectID
			projectName = safeProjectName(s.data.Projects[c.ProjectID])
		}
		if !isAdmin {
			if c == nil {
				continue
			}
			p := s.data.Projects[c.ProjectID]
			if p == nil || p.CreatorID != user.ID {
				continue
			}
		}
		if q["campaignId"] != "" && cdk.CampaignID != q["campaignId"] {
			continue
		}
		if q["status"] != "" && q["status"] != "all" && cdk.Status != q["status"] {
			continue
		}
		if projectFilter != "" && projectID != projectFilter {
			continue
		}
		if kw != "" && !containsAny(kw, cdk.Code, cdk.ID, safeCampaignName(c), projectName) {
			continue
		}
		rows = append(rows, map[string]interface{}{"id": cdk.ID, "projectId": projectID, "projectName": projectName, "campaignId": cdk.CampaignID, "campaignName": safeCampaignName(c), "code": cdk.Code, "status": cdk.Status, "claimedByRecordId": cdk.ClaimedByRecordID, "claimedAt": cdk.ClaimedAt, "nodeId": cdk.NodeID, "createdAt": cdk.CreatedAt, "updatedAt": cdk.UpdatedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) ImportCDKs(campaignID string, codes []string, actor string) (*model.ImportCDKResponse, error) {
	return s.ImportCDKsForProject("", campaignID, codes, actor)
}

func (s *Store) ImportCDKsForProject(projectID, campaignID string, codes []string, actor string) (*model.ImportCDKResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(campaignID) == "" {
		return nil, model.NewAppError(http.StatusBadRequest, "CAMPAIGN_REQUIRED", "请先创建活动")
	}
	if len(codes) == 0 {
		return nil, model.NewAppError(http.StatusBadRequest, "CDK_EMPTY", "CDK 不能为空")
	}
	c := s.data.Campaigns[campaignID]
	if c == nil {
		return nil, model.NewAppError(http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "活动不存在，请先创建活动")
	}
	p := s.data.Projects[c.ProjectID]
	if p == nil {
		return nil, model.NewAppError(http.StatusNotFound, "PROJECT_NOT_FOUND", "活动所属项目不存在，请先创建项目")
	}
	if projectID != "" && projectID != c.ProjectID {
		return nil, model.NewAppError(http.StatusBadRequest, "PROJECT_CAMPAIGN_MISMATCH", "活动不属于所选项目")
	}
	// 一活动一 CDK 池：活动一旦有过 CDK，就不允许再次导入。
	if s.campaignHasCDKLocked(campaignID) {
		return nil, model.ErrCampaignCDKLocked
	}
	now := nowISO()
	// 全局 code 索引，确保同一个 CDK 字符串不会出现在多个活动里。
	globalUsed := map[string]string{}
	for _, cdk := range s.data.CDKs {
		globalUsed[cdk.Code] = cdk.CampaignID
	}
	seen := map[string]bool{}
	resp := &model.ImportCDKResponse{}
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" {
			resp.Invalid++
			continue
		}
		if seen[code] {
			resp.Duplicates++
			continue
		}
		if owner, exists := globalUsed[code]; exists {
			if owner == campaignID {
				// 理论上不会到这里：上面已经拦了"活动已锁定"，但保险起见。
				resp.Duplicates++
			} else {
				resp.UsedElsewhere++
				if len(resp.RejectedSample) < 5 {
					resp.RejectedSample = append(resp.RejectedSample, code)
				}
			}
			continue
		}
		seen[code] = true
		resp.Imported++
		resp.Success = resp.Imported
		if len(resp.Preview) < 10 {
			resp.Preview = append(resp.Preview, code)
		}
		s.addCDKLocked(campaignID, code, now)
		globalUsed[code] = campaignID
	}
	resp.Failed = resp.Invalid + resp.Duplicates + resp.UsedElsewhere
	s.recalcCampaignLocked(c)
	s.logLocked("operation", "info", "导入 CDK", fmt.Sprintf("活动 %s 导入 %d 条 CDK", c.Name, resp.Imported), actor, "", "campaign", campaignID, map[string]interface{}{"duplicates": resp.Duplicates, "invalid": resp.Invalid, "usedElsewhere": resp.UsedElsewhere})
	return resp, s.saveLocked()
}

// campaignHasCDKLocked 判定活动是否已经存在 CDK（无论状态）。
func (s *Store) campaignHasCDKLocked(campaignID string) bool {
	for _, cdk := range s.data.CDKs {
		if cdk.CampaignID == campaignID {
			return true
		}
	}
	return false
}

func (s *Store) SetCDKStatus(id, status, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cdk := s.data.CDKs[id]
	if cdk == nil {
		return model.ErrNotFound
	}
	if cdk.Status == model.StatusClaimed && status != model.StatusClaimed {
		return model.NewAppError(http.StatusConflict, "CDK_CLAIMED", "已领取 CDK 不能修改状态")
	}
	cdk.Status = status
	cdk.UpdatedAt = nowISO()
	if c := s.data.Campaigns[cdk.CampaignID]; c != nil {
		s.recalcCampaignLocked(c)
	}
	s.logLocked("operation", "info", "修改 CDK 状态", "CDK 状态变更为 "+status, actor, "", "cdk", id, nil)
	return s.saveLocked()
}

func (s *Store) DeleteCDK(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cdk := s.data.CDKs[id]
	if cdk == nil {
		return model.ErrNotFound
	}
	if cdk.Status == model.StatusClaimed {
		return model.NewAppError(http.StatusConflict, "CDK_CLAIMED", "已领取 CDK 不能删除")
	}
	delete(s.data.CDKs, id)
	if c := s.data.Campaigns[cdk.CampaignID]; c != nil {
		s.recalcCampaignLocked(c)
	}
	s.logLocked("operation", "warn", "删除 CDK", "删除未使用 CDK", actor, "", "cdk", id, nil)
	return s.saveLocked()
}

func (s *Store) BatchSetCDKStatus(ids []string, status, actor string) (int, error) {
	count := 0
	for _, id := range ids {
		if err := s.SetCDKStatus(id, status, actor); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Store) ListNodes(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	for _, n := range s.data.Nodes {
		if q["status"] != "" && q["status"] != "all" && n.Status != q["status"] {
			continue
		}
		if q["campaignId"] != "" && n.CampaignID != q["campaignId"] {
			continue
		}
		if q["projectId"] != "" && n.ProjectID != q["projectId"] {
			continue
		}
		if kw != "" && !containsAny(kw, n.Name, n.Slug, n.Title, n.Description) {
			continue
		}
		rows = append(rows, s.nodeViewLocked(n))
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) ListNodesForUser(q map[string]string, user *model.User) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	isAdmin := user.Role == "admin"
	for _, n := range s.data.Nodes {
		p := s.data.Projects[n.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		if q["status"] != "" && q["status"] != "all" && n.Status != q["status"] {
			continue
		}
		if q["campaignId"] != "" && n.CampaignID != q["campaignId"] {
			continue
		}
		if q["projectId"] != "" && n.ProjectID != q["projectId"] {
			continue
		}
		if kw != "" && !containsAny(kw, n.Name, n.Slug, n.Title, n.Description) {
			continue
		}
		rows = append(rows, s.nodeViewLocked(n))
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) GetNode(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := s.data.Nodes[id]
	if n == nil {
		return nil, model.ErrNotFound
	}
	return s.nodeViewLocked(n), nil
}

func (s *Store) GetNodeBySlug(slug string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := s.findNodeBySlugLocked(slug)
	if n == nil {
		return nil, model.ErrNotFound
	}
	return s.nodeViewLocked(n), nil
}

func (s *Store) CreateNode(req model.CreateNodeRequest, actor string) (map[string]interface{}, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" || req.CampaignID == "" {
		return nil, model.NewAppError(http.StatusBadRequest, "NODE_REQUIRED", "节点名称、slug 和绑定活动不能为空")
	}
	if !slugPattern.MatchString(req.Slug) {
		return nil, model.NewAppError(http.StatusBadRequest, "INVALID_SLUG", "slug 只能包含字母、数字、短横线和下划线")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.data.Campaigns[req.CampaignID]
	if c == nil {
		return nil, model.ErrNotFound
	}
	if s.findNodeBySlugLocked(req.Slug) != nil {
		return nil, model.NewAppError(http.StatusConflict, "SLUG_EXISTS", "节点标识已存在")
	}
	now := nowISO()
	projectID := firstNonEmpty(req.ProjectID, c.ProjectID)
	status := firstNonEmpty(req.Status, model.StatusActive)
	n := &model.DistributionNode{ID: MakeRandomString(12), ProjectID: projectID, CampaignID: req.CampaignID, Name: strings.TrimSpace(req.Name), Slug: strings.TrimSpace(req.Slug), Status: status, Title: defaultString(req.Title, req.Name), Description: req.Description, ButtonText: defaultString(req.ButtonText, "立即领取"), RequireCaptcha: req.RequireCaptcha, ShowStock: req.ShowStock, ShowEndTime: req.ShowEndTime, Limit: req.Limit, IPLimitEnabled: req.IPLimitEnabled, DeviceLimitEnabled: req.DeviceLimitEnabled, CreatedAt: now, UpdatedAt: now}
	s.data.Nodes[n.ID] = n
	c.NodeIDs = appendUnique(c.NodeIDs, n.ID)
	if p := s.data.Projects[projectID]; p != nil {
		p.NodeIDs = appendUnique(p.NodeIDs, n.ID)
	}
	s.logLocked("operation", "info", "创建节点", "创建节点 "+n.Name, actor, "", "node", n.ID, nil)
	return s.nodeViewLocked(n), s.saveLocked()
}

func (s *Store) UpdateNode(id string, req model.UpdateNodeRequest, actor string) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.data.Nodes[id]
	if n == nil {
		return nil, model.ErrNotFound
	}
	if req.Slug != nil && *req.Slug != n.Slug {
		if !slugPattern.MatchString(*req.Slug) {
			return nil, model.NewAppError(http.StatusBadRequest, "INVALID_SLUG", "slug 只能包含字母、数字、短横线和下划线")
		}
		if other := s.findNodeBySlugLocked(*req.Slug); other != nil && other.ID != id {
			return nil, model.NewAppError(http.StatusConflict, "SLUG_EXISTS", "节点标识已存在")
		}
		n.Slug = *req.Slug
	}
	if req.CampaignID != nil && *req.CampaignID != n.CampaignID {
		if s.data.Campaigns[*req.CampaignID] == nil {
			return nil, model.ErrNotFound
		}
		if old := s.data.Campaigns[n.CampaignID]; old != nil {
			old.NodeIDs = removeString(old.NodeIDs, id)
		}
		n.CampaignID = *req.CampaignID
		s.data.Campaigns[n.CampaignID].NodeIDs = appendUnique(s.data.Campaigns[n.CampaignID].NodeIDs, id)
	}
	if req.ProjectID != nil {
		n.ProjectID = *req.ProjectID
	}
	if req.Name != nil {
		n.Name = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		n.Status = *req.Status
	}
	if req.Title != nil {
		n.Title = *req.Title
	}
	if req.Description != nil {
		n.Description = *req.Description
	}
	if req.ButtonText != nil {
		n.ButtonText = *req.ButtonText
	}
	if req.RequireCaptcha != nil {
		n.RequireCaptcha = *req.RequireCaptcha
	}
	if req.ShowStock != nil {
		n.ShowStock = *req.ShowStock
	}
	if req.ShowEndTime != nil {
		n.ShowEndTime = *req.ShowEndTime
	}
	if req.Limit != nil {
		n.Limit = *req.Limit
	}
	if req.IPLimitEnabled != nil {
		n.IPLimitEnabled = *req.IPLimitEnabled
	}
	if req.DeviceLimitEnabled != nil {
		n.DeviceLimitEnabled = *req.DeviceLimitEnabled
	}
	n.UpdatedAt = nowISO()
	s.logLocked("operation", "info", "编辑节点", "编辑节点 "+n.Name, actor, "", "node", id, nil)
	return s.nodeViewLocked(n), s.saveLocked()
}

func (s *Store) SetNodeStatus(id, status, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.data.Nodes[id]
	if n == nil {
		return model.ErrNotFound
	}
	n.Status = status
	n.UpdatedAt = nowISO()
	title := "恢复节点"
	if status != model.StatusActive {
		title = "暂停节点"
	}
	s.logLocked("operation", "info", title, title+" "+n.Name, actor, "", "node", id, nil)
	return s.saveLocked()
}

func (s *Store) DeleteNode(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.data.Nodes[id]
	if n == nil {
		return model.ErrNotFound
	}
	delete(s.data.Nodes, id)
	if c := s.data.Campaigns[n.CampaignID]; c != nil {
		c.NodeIDs = removeString(c.NodeIDs, id)
	}
	if p := s.data.Projects[n.ProjectID]; p != nil {
		p.NodeIDs = removeString(p.NodeIDs, id)
	}
	s.logLocked("operation", "warn", "删除节点", "删除节点 "+n.Name, actor, "", "node", id, nil)
	return s.saveLocked()
}

func (s *Store) BatchNodeCaptcha(ids []string, enabled bool, actor string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, id := range ids {
		if n := s.data.Nodes[id]; n != nil {
			n.RequireCaptcha = enabled
			n.UpdatedAt = nowISO()
			count++
		}
	}
	s.logLocked("operation", "info", "批量修改节点验证码", "批量修改节点验证码", actor, "", "node", "batch", map[string]interface{}{"enabled": enabled, "count": count})
	return count, s.saveLocked()
}

func (s *Store) GetPublicNodeClaim(slug, userID, fingerprint string) (*model.ClaimPageInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.findNodeBySlugLocked(slug)
	if n == nil {
		return nil, model.ErrNotFound
	}
	c := s.data.Campaigns[n.CampaignID]
	if c == nil {
		return nil, model.ErrNotFound
	}
	n.Visits++
	n.LastVisitedAt = nowISO()
	n.UpdatedAt = n.LastVisitedAt
	if fingerprint != "" {
		n.UniqueVisitors = s.uniqueVisitorCountLocked(n.ID, fingerprint)
	}
	_ = s.saveLocked()
	return s.claimInfoLocked(c, n, userID, fingerprint), nil
}

func (s *Store) ClaimNodeReward(slug, userID, fingerprint, ip, ua string, captchaPassed bool) (*model.ClaimResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.findNodeBySlugLocked(slug)
	if n == nil {
		return nil, model.ErrNotFound
	}
	c := s.data.Campaigns[n.CampaignID]
	if c == nil {
		return nil, model.ErrNotFound
	}
	nodeID := n.ID
	// 优先使用 userID 做幂等判定。已登录用户在同一活动下只允许领取一次。
	if userID != "" {
		if existing := s.findUserClaimLocked(c.ID, nodeID, userID); existing != nil {
			return claimResponseFromRecord(existing), nil
		}
	}
	key := buildIdempotencyKey(c.ID, nodeID, fingerprint, ip, ua)
	if existing := s.findExistingSuccessLocked(c.ID, nodeID, key, userID, fingerprint, ip, ua); existing != nil {
		return claimResponseFromRecord(existing), nil
	}
	if err := s.validateClaimTargetLocked(c, n); err != nil {
		s.recordFailedClaimLocked(c, n, "", ip, ua, fingerprint, err.Error(), model.StatusFailed, false, nil, captchaPassed)
		return nil, err
	}
	if hit, err := s.evaluateRiskLocked(c, n, ip, ua, fingerprint); err != nil {
		s.recordFailedClaimLocked(c, n, "", ip, ua, fingerprint, err.Error(), model.StatusBlocked, true, hit, captchaPassed)
		s.logLocked("security", "warn", "风控拦截", err.Error(), "public", ip, "node", n.ID, map[string]interface{}{"rules": hit})
		return nil, err
	}
	var cdk *model.CDK
	for _, item := range s.data.CDKs {
		if item.CampaignID == c.ID && item.Status == model.StatusUnused {
			cdk = item
			break
		}
	}
	if cdk == nil {
		c.Status = model.StatusExhausted
		c.Enabled = false
		s.recalcCampaignLocked(c)
		s.recordFailedClaimLocked(c, n, "", ip, ua, fingerprint, "库存已耗尽", model.StatusFailed, false, nil, captchaPassed)
		return nil, model.ErrSoldOut
	}
	now := nowISO()
	rec := &model.ClaimRecord{ID: MakeRandomString(12), CampaignID: c.ID, NodeID: n.ID, ProjectID: c.ProjectID, CDKID: cdk.ID, Code: cdk.Code, RewardContent: cdk.Code, Status: model.StatusSuccess, IP: ip, UserAgent: ua, Fingerprint: fingerprint, IdempotencyKey: key, ClaimToken: MakeRandomString(32), HCaptchaPassed: captchaPassed, CreatedAt: now, UserID: userID, RewardItemID: cdk.ID}
	cdk.Status = model.StatusClaimed
	cdk.ClaimedAt = now
	cdk.ClaimedByRecordID = rec.ID
	cdk.NodeID = n.ID
	cdk.UpdatedAt = now
	s.data.ClaimRecords[rec.ID] = rec
	n.Claims++
	n.UpdatedAt = now
	s.recalcCampaignLocked(c)
	s.logLocked("claim", "info", "领取成功", "领取成功 "+cdk.Code, "public", ip, "claim", rec.ID, map[string]interface{}{"campaignId": c.ID, "nodeId": n.ID})
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return claimResponseFromRecord(rec), nil
}

func (s *Store) GetClaimResultByToken(slug, token string) (*model.ClaimResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := s.findNodeBySlugLocked(slug)
	if n == nil {
		return nil, model.ErrNotFound
	}
	for _, r := range s.data.ClaimRecords {
		if r.NodeID == n.ID && r.ClaimToken == token && r.Status == model.StatusSuccess {
			return claimResponseFromRecord(r), nil
		}
	}
	return nil, model.ErrNotFound
}

func (s *Store) ListClaims(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	start, _ := time.Parse(timeLayout, q["startAt"])
	end, _ := time.Parse(timeLayout, q["endAt"])
	for _, r := range s.data.ClaimRecords {
		if q["campaignId"] != "" && r.CampaignID != q["campaignId"] {
			continue
		}
		if q["nodeId"] != "" && r.NodeID != q["nodeId"] {
			continue
		}
		if q["projectId"] != "" && r.ProjectID != q["projectId"] {
			continue
		}
		if q["status"] != "" && q["status"] != "all" && r.Status != q["status"] {
			continue
		}
		if q["ip"] != "" && !strings.Contains(r.IP, q["ip"]) {
			continue
		}
		if q["fingerprint"] != "" && !strings.Contains(r.Fingerprint, q["fingerprint"]) {
			continue
		}
		if !start.IsZero() || !end.IsZero() {
			if t, err := time.Parse(timeLayout, r.CreatedAt); err == nil {
				if !start.IsZero() && t.Before(start) {
					continue
				}
				if !end.IsZero() && t.After(end) {
					continue
				}
			}
		}
		c := s.data.Campaigns[r.CampaignID]
		n := s.data.Nodes[r.NodeID]
		if kw != "" && !containsAny(kw, r.Code, r.IP, r.Fingerprint, r.UserAgent, safeCampaignName(c), safeNodeName(n)) {
			continue
		}
		rows = append(rows, s.claimViewLocked(r))
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) ListClaimsForUser(q map[string]string, user *model.User) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	start, _ := time.Parse(timeLayout, q["startAt"])
	end, _ := time.Parse(timeLayout, q["endAt"])
	isAdmin := user.Role == "admin"
	for _, r := range s.data.ClaimRecords {
		p := s.data.Projects[r.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		if q["campaignId"] != "" && r.CampaignID != q["campaignId"] {
			continue
		}
		if q["nodeId"] != "" && r.NodeID != q["nodeId"] {
			continue
		}
		if q["projectId"] != "" && r.ProjectID != q["projectId"] {
			continue
		}
		if q["status"] != "" && q["status"] != "all" && r.Status != q["status"] {
			continue
		}
		if q["ip"] != "" && !strings.Contains(r.IP, q["ip"]) {
			continue
		}
		if q["fingerprint"] != "" && !strings.Contains(r.Fingerprint, q["fingerprint"]) {
			continue
		}
		if !start.IsZero() || !end.IsZero() {
			if t, err := time.Parse(timeLayout, r.CreatedAt); err == nil {
				if !start.IsZero() && t.Before(start) {
					continue
				}
				if !end.IsZero() && t.After(end) {
					continue
				}
			}
		}
		c := s.data.Campaigns[r.CampaignID]
		n := s.data.Nodes[r.NodeID]
		if kw != "" && !containsAny(kw, r.Code, r.IP, r.Fingerprint, r.UserAgent, safeCampaignName(c), safeNodeName(n)) {
			continue
		}
		rows = append(rows, s.claimViewLocked(r))
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) GetClaim(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.data.ClaimRecords[id]
	if r == nil {
		return nil, model.ErrNotFound
	}
	return s.claimViewLocked(r), nil
}

func (s *Store) MarkClaimRisk(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.data.ClaimRecords[id]
	if r == nil {
		return model.ErrNotFound
	}
	r.RiskHit = true
	s.logLocked("security", "warn", "标记异常领取", "管理员标记异常领取", actor, "", "claim", id, nil)
	return s.saveLocked()
}

func (s *Store) Analytics(q map[string]string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	start, end := rangeBounds(q)
	days := dateBuckets(start, end)
	visits := make([]map[string]interface{}, len(days))
	claims := make([]map[string]interface{}, len(days))
	for i, d := range days {
		visits[i] = map[string]interface{}{"date": d, "value": 0}
		claims[i] = map[string]interface{}{"date": d, "success": 0, "failed": 0, "blocked": 0}
	}
	success, failed, blocked := 0, 0, 0
	failReasons := map[string]int{}
	campaignRank := []map[string]interface{}{}
	nodeRank := []map[string]interface{}{}
	for _, n := range s.data.Nodes {
		nodeRank = append(nodeRank, s.nodeViewLocked(n))
		if n.LastVisitedAt != "" {
			addTrendValue(visits, n.LastVisitedAt, n.Visits)
		}
	}
	for _, c := range s.data.Campaigns {
		campaignRank = append(campaignRank, s.campaignViewLocked(c))
	}
	for _, r := range s.data.ClaimRecords {
		if !inRange(r.CreatedAt, start, end) {
			continue
		}
		switch r.Status {
		case model.StatusSuccess:
			success++
			addClaimTrend(claims, r.CreatedAt, "success")
		case model.StatusBlocked:
			blocked++
			addClaimTrend(claims, r.CreatedAt, "blocked")
		default:
			failed++
			addClaimTrend(claims, r.CreatedAt, "failed")
		}
		if r.Status != model.StatusSuccess {
			failReasons[firstNonEmpty(r.Reason, "未知原因")]++
		}
	}
	sort.Slice(campaignRank, func(i, j int) bool {
		return toInt(campaignRank[i]["claimedCount"]) > toInt(campaignRank[j]["claimedCount"])
	})
	sort.Slice(nodeRank, func(i, j int) bool { return toInt(nodeRank[i]["claims"]) > toInt(nodeRank[j]["claims"]) })
	return map[string]interface{}{"overview": map[string]interface{}{"success": success, "failed": failed, "blocked": blocked, "total": success + failed + blocked, "successRate": percent(success, success+failed+blocked)}, "visitsTrend": visits, "claimsTrend": claims, "conversionRanking": limitMaps(nodeRank, 10), "campaignRanking": limitMaps(campaignRank, 10), "nodeRanking": limitMaps(nodeRank, 10), "failureReasons": reasonRows(failReasons)}
}

func (s *Store) AnalyticsForUser(q map[string]string, user *model.User) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	start, end := rangeBounds(q)
	days := dateBuckets(start, end)
	visits := make([]map[string]interface{}, len(days))
	claims := make([]map[string]interface{}, len(days))
	for i, d := range days {
		visits[i] = map[string]interface{}{"date": d, "value": 0}
		claims[i] = map[string]interface{}{"date": d, "success": 0, "failed": 0, "blocked": 0}
	}
	success, failed, blocked := 0, 0, 0
	failReasons := map[string]int{}
	campaignRank := []map[string]interface{}{}
	nodeRank := []map[string]interface{}{}
	isAdmin := user.Role == "admin"

	for _, n := range s.data.Nodes {
		p := s.data.Projects[n.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		nodeRank = append(nodeRank, s.nodeViewLocked(n))
		if n.LastVisitedAt != "" {
			addTrendValue(visits, n.LastVisitedAt, n.Visits)
		}
	}
	for _, c := range s.data.Campaigns {
		p := s.data.Projects[c.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		campaignRank = append(campaignRank, s.campaignViewLocked(c))
	}
	for _, r := range s.data.ClaimRecords {
		p := s.data.Projects[r.ProjectID]
		if !isAdmin && (p == nil || p.CreatorID != user.ID) {
			continue
		}
		if !inRange(r.CreatedAt, start, end) {
			continue
		}
		switch r.Status {
		case model.StatusSuccess:
			success++
			addClaimTrend(claims, r.CreatedAt, "success")
		case model.StatusBlocked:
			blocked++
			addClaimTrend(claims, r.CreatedAt, "blocked")
		default:
			failed++
			addClaimTrend(claims, r.CreatedAt, "failed")
		}
		if r.Status != model.StatusSuccess {
			failReasons[firstNonEmpty(r.Reason, "未知原因")]++
		}
	}
	sort.Slice(campaignRank, func(i, j int) bool {
		return toInt(campaignRank[i]["claimedCount"]) > toInt(campaignRank[j]["claimedCount"])
	})
	sort.Slice(nodeRank, func(i, j int) bool { return toInt(nodeRank[i]["claims"]) > toInt(nodeRank[j]["claims"]) })
	return map[string]interface{}{"overview": map[string]interface{}{"success": success, "failed": failed, "blocked": blocked, "total": success + failed + blocked, "successRate": percent(success, success+failed+blocked)}, "visitsTrend": visits, "claimsTrend": claims, "conversionRanking": limitMaps(nodeRank, 10), "campaignRanking": limitMaps(campaignRank, 10), "nodeRanking": limitMaps(nodeRank, 10), "failureReasons": reasonRows(failReasons)}
}

func (s *Store) ListLogs(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	kw := strings.ToLower(q["keyword"])
	for _, l := range s.data.SystemLogs {
		if q["type"] != "" && q["type"] != "all" && l.Type != q["type"] {
			continue
		}
		if q["level"] != "" && q["level"] != "all" && l.Level != q["level"] {
			continue
		}
		if kw != "" && !containsAny(kw, l.Title, l.Message, l.Actor, l.TargetID) {
			continue
		}
		rows = append(rows, map[string]interface{}{"id": l.ID, "type": l.Type, "level": l.Level, "title": l.Title, "message": l.Message, "actor": l.Actor, "ip": l.IP, "targetType": l.TargetType, "targetId": l.TargetID, "metadata": l.Metadata, "createdAt": l.CreatedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) GetLog(id string) (*model.SystemLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := s.data.SystemLogs[id]
	if l == nil {
		return nil, model.ErrNotFound
	}
	cp := *l
	return &cp, nil
}
func (s *Store) DeleteLog(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.SystemLogs[id] == nil {
		return model.ErrNotFound
	}
	delete(s.data.SystemLogs, id)
	return s.saveLocked()
}
func (s *Store) CleanupLogs(days int, actor string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if days <= 0 {
		days = 30
	}
	cut := time.Now().AddDate(0, 0, -days)
	count := 0
	for id, l := range s.data.SystemLogs {
		if t, err := time.Parse(timeLayout, l.CreatedAt); err == nil && t.Before(cut) {
			delete(s.data.SystemLogs, id)
			count++
		}
	}
	s.logLocked("operation", "warn", "清理旧日志", fmt.Sprintf("清理 %d 天前日志 %d 条", days, count), actor, "", "log", "cleanup", nil)
	return count, s.saveLocked()
}

func (s *Store) RiskOverview() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hits := 0
	for _, r := range s.data.ClaimRecords {
		if r.RiskHit || r.Status == model.StatusBlocked {
			hits++
		}
	}
	return map[string]interface{}{"rules": len(s.data.RiskRules), "enabledRules": countEnabledRules(s.data.RiskRules), "blacklist": len(s.data.Blacklist), "enabledBlacklist": countEnabledBlacklist(s.data.Blacklist), "hits": hits}
}

func (s *Store) ListRiskRules() []model.RiskRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []model.RiskRule{}
	for _, r := range s.data.RiskRules {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}
func (s *Store) SaveRiskRule(id string, rule model.RiskRule, actor string) (*model.RiskRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowISO()
	if id == "" {
		id = MakeRandomString(12)
		rule.CreatedAt = now
	}
	rule.ID = id
	if rule.Action == "" {
		rule.Action = "block"
	}
	if rule.Config == nil {
		rule.Config = map[string]interface{}{}
	}
	rule.UpdatedAt = now
	s.data.RiskRules[id] = &rule
	s.logLocked("operation", "info", "保存风控规则", "保存风控规则 "+rule.Name, actor, "", "riskRule", id, nil)
	return &rule, s.saveLocked()
}
func (s *Store) DeleteRiskRule(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.RiskRules[id] == nil {
		return model.ErrNotFound
	}
	delete(s.data.RiskRules, id)
	s.logLocked("operation", "warn", "删除风控规则", "删除风控规则", actor, "", "riskRule", id, nil)
	return s.saveLocked()
}
func (s *Store) EnableRiskRule(id string, enabled bool, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.data.RiskRules[id]
	if r == nil {
		return model.ErrNotFound
	}
	r.Enabled = enabled
	r.UpdatedAt = nowISO()
	return s.saveLocked()
}
func (s *Store) ListBlacklist() []model.BlacklistItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []model.BlacklistItem{}
	for _, r := range s.data.Blacklist {
		out = append(out, *r)
	}
	return out
}
func (s *Store) SaveBlacklist(id string, item model.BlacklistItem, actor string) (*model.BlacklistItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowISO()
	if id == "" {
		id = MakeRandomString(12)
		item.CreatedAt = now
	}
	item.ID = id
	item.UpdatedAt = now
	s.data.Blacklist[id] = &item
	s.logLocked("operation", "info", "保存黑名单", "保存黑名单 "+item.Value, actor, "", "blacklist", id, nil)
	return &item, s.saveLocked()
}
func (s *Store) DeleteBlacklist(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Blacklist[id] == nil {
		return model.ErrNotFound
	}
	delete(s.data.Blacklist, id)
	return s.saveLocked()
}
func (s *Store) EnableBlacklist(id string, enabled bool, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.data.Blacklist[id]
	if b == nil {
		return model.ErrNotFound
	}
	b.Enabled = enabled
	b.UpdatedAt = nowISO()
	return s.saveLocked()
}
func (s *Store) RiskHits() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	for _, r := range s.data.ClaimRecords {
		if r.RiskHit || r.Status == model.StatusBlocked {
			rows = append(rows, s.claimViewLocked(r))
		}
	}
	return rows
}

func (s *Store) CaptchaConfig(siteConfigured, secretConfigured bool) model.CaptchaConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.data.CaptchaConfig
	c.HCaptchaSiteKeyConfigured = siteConfigured
	c.HCaptchaSecretConfigured = secretConfigured
	return c
}
func (s *Store) UpdateCaptchaConfig(c model.CaptchaConfig, actor string) (model.CaptchaConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.data.CaptchaConfig
	if c.Provider == "" {
		c.Provider = old.Provider
	}
	c.HCaptchaSecretConfigured = false
	c.HCaptchaSiteKeyConfigured = false
	c.LastTestStatus = old.LastTestStatus
	c.LastTestMessage = old.LastTestMessage
	c.LastTestAt = old.LastTestAt
	s.data.CaptchaConfig = c
	s.logLocked("operation", "info", "修改验证码配置", "修改验证码配置", actor, "", "captcha", "config", nil)
	return c, s.saveLocked()
}
func (s *Store) SetCaptchaTest(status, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.CaptchaConfig.LastTestStatus = status
	s.data.CaptchaConfig.LastTestMessage = msg
	s.data.CaptchaConfig.LastTestAt = nowISO()
	return s.saveLocked()
}
func (s *Store) Settings() model.Settings { s.mu.RLock(); defer s.mu.RUnlock(); return s.data.Settings }
func (s *Store) UpdateSettings(settings model.Settings, actor string) (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if settings.SystemName == "" {
		settings.SystemName = s.data.Settings.SystemName
	}
	if settings.BrandName == "" {
		settings.BrandName = s.data.Settings.BrandName
	}
	if settings.BrandEnglishName == "" {
		settings.BrandEnglishName = s.data.Settings.BrandEnglishName
	}
	if settings.LogoText == "" {
		settings.LogoText = s.data.Settings.LogoText
	}
	if settings.StorageMode == "" {
		settings.StorageMode = "json"
	}
	settings.RedisEnabled = s.redisClient != nil
	settings.RabbitMQEnabled = s.amqpChannel != nil
	settings.CreatedAt = firstNonEmpty(s.data.Settings.CreatedAt, nowISO())
	settings.UpdatedAt = nowISO()
	s.data.Settings = settings
	s.logLocked("operation", "info", "修改系统设置", "修改系统设置", actor, "", "settings", "global", nil)
	return settings, s.saveLocked()
}
func (s *Store) ListAdmins() []model.Admin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []model.Admin{}
	for _, u := range s.data.Admins {
		out = append(out, *u)
	}
	return out
}
func (s *Store) UpdateAdmin(id string, req model.Admin, actor string) (*model.Admin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.data.Admins[id]
	if u == nil {
		return nil, model.ErrNotFound
	}
	if req.Username != "" {
		u.Username = req.Username
	}
	if req.Password != "" {
		u.Password = req.Password
	}
	if req.Role != "" {
		u.Role = req.Role
	}
	u.UpdatedAt = nowISO()
	if s.data.Users[id] != nil {
		*s.data.Users[id] = *u
	}
	return u, s.saveLocked()
}
func (s *Store) DeleteAdmin(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Admins[id] == nil {
		return model.ErrNotFound
	}
	delete(s.data.Admins, id)
	delete(s.data.Users, id)
	return s.saveLocked()
}

func (s *Store) ExportCSV(kind string, q map[string]string, actor string) (*model.ExportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(filepath.Dir(s.path), "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s-%s.csv", kind, time.Now().Format("20060102-150405"))
	filePath := filepath.Join(dir, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	switch kind {
	case "cdks":
		w.Write([]string{"id", "campaignId", "campaignName", "code", "status", "claimedAt", "nodeId"})
		for _, cdk := range s.data.CDKs {
			c := s.data.Campaigns[cdk.CampaignID]
			w.Write([]string{cdk.ID, cdk.CampaignID, safeCampaignName(c), cdk.Code, cdk.Status, cdk.ClaimedAt, cdk.NodeID})
		}
	case "logs":
		w.Write([]string{"id", "type", "level", "title", "message", "actor", "ip", "createdAt"})
		for _, l := range s.data.SystemLogs {
			w.Write([]string{l.ID, l.Type, l.Level, l.Title, l.Message, l.Actor, l.IP, l.CreatedAt})
		}
	default:
		w.Write([]string{"id", "campaignId", "nodeId", "projectId", "code", "status", "reason", "ip", "fingerprint", "createdAt"})
		for _, r := range s.data.ClaimRecords {
			w.Write([]string{r.ID, r.CampaignID, r.NodeID, r.ProjectID, r.Code, r.Status, r.Reason, r.IP, r.Fingerprint, r.CreatedAt})
		}
	}
	now := nowISO()
	task := &model.ExportTask{ID: MakeRandomString(12), Type: kind, Status: "done", Filename: filename, FilePath: "/api/admin/exports/" + filename, CreatedAt: now, FinishedAt: now, Filter: map[string]interface{}{}}
	s.data.ExportTasks[task.ID] = task
	s.logLocked("operation", "info", "导出 CSV", "导出 "+kind, actor, "", "export", task.ID, nil)
	return task, s.saveLocked()
}

func (s *Store) ListExportTasks(q map[string]string) model.PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []map[string]interface{}{}
	taskType := q["type"]
	for _, task := range s.data.ExportTasks {
		if taskType != "" && taskType != "all" && task.Type != taskType {
			continue
		}
		rows = append(rows, map[string]interface{}{
			"id":         task.ID,
			"type":       task.Type,
			"status":     task.Status,
			"filename":   task.Filename,
			"filePath":   task.FilePath,
			"filter":     task.Filter,
			"error":      task.Error,
			"createdAt":  task.CreatedAt,
			"finishedAt": task.FinishedAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["createdAt"]) > fmt.Sprint(rows[j]["createdAt"]) })
	return paginateMaps(rows, q)
}

func (s *Store) Backup(actor string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	filename := "state-" + time.Now().Format("20060102-150405") + ".json"
	target := filepath.Join(dir, filename)
	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(target, payload, 0o644); err != nil {
		return nil, err
	}
	s.logLocked("operation", "info", "数据备份", "创建数据备份 "+filename, actor, "", "backup", filename, nil)
	_ = s.saveLocked()
	return map[string]string{"filename": filename, "path": "/api/admin/backups/" + filename}, nil
}

func (s *Store) Restore(filename string, payload []byte, actor string) error {
	var next model.SystemData
	if len(payload) == 0 {
		clean := filepath.Clean(filename)
		if strings.Contains(clean, "..") {
			return model.ErrBadRequest
		}
		b, err := os.ReadFile(filepath.Join(filepath.Dir(s.path), "backups", clean))
		if err != nil {
			return err
		}
		payload = b
	}
	if err := json.Unmarshal(payload, &next); err != nil {
		return model.NewAppError(http.StatusBadRequest, "INVALID_BACKUP", "备份 JSON 格式不正确")
	}
	if _, err := s.Backup(actor); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = next
	s.migrateLocked()
	s.logLocked("operation", "warn", "数据恢复", "恢复数据备份", actor, "", "backup", filename, nil)
	return s.saveLocked()
}

func (s *Store) ServeDataFile(w http.ResponseWriter, r *http.Request, folder string) {
	name := filepath.Base(r.URL.Path)
	path := filepath.Join(filepath.Dir(s.path), folder, name)
	http.ServeFile(w, r, path)
}

func (s *Store) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.saveLocked()
	if s.amqpChannel != nil {
		_ = s.amqpChannel.Close()
	}
	if s.amqpConn != nil {
		_ = s.amqpConn.Close()
	}
	if s.redisClient != nil {
		_ = s.redisClient.Close()
	}
}

// Legacy compatibility methods used by older pages and the welfare extension.
func (s *Store) ListProjects() []map[string]interface{} {
	return s.ListCampaigns(map[string]string{}).Items.([]map[string]interface{})
}
func (s *Store) DisableProject(id, userID string) error {
	return s.SetCampaignStatus(id, model.StatusPaused, userID)
}
func (s *Store) PauseProject(id string) error { return s.SetCampaignStatus(id, model.StatusPaused, "") }
func (s *Store) ResumeProject(id string) error {
	return s.SetCampaignStatus(id, model.StatusActive, "")
}
func (s *Store) ClaimReward(code, userID, fingerprint, ip, ua string) (*model.ClaimResponse, error) {
	s.mu.RLock()
	var slug string
	for _, n := range s.data.Nodes {
		if n.Slug == code {
			slug = n.Slug
			break
		}
	}
	s.mu.RUnlock()
	if slug == "" {
		slug = code
	}
	return s.ClaimNodeReward(slug, userID, fingerprint, ip, ua, false)
}
func (s *Store) GetClaimPageInfo(code, userID, fingerprint string) (*model.ClaimPageInfo, error) {
	return s.GetPublicNodeClaim(code, userID, fingerprint)
}
func (s *Store) SyncStockToRedis(p *model.Project) {}

func (s *Store) saveLocked() error {
	if s.db != nil {
		// SQL mode: sync all in-memory data to the configured database.
		return s.syncAllToSQL()
	}
	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// syncAllToSQL writes all in-memory data to the configured SQL database.
// Called on every mutation (same as the old JSON full-write pattern).
// For better perf, individual dbSave* calls can be used.
func (s *Store) syncAllToSQL() error {
	for _, u := range s.data.Users {
		s.dbSaveUser(u)
	}
	for _, p := range s.data.Projects {
		s.dbSaveProject(p)
	}
	for _, c := range s.data.Campaigns {
		s.dbSaveCampaign(c)
	}
	for _, k := range s.data.CDKs {
		s.dbSaveCDK(k)
	}
	for _, n := range s.data.Nodes {
		s.dbSaveNode(n)
	}
	for _, r := range s.data.ClaimRecords {
		s.dbSaveClaimRecord(r)
	}
	for _, r := range s.data.RiskRules {
		s.dbSaveRiskRule(r)
	}
	for _, b := range s.data.Blacklist {
		s.dbSaveBlacklist(b)
	}
	for _, task := range s.data.ExportTasks {
		s.dbSaveExportTask(task)
	}
	s.dbSaveSettings(s.data.Settings)
	s.dbSaveCaptchaConfig(s.data.CaptchaConfig)
	return nil
}

func (s *Store) addCDKLocked(campaignID, raw, now string) {
	code := strings.TrimSpace(raw)
	if code == "" {
		return
	}
	id := MakeRandomString(12)
	s.data.CDKs[id] = &model.CDK{ID: id, CampaignID: campaignID, Code: code, Status: model.StatusUnused, CreatedAt: now, UpdatedAt: now}
}

func (s *Store) recalcCampaignLocked(c *model.Campaign) {
	total, claimed, remaining := 0, 0, 0
	for _, cdk := range s.data.CDKs {
		if cdk.CampaignID == c.ID {
			total++
			if cdk.Status == model.StatusClaimed {
				claimed++
			}
			if cdk.Status == model.StatusUnused {
				remaining++
			}
		}
	}
	c.TotalStock = total
	c.ClaimedCount = claimed
	c.RemainingCount = remaining
	if c.Status == model.StatusActive && total > 0 && remaining == 0 {
		c.Status = model.StatusExhausted
		c.Enabled = false
	}
	if c.Status == model.StatusExhausted && remaining > 0 {
		c.Status = model.StatusActive
		c.Enabled = true
	}
}

func (s *Store) campaignViewLocked(c *model.Campaign) map[string]interface{} {
	status := s.effectiveCampaignStatusLocked(c)
	projectName := ""
	if p := s.data.Projects[c.ProjectID]; p != nil {
		projectName = p.Name
	}
	return map[string]interface{}{"id": c.ID, "projectId": c.ProjectID, "projectName": projectName, "name": c.Name, "description": c.Description, "status": status, "totalStock": c.TotalStock, "claimedCount": c.ClaimedCount, "remainingCount": c.RemainingCount, "remaining": c.RemainingCount, "startAt": c.StartAt, "endAt": c.EndAt, "startTime": c.StartAt, "endTime": c.EndAt, "allowRepeat": c.AllowRepeat, "perUserLimit": c.PerUserLimit, "perIPLimit": c.PerIPLimit, "perDeviceLimit": c.PerDeviceLimit, "requireCaptchaDefault": c.RequireCaptchaDefault, "nodeIds": c.NodeIDs, "nodeCount": len(c.NodeIDs), "createdAt": c.CreatedAt, "updatedAt": c.UpdatedAt, "enabled": status == model.StatusActive, "projectCode": c.ProjectCode, "claimUrl": "/claim/" + c.ProjectCode, "rules": c.Rules, "cdkPoolLocked": c.TotalStock > 0}
}

func (s *Store) projectDetailLocked(p *model.Project) map[string]interface{} {
	campaigns := []map[string]interface{}{}
	nodes := []map[string]interface{}{}
	for _, id := range p.CampaignIDs {
		if c := s.data.Campaigns[id]; c != nil {
			campaigns = append(campaigns, s.campaignViewLocked(c))
		}
	}
	for _, id := range p.NodeIDs {
		if n := s.data.Nodes[id]; n != nil {
			nodes = append(nodes, s.nodeViewLocked(n))
		}
	}
	return map[string]interface{}{"id": p.ID, "name": p.Name, "description": p.Description, "status": p.Status, "campaignIds": p.CampaignIDs, "nodeIds": p.NodeIDs, "campaigns": campaigns, "nodes": nodes, "createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt}
}

func (s *Store) nodeViewLocked(n *model.DistributionNode) map[string]interface{} {
	c := s.data.Campaigns[n.CampaignID]
	campaignStatus := ""
	remaining := 0
	campaignName := ""
	if c != nil {
		campaignStatus = s.effectiveCampaignStatusLocked(c)
		remaining = c.RemainingCount
		campaignName = c.Name
	}
	conversion := 0
	if n.Visits > 0 {
		conversion = int(float64(n.Claims) / float64(n.Visits) * 100)
	}
	return map[string]interface{}{"id": n.ID, "projectId": n.ProjectID, "campaignId": n.CampaignID, "campaignName": campaignName, "campaignStatus": campaignStatus, "name": n.Name, "slug": n.Slug, "status": n.Status, "title": n.Title, "description": n.Description, "buttonText": n.ButtonText, "requireCaptcha": n.RequireCaptcha, "showStock": n.ShowStock, "showEndTime": n.ShowEndTime, "visits": n.Visits, "uniqueVisitors": n.UniqueVisitors, "claims": n.Claims, "failedClaims": n.FailedClaims, "remaining": remaining, "conversion": conversion, "lastVisitedAt": n.LastVisitedAt, "createdAt": n.CreatedAt, "updatedAt": n.UpdatedAt, "claimUrl": "/claim/" + n.Slug, "limit": n.Limit, "ipLimitEnabled": n.IPLimitEnabled, "deviceLimitEnabled": n.DeviceLimitEnabled}
}

func (s *Store) claimViewLocked(r *model.ClaimRecord) map[string]interface{} {
	return map[string]interface{}{"id": r.ID, "campaignId": r.CampaignID, "campaignName": safeCampaignName(s.data.Campaigns[r.CampaignID]), "nodeId": r.NodeID, "nodeName": safeNodeName(s.data.Nodes[r.NodeID]), "projectId": r.ProjectID, "cdkId": r.CDKID, "code": r.Code, "rewardContent": firstNonEmpty(r.RewardContent, r.Code), "status": r.Status, "reason": r.Reason, "ip": r.IP, "userAgent": r.UserAgent, "fingerprint": r.Fingerprint, "idempotencyKey": r.IdempotencyKey, "claimToken": r.ClaimToken, "hcaptchaPassed": r.HCaptchaPassed, "riskHit": r.RiskHit, "riskRuleIds": r.RiskRuleIDs, "createdAt": r.CreatedAt}
}

func (s *Store) claimInfoLocked(c *model.Campaign, n *model.DistributionNode, userID, fingerprint string) *model.ClaimPageInfo {
	info := &model.ClaimPageInfo{NodeID: n.ID, NodeSlug: n.Slug, ProjectCode: n.Slug, Name: defaultString(n.Title, c.Name), Description: defaultString(n.Description, c.Description), StartTime: c.StartAt, EndTime: c.EndAt, StartAt: c.StartAt, EndAt: c.EndAt, TotalStock: c.TotalStock, ClaimedCount: c.ClaimedCount, Remaining: c.RemainingCount, RemainingCount: c.RemainingCount, PerUserLimit: c.PerUserLimit, RewardType: "cdk_list", Rules: c.Rules, Enabled: n.Status == model.StatusActive && s.effectiveCampaignStatusLocked(c) == model.StatusActive, Status: s.effectiveCampaignStatusLocked(c), ButtonText: defaultString(n.ButtonText, "立即领取"), ShowStock: n.ShowStock, ShowEndTime: n.ShowEndTime, RequireCaptcha: n.RequireCaptcha || c.RequireCaptchaDefault}
	// 当登录用户存在时，严格按 userID 查找已领取记录，避免相同浏览器/指纹下不同账号互相串数据
	var existing *model.ClaimRecord
	if userID != "" {
		existing = s.findUserClaimLocked(c.ID, n.ID, userID)
	} else {
		existing = s.findExistingSuccessLocked(c.ID, n.ID, "", "", fingerprint, "", "")
	}
	if existing != nil {
		info.UserClaimed = true
		info.UserRewardContent = firstNonEmpty(existing.Code, existing.RewardContent)
		info.UserClaimedAt = existing.CreatedAt
	}
	if userID != "" {
		if u := s.data.Users[userID]; u != nil {
			info.UserInfo = &model.UserInfo{
				UserID:          u.ID,
				Username:        u.Username,
				Role:            u.Role,
				Avatar:          u.Avatar,
				Nickname:        u.Nickname,
				Email:           u.Email,
				CommunityUserID: u.CommunityUserID,
			}
		}
	}
	return info
}

// findUserClaimLocked 严格按 userID + campaign(+node) 查找用户成功的领取记录，
// 用于"是否已领取"的判定，不会被指纹/IP 命中其他用户的记录串号。
func (s *Store) findUserClaimLocked(campaignID, nodeID, userID string) *model.ClaimRecord {
	if userID == "" {
		return nil
	}
	var best *model.ClaimRecord
	for _, r := range s.data.ClaimRecords {
		if r.Status != model.StatusSuccess || r.UserID != userID {
			continue
		}
		if campaignID != "" && r.CampaignID != campaignID {
			continue
		}
		if nodeID != "" && r.NodeID != "" && r.NodeID != nodeID {
			continue
		}
		if best == nil || r.CreatedAt > best.CreatedAt {
			best = r
		}
	}
	return best
}

func (s *Store) effectiveCampaignStatusLocked(c *model.Campaign) string {
	if c.Status == model.StatusPaused || c.Status == model.StatusEnded || c.Status == model.StatusArchived || c.Status == model.StatusExhausted || c.Status == model.StatusDraft {
		return c.Status
	}
	now := time.Now()
	if c.StartAt != "" {
		if t, err := time.Parse(timeLayout, c.StartAt); err == nil && now.Before(t) {
			return model.StatusDraft
		}
	}
	if c.EndAt != "" {
		if t, err := time.Parse(timeLayout, c.EndAt); err == nil && now.After(t) {
			return model.StatusEnded
		}
	}
	if c.TotalStock > 0 && c.RemainingCount <= 0 {
		return model.StatusExhausted
	}
	return model.StatusActive
}

func (s *Store) validateClaimTargetLocked(c *model.Campaign, n *model.DistributionNode) error {
	if n.Status != model.StatusActive {
		return model.ErrProjectClosed
	}
	if n.Limit > 0 && n.Claims >= n.Limit {
		return model.ErrSoldOut
	}
	switch s.effectiveCampaignStatusLocked(c) {
	case model.StatusDraft:
		return model.ErrNotStarted
	case model.StatusEnded:
		return model.ErrEnded
	case model.StatusPaused, model.StatusArchived:
		return model.ErrProjectClosed
	case model.StatusExhausted:
		return model.ErrSoldOut
	}
	return nil
}

func (s *Store) evaluateRiskLocked(c *model.Campaign, n *model.DistributionNode, ip, ua, fingerprint string) ([]string, error) {
	hits := []string{}
	for _, b := range s.data.Blacklist {
		if !b.Enabled {
			continue
		}
		matched := false
		switch b.Type {
		case "ip":
			matched = b.Value == ip
		case "fingerprint":
			matched = b.Value != "" && b.Value == fingerprint
		case "user_agent":
			matched = b.Value != "" && strings.Contains(strings.ToLower(ua), strings.ToLower(b.Value))
		}
		if matched {
			hits = append(hits, b.ID)
			return hits, model.NewAppError(http.StatusForbidden, "RISK_BLOCKED", "命中黑名单："+b.Reason)
		}
	}
	for _, r := range s.data.RiskRules {
		if !r.Enabled {
			continue
		}
		switch r.Type {
		case "rate_limit", "ip_limit":
			limit := configInt(r.Config, "limit", 10)
			win := configInt(r.Config, "windowSeconds", 60)
			if s.countRecentClaimsByIPLocked(ip, win) >= limit {
				hits = append(hits, r.ID)
			}
		case "device_limit":
			if !c.AllowRepeat && fingerprint != "" {
				if ex := s.findExistingSuccessLocked(c.ID, n.ID, "", "", fingerprint, "", ""); ex != nil {
					return []string{r.ID}, nil
				}
			}
		case "user_agent":
			pattern := strings.ToLower(fmt.Sprint(r.Config["pattern"]))
			if pattern != "" && strings.Contains(strings.ToLower(ua), pattern) {
				hits = append(hits, r.ID)
			}
		}
	}
	if len(hits) > 0 {
		return hits, model.NewAppError(http.StatusForbidden, "RISK_BLOCKED", "领取请求触发风控限制")
	}
	return nil, nil
}

func (s *Store) recordFailedClaimLocked(c *model.Campaign, n *model.DistributionNode, code, ip, ua, fp, reason, status string, risk bool, rules []string, captcha bool) {
	now := nowISO()
	rec := &model.ClaimRecord{ID: MakeRandomString(12), CampaignID: c.ID, NodeID: n.ID, ProjectID: c.ProjectID, Code: code, RewardContent: code, Status: status, Reason: reason, IP: ip, UserAgent: ua, Fingerprint: fp, HCaptchaPassed: captcha, RiskHit: risk, RiskRuleIDs: rules, CreatedAt: now}
	s.data.ClaimRecords[rec.ID] = rec
	n.FailedClaims++
	n.UpdatedAt = now
	typ := "claim"
	level := "warn"
	title := "领取失败"
	if status == model.StatusBlocked {
		typ = "security"
		title = "风控拦截"
	}
	s.logLocked(typ, level, title, reason, "public", ip, "claim", rec.ID, map[string]interface{}{"campaignId": c.ID, "nodeId": n.ID})
	_ = s.saveLocked()
}

func (s *Store) findExistingSuccessLocked(campaignID, nodeID, key, userID, fingerprint, ip, ua string) *model.ClaimRecord {
	var best *model.ClaimRecord
	for _, r := range s.data.ClaimRecords {
		if r.CampaignID != campaignID || r.Status != model.StatusSuccess {
			continue
		}
		if nodeID != "" && r.NodeID != "" && r.NodeID != nodeID {
			continue
		}
		matched := false
		if key != "" && r.IdempotencyKey == key {
			matched = true
		}
		if !matched && userID != "" && r.UserID == userID {
			matched = true
		}
		if !matched && fingerprint != "" && r.Fingerprint == fingerprint {
			matched = true
		}
		if !matched && fingerprint == "" && userID == "" && ip != "" && r.IP == ip && r.UserAgent == ua {
			matched = true
		}
		if matched && (best == nil || r.CreatedAt > best.CreatedAt) {
			best = r
		}
	}
	return best
}

func (s *Store) findNodeBySlugLocked(slug string) *model.DistributionNode {
	for _, n := range s.data.Nodes {
		if n.Slug == slug {
			return n
		}
	}
	return nil
}
func (s *Store) uniqueVisitorCountLocked(nodeID, fingerprint string) int {
	seen := map[string]bool{fingerprint: true}
	for _, r := range s.data.ClaimRecords {
		if r.NodeID == nodeID && r.Fingerprint != "" {
			seen[r.Fingerprint] = true
		}
	}
	return len(seen)
}
func (s *Store) countRecentClaimsByIPLocked(ip string, seconds int) int {
	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)
	count := 0
	for _, r := range s.data.ClaimRecords {
		if r.IP == ip {
			if t, err := time.Parse(timeLayout, r.CreatedAt); err == nil && t.After(cutoff) {
				count++
			}
		}
	}
	return count
}
func (s *Store) logLocked(typ, level, title, msg, actor, ip, targetType, targetID string, meta map[string]interface{}) {
	id := MakeRandomString(12)
	l := &model.SystemLog{ID: id, Type: typ, Level: level, Title: title, Message: msg, Actor: actor, IP: ip, TargetType: targetType, TargetID: targetID, Metadata: meta, CreatedAt: nowISO()}
	s.data.SystemLogs[id] = l
	s.dbSaveLog(l)
}

func legacyCampaignStatus(p *model.Project) string {
	if !p.Enabled {
		return model.StatusPaused
	}
	return model.StatusActive
}
func buildIdempotencyKey(campaignID, nodeID, fingerprint, ip, ua string) string {
	base := campaignID + ":" + nodeID + ":"
	if strings.TrimSpace(fingerprint) != "" {
		return base + "fp:" + strings.TrimSpace(fingerprint)
	}
	sum := sha256.Sum256([]byte(ua))
	return base + "net:" + ip + ":" + hex.EncodeToString(sum[:])
}
func claimResponseFromRecord(r *model.ClaimRecord) *model.ClaimResponse {
	if r.ClaimToken == "" {
		r.ClaimToken = MakeRandomString(32)
	}
	code := firstNonEmpty(r.Code, r.RewardContent)
	return &model.ClaimResponse{Success: true, ClaimID: r.ID, ClaimToken: r.ClaimToken, CampaignID: r.CampaignID, NodeID: r.NodeID, Code: code, RewardContent: code, ClaimedAt: r.CreatedAt, Message: "领取成功"}
}
func nowISO() string { return time.Now().Format(timeLayout) }
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func defaultString(v, f string) string {
	if strings.TrimSpace(v) == "" {
		return f
	}
	return v
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func appendUnique(list []string, value string) []string {
	if value == "" {
		return list
	}
	for _, v := range list {
		if v == value {
			return list
		}
	}
	return append(list, value)
}
func removeString(list []string, value string) []string {
	out := list[:0]
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}
func containsAny(q string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	return false
}
func safeCampaignName(c *model.Campaign) string {
	if c == nil {
		return ""
	}
	return c.Name
}
func safeProjectName(p *model.Project) string {
	if p == nil {
		return ""
	}
	return p.Name
}
func safeNodeName(n *model.DistributionNode) string {
	if n == nil {
		return ""
	}
	return n.Name
}
func campaignRequestProjectID(req model.CreateCampaignRequest) string {
	return strings.TrimSpace(firstNonEmpty(req.ProjectID, req.ProjectIDSnake))
}
func paginateMaps(rows []map[string]interface{}, q map[string]string) model.PageResult {
	page := queryInt(q, "page", 1)
	size := queryInt(q, "pageSize", 20)
	if size <= 0 {
		size = 20
	}
	total := len(rows)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return model.PageResult{Items: rows[start:end], Total: total, Page: page, PageSize: size}
}
func queryInt(q map[string]string, key string, def int) int {
	v, err := strconv.Atoi(q[key])
	if err != nil {
		return def
	}
	return v
}
func configInt(m map[string]interface{}, key string, def int) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func limitMaps(rows []map[string]interface{}, n int) []map[string]interface{} {
	if len(rows) > n {
		return rows[:n]
	}
	return rows
}
func limitClaims(rows []model.ClaimRecord, n int) []model.ClaimRecord {
	if len(rows) > n {
		return rows[:n]
	}
	return rows
}
func isToday(value string) bool {
	t, err := time.Parse(timeLayout, value)
	if err != nil {
		return false
	}
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}
func rangeBounds(q map[string]string) (time.Time, time.Time) {
	now := time.Now()
	switch q["range"] {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), now
	case "30d":
		return now.AddDate(0, 0, -29), now
	default:
		return now.AddDate(0, 0, -6), now
	}
}
func dateBuckets(start, end time.Time) []string {
	out := []string{}
	for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()); !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}
func addTrendValue(rows []map[string]interface{}, iso string, value int) {
	if t, err := time.Parse(timeLayout, iso); err == nil {
		day := t.Format("2006-01-02")
		for _, r := range rows {
			if r["date"] == day {
				r["value"] = toInt(r["value"]) + value
			}
		}
	}
}
func addClaimTrend(rows []map[string]interface{}, iso, key string) {
	if t, err := time.Parse(timeLayout, iso); err == nil {
		day := t.Format("2006-01-02")
		for _, r := range rows {
			if r["date"] == day {
				r[key] = toInt(r[key]) + 1
			}
		}
	}
}
func inRange(iso string, start, end time.Time) bool {
	t, err := time.Parse(timeLayout, iso)
	if err != nil {
		return true
	}
	return !t.Before(start) && !t.After(end)
}
func reasonRows(m map[string]int) []map[string]interface{} {
	rows := []map[string]interface{}{}
	for k, v := range m {
		rows = append(rows, map[string]interface{}{"reason": k, "count": v})
	}
	sort.Slice(rows, func(i, j int) bool { return toInt(rows[i]["count"]) > toInt(rows[j]["count"]) })
	return rows
}
func percent(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
func toInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}
func countEnabledRules(m map[string]*model.RiskRule) int {
	n := 0
	for _, r := range m {
		if r.Enabled {
			n++
		}
	}
	return n
}
func countEnabledBlacklist(m map[string]*model.BlacklistItem) int {
	n := 0
	for _, r := range m {
		if r.Enabled {
			n++
		}
	}
	return n
}

func MakeRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}
