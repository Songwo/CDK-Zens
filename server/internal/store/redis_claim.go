package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本：原子化检查库存 + 防重领 + 扣减
// 返回值：
//   >= 0 : 剩余库存数量
//   -1   : 用户已领取过
//   -2   : IP 超限
//   -3   : 设备超限
//   -4   : 已售罄
const claimLuaScript = `
local stock_key = KEYS[1]
local claimed_key = KEYS[2]
local ip_key = KEYS[3]
local device_key = KEYS[4]

local user_id = ARGV[1]
local ip = ARGV[2]
local fingerprint = ARGV[3]
local ip_limit = tonumber(ARGV[4])
local device_limit = tonumber(ARGV[5])

-- 1. 检查用户是否已领取
local user_claimed = redis.call('SISMEMBER', claimed_key, user_id)
if user_claimed == 1 then
    return -1
end

-- 2. 检查 IP 限制
if ip_limit > 0 then
    local ip_count = redis.call('SCARD', ip_key)
    if ip_count >= ip_limit then
        return -2
    end
end

-- 3. 检查设备限制
if device_limit > 0 then
    local device_count = redis.call('SCARD', device_key)
    if device_count >= device_limit then
        return -3
    end
end

-- 4. 原子扣减库存
local stock = redis.call('DECR', stock_key)
if stock < 0 then
    redis.call('INCR', stock_key)
    return -4
end

-- 5. 记录领取
redis.call('SADD', claimed_key, user_id)
if ip_limit > 0 then
    redis.call('SADD', ip_key, ip)
end
if device_limit > 0 then
    redis.call('SADD', device_key, fingerprint)
end

-- 6. 设置过期时间（24小时后清理限制记录）
redis.call('EXPIRE', claimed_key, 86400)
if ip_limit > 0 then
    redis.call('EXPIRE', ip_key, 86400)
end
if device_limit > 0 then
    redis.call('EXPIRE', device_key, 86400)
end

return stock
`

// RedisClaimer Redis 高并发领取器
type RedisClaimer struct {
	client *redis.Client
	script *redis.Script
}

// NewRedisClaimer 创建 Redis 领取器
func NewRedisClaimer(client *redis.Client) *RedisClaimer {
	return &RedisClaimer{
		client: client,
		script: redis.NewScript(claimLuaScript),
	}
}

// ClaimResult 领取结果
type ClaimResult struct {
	Success   bool
	Remaining int
	Error     string
}

// ExecuteClaim 执行领取（原子操作）
func (r *RedisClaimer) ExecuteClaim(
	ctx context.Context,
	projectCode string,
	userID string,
	ip string,
	fingerprint string,
	ipLimit int,
	deviceLimit int,
) (*ClaimResult, error) {

	keys := []string{
		fmt.Sprintf("zenspulse:stock:%s", projectCode),
		fmt.Sprintf("zenspulse:claimed:%s", projectCode),
		fmt.Sprintf("zenspulse:ip:%s:%s", projectCode, ip),
		fmt.Sprintf("zenspulse:device:%s:%s", projectCode, fingerprint),
	}

	result, err := r.script.Run(
		ctx,
		r.client,
		keys,
		userID,
		ip,
		fingerprint,
		ipLimit,
		deviceLimit,
	).Int()

	if err != nil {
		return nil, fmt.Errorf("redis claim failed: %w", err)
	}

	switch result {
	case -1:
		return &ClaimResult{Error: "您已领取过"}, nil
	case -2:
		return &ClaimResult{Error: "当前 IP 领取次数已达上限"}, nil
	case -3:
		return &ClaimResult{Error: "当前设备领取次数已达上限"}, nil
	case -4:
		return &ClaimResult{Error: "已被领完"}, nil
	default:
		return &ClaimResult{Success: true, Remaining: result}, nil
	}
}

// SyncStockToRedis 同步库存到 Redis
func (r *RedisClaimer) SyncStockToRedis(ctx context.Context, projectCode string, stock int) error {
	key := fmt.Sprintf("zenspulse:stock:%s", projectCode)
	return r.client.Set(ctx, key, stock, 0).Err()
}

// GetRemainingStock 获取剩余库存
func (r *RedisClaimer) GetRemainingStock(ctx context.Context, projectCode string) (int, error) {
	key := fmt.Sprintf("zenspulse:stock:%s", projectCode)
	val, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// IsUserClaimed 检查用户是否已领取
func (r *RedisClaimer) IsUserClaimed(ctx context.Context, projectCode string, userID string) (bool, error) {
	key := fmt.Sprintf("zenspulse:claimed:%s", projectCode)
	return r.client.SIsMember(ctx, key, userID).Result()
}

// GetClaimedUsers 获取已领取用户列表
func (r *RedisClaimer) GetClaimedUsers(ctx context.Context, projectCode string) ([]string, error) {
	key := fmt.Sprintf("zenspulse:claimed:%s", projectCode)
	return r.client.SMembers(ctx, key).Result()
}

// ResetProject 重置项目（用于测试或管理员操作）
func (r *RedisClaimer) ResetProject(ctx context.Context, projectCode string) error {
	keys := []string{
		fmt.Sprintf("zenspulse:stock:%s", projectCode),
		fmt.Sprintf("zenspulse:claimed:%s", projectCode),
	}
	return r.client.Del(ctx, keys...).Err()
}

// GetIPCount 获取某 IP 在该项目的领取次数
func (r *RedisClaimer) GetIPCount(ctx context.Context, projectCode string, ip string) (int64, error) {
	key := fmt.Sprintf("zenspulse:ip:%s:%s", projectCode, ip)
	return r.client.SCard(ctx, key).Result()
}

// GetDeviceCount 获取某设备在该项目的领取次数
func (r *RedisClaimer) GetDeviceCount(ctx context.Context, projectCode string, fingerprint string) (int64, error) {
	key := fmt.Sprintf("zenspulse:device:%s:%s", projectCode, fingerprint)
	return r.client.SCard(ctx, key).Result()
}
