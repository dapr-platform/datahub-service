/*
 * @module service/rate_limiter/redis_rate_limiter
 * @description 基于Redis的分布式限流服务，支持全局、API Key、应用三层限流
 * @architecture 工具层 - 提供分布式限流能力
 * @documentReference ai_docs/rate_limit_design.md
 * @stateFlow 检查限流规则 -> Redis计数 -> 判断是否超限
 * @rules 使用Redis INCR和EXPIRE实现滑动窗口限流
 * @dependencies github.com/redis/go-redis/v9
 * @refs service/sharing/sharing_service.go, controllers/data_proxy_controller.go
 */

package rate_limiter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimitResult 限流检查结果
type RateLimitResult struct {
	Allowed       bool   `json:"allowed"`    // 是否允许请求
	Limit         int    `json:"limit"`      // 限制数量
	Remaining     int    `json:"remaining"`  // 剩余数量
	ResetAt       int64  `json:"reset_at"`   // 重置时间（Unix时间戳）
	RateLimitType string `json:"limit_type"` // 限流类型：global/api_key/application
	Message       string `json:"message"`    // 提示信息
}

// RateLimitRule 限流规则
type RateLimitRule struct {
	Type        string // global/api_key/application
	TargetID    string // 目标ID（api_key_id或application_id，全局时为空）
	TimeWindow  int    // 时间窗口（秒）
	MaxRequests int    // 最大请求数
}

// RedisRateLimiter Redis限流器
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter 创建Redis限流器
func NewRedisRateLimiter() (*RedisRateLimiter, error) {
	// 从环境变量读取Redis配置
	host := getEnvWithDefault("REDIS_HOST", "localhost")
	port := getEnvWithDefault("REDIS_PORT", "6379")
	password := os.Getenv("REDIS_PASSWORD")
	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		fmt.Sscanf(dbStr, "%d", &db)
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	slog.Info("Redis限流器初始化成功",
		"redis_host", host,
		"redis_port", port)

	return &RedisRateLimiter{
		client: client,
	}, nil
}

// CheckRateLimit 检查是否超过限流（按优先级检查：应用 -> 密钥 -> 全局）
func (r *RedisRateLimiter) CheckRateLimit(ctx context.Context, rules []RateLimitRule) (*RateLimitResult, error) {
	// 按优先级排序：application > api_key > global
	sortedRules := r.sortRulesByPriority(rules)

	// 依次检查每层限流
	for _, rule := range sortedRules {
		result, err := r.checkSingleRule(ctx, rule)
		if err != nil {
			return nil, err
		}

		// 如果任何一层超限，直接返回
		if !result.Allowed {
			return result, nil
		}
	}

	// 所有层都未超限，返回最宽松的限制信息
	if len(sortedRules) > 0 {
		lastRule := sortedRules[len(sortedRules)-1]
		return r.checkSingleRule(ctx, lastRule)
	}

	// 没有限流规则，允许通过
	return &RateLimitResult{
		Allowed:       true,
		Limit:         -1,
		Remaining:     -1,
		RateLimitType: "none",
		Message:       "无限流规则",
	}, nil
}

// checkSingleRule 检查单个限流规则
func (r *RedisRateLimiter) checkSingleRule(ctx context.Context, rule RateLimitRule) (*RateLimitResult, error) {
	// 构造Redis Key
	key := r.buildRateLimitKey(rule.Type, rule.TargetID, rule.TimeWindow)

	// 使用Lua脚本实现原子性限流检查
	script := `
		local key = KEYS[1]
		local max_requests = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local current_time = tonumber(ARGV[3])

		-- 获取当前计数
		local current = redis.call('GET', key)
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end

		-- 检查是否超限
		if current >= max_requests then
			local ttl = redis.call('TTL', key)
			if ttl == -1 then
				ttl = window
			end
			return {0, current, max_requests, ttl}
		end

		-- 增加计数
		local new_count = redis.call('INCR', key)
		
		-- 如果是第一次请求，设置过期时间
		if new_count == 1 then
			redis.call('EXPIRE', key, window)
		end

		local ttl = redis.call('TTL', key)
		if ttl == -1 then
			ttl = window
		end

		return {1, new_count, max_requests, ttl}
	`

	currentTime := time.Now().Unix()
	result, err := r.client.Eval(ctx, script, []string{key}, rule.MaxRequests, rule.TimeWindow, currentTime).Result()
	if err != nil {
		return nil, fmt.Errorf("限流检查失败: %w", err)
	}

	// 解析结果
	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	currentCount := int(results[1].(int64))
	maxRequests := int(results[2].(int64))
	ttl := int(results[3].(int64))

	resetAt := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	remaining := maxRequests - currentCount
	if remaining < 0 {
		remaining = 0
	}

	message := "允许请求"
	if !allowed {
		message = fmt.Sprintf("超过%s限流限制", r.getRateLimitTypeName(rule.Type))
	}

	return &RateLimitResult{
		Allowed:       allowed,
		Limit:         maxRequests,
		Remaining:     remaining,
		ResetAt:       resetAt,
		RateLimitType: rule.Type,
		Message:       message,
	}, nil
}

// buildRateLimitKey 构造限流Key
func (r *RedisRateLimiter) buildRateLimitKey(limitType, targetID string, window int) string {
	baseKey := "rate_limit"
	currentWindow := time.Now().Unix() / int64(window)

	if limitType == "global" {
		return fmt.Sprintf("%s:%s:%d", baseKey, limitType, currentWindow)
	}
	return fmt.Sprintf("%s:%s:%s:%d", baseKey, limitType, targetID, currentWindow)
}

// sortRulesByPriority 按优先级排序规则：application > api_key > global
func (r *RedisRateLimiter) sortRulesByPriority(rules []RateLimitRule) []RateLimitRule {
	priorityMap := map[string]int{
		"application": 3,
		"api_key":     2,
		"global":      1,
	}

	sorted := make([]RateLimitRule, len(rules))
	copy(sorted, rules)

	// 简单冒泡排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if priorityMap[sorted[j].Type] < priorityMap[sorted[j+1].Type] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// getRateLimitTypeName 获取限流类型名称
func (r *RedisRateLimiter) getRateLimitTypeName(limitType string) string {
	switch limitType {
	case "global":
		return "全局"
	case "api_key":
		return "API密钥"
	case "application":
		return "应用"
	default:
		return "未知"
	}
}

// Close 关闭Redis客户端
func (r *RedisRateLimiter) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// GetStats 获取限流统计信息
func (r *RedisRateLimiter) GetStats(ctx context.Context, rule RateLimitRule) (map[string]interface{}, error) {
	key := r.buildRateLimitKey(rule.Type, rule.TargetID, rule.TimeWindow)

	current, err := r.client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	remaining := rule.MaxRequests - current
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"type":        rule.Type,
		"target_id":   rule.TargetID,
		"current":     current,
		"limit":       rule.MaxRequests,
		"remaining":   remaining,
		"window":      rule.TimeWindow,
		"ttl_seconds": int(ttl.Seconds()),
		"reset_at":    time.Now().Add(ttl).Unix(),
	}, nil
}

// ResetRateLimit 重置限流计数（仅用于测试或管理）
func (r *RedisRateLimiter) ResetRateLimit(ctx context.Context, rule RateLimitRule) error {
	key := r.buildRateLimitKey(rule.Type, rule.TargetID, rule.TimeWindow)
	return r.client.Del(ctx, key).Err()
}

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
