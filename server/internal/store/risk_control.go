package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cdk-airdrop-station/server/internal/model"

	"github.com/redis/go-redis/v9"
)

// RiskEngine 风控引擎
type RiskEngine struct {
	redis *redis.Client
}

// NewRiskEngine 创建风控引擎
func NewRiskEngine(redis *redis.Client) *RiskEngine {
	return &RiskEngine{redis: redis}
}

// EvaluateRisk 评估领取风险
func (r *RiskEngine) EvaluateRisk(
	ctx context.Context,
	req *http.Request,
	userID string,
	fingerprint string,
	projectCode string,
) *model.RiskAssessment {

	assessment := &model.RiskAssessment{}
	ip := getClientIP(req)
	ua := req.UserAgent()

	// 1. IP 频率检测（5分钟内同一 IP 领取超过 5 次）
	ipKey := fmt.Sprintf("risk:ip:%s", ip)
	ipCount, _ := r.redis.Incr(ctx, ipKey).Result()
	r.redis.Expire(ctx, ipKey, 5*time.Minute)
	if ipCount > 5 {
		assessment.Score += 30
		assessment.Reasons = append(assessment.Reasons, "IP 频率异常")
	}

	// 2. 设备指纹检测（同一设备切换账号）
	deviceKey := fmt.Sprintf("risk:device:%s", fingerprint)
	deviceUsers, _ := r.redis.SCard(ctx, deviceKey).Result()
	r.redis.SAdd(ctx, deviceKey, userID)
	r.redis.Expire(ctx, deviceKey, 1*time.Hour)
	if deviceUsers > 2 {
		assessment.Score += 40
		assessment.Reasons = append(assessment.Reasons, "设备多账号异常")
	}

	// 3. User-Agent 检测（爬虫/脚本特征）
	if isSuspiciousUA(ua) {
		assessment.Score += 25
		assessment.Reasons = append(assessment.Reasons, "可疑 User-Agent")
	}

	// 4. 生成设备指纹哈希（用于跨请求追踪）
	fingerprintHash := generateFingerprintHash(ip, ua, fingerprint)
	hashKey := fmt.Sprintf("risk:hash:%s", fingerprintHash)
	hashCount, _ := r.redis.Incr(ctx, hashKey).Result()
	r.redis.Expire(ctx, hashKey, 1*time.Hour)
	if hashCount > 10 {
		assessment.Score += 35
		assessment.Reasons = append(assessment.Reasons, "指纹哈希频率异常")
	}

	// 5. 同一项目短时间多次尝试
	projectKey := fmt.Sprintf("risk:project:%s:%s", projectCode, userID)
	projectAttempts, _ := r.redis.Incr(ctx, projectKey).Result()
	r.redis.Expire(ctx, projectKey, 10*time.Minute)
	if projectAttempts > 3 {
		assessment.Score += 20
		assessment.Reasons = append(assessment.Reasons, "项目领取尝试次数过多")
	}

	// 确定风险等级
	switch {
	case assessment.Score >= 80:
		assessment.Level = "blocked"
	case assessment.Score >= 60:
		assessment.Level = "high"
	case assessment.Score >= 30:
		assessment.Level = "medium"
	default:
		assessment.Level = "low"
	}

	return assessment
}

// IsBlocked 检查是否被封禁
func (r *RiskEngine) IsBlocked(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("risk:blocked:%s", userID)
	exists, err := r.redis.Exists(ctx, key).Result()
	return exists > 0, err
}

// BlockUser 封禁用户
func (r *RiskEngine) BlockUser(ctx context.Context, userID string, duration time.Duration) error {
	key := fmt.Sprintf("risk:blocked:%s", userID)
	return r.redis.Set(ctx, key, "1", duration).Err()
}

// RecordClaimAttempt 记录领取尝试
func (r *RiskEngine) RecordClaimAttempt(
	ctx context.Context,
	userID string,
	ip string,
	fingerprint string,
	projectCode string,
	success bool,
) {
	// 记录到 Redis Sorted Set（用于统计）
	key := "risk:claim_attempts"
	member := fmt.Sprintf("%s:%s:%s:%s:%v", userID, ip, fingerprint, projectCode, success)
	r.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: member,
	})
	// 只保留最近 24 小时的记录
	r.redis.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", time.Now().Unix()-86400))
}

// generateFingerprintHash 生成设备指纹哈希
func generateFingerprintHash(ip, ua, fingerprint string) string {
	h := sha256.New()
	h.Write([]byte(ip + "|" + ua + "|" + fingerprint))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// isSuspiciousUA 检测可疑 User-Agent
func isSuspiciousUA(ua string) bool {
	suspicious := []string{
		"python", "curl", "wget", "httpclient",
		"scrapy", "selenium", "puppeteer", "headless",
		"phantomjs", "nightmare", "playwright",
	}
	lowerUA := strings.ToLower(ua)
	for _, s := range suspicious {
		if strings.Contains(lowerUA, s) {
			return true
		}
	}
	return false
}

// getClientIP 获取真实 IP
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
