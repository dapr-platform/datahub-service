/*
 * @module service/rate_limiter/redis_rate_limiter_test
 * @description Redis限流器单元测试和性能测试
 * @architecture 测试层
 * @documentReference ai_docs/rate_limit_design.md
 */

package rate_limiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 设置测试用Redis环境
func setupTestRedis(t *testing.T) *RedisRateLimiter {
	// 从环境变量读取Redis配置（与start.sh保持一致）
	os.Setenv("REDIS_HOST", getEnvWithDefault("REDIS_HOST", "localhost"))
	os.Setenv("REDIS_PORT", getEnvWithDefault("REDIS_PORT", "6379"))
	os.Setenv("REDIS_PASSWORD", os.Getenv("REDIS_PASSWORD"))
	os.Setenv("REDIS_DB", getEnvWithDefault("REDIS_DB", "0"))

	limiter, err := NewRedisRateLimiter()
	require.NoError(t, err, "Redis限流器初始化失败")
	require.NotNil(t, limiter, "Redis限流器不应为nil")

	// 清理测试数据
	ctx := context.Background()
	limiter.client.FlushDB(ctx)

	return limiter
}

// TestNewRedisRateLimiter 测试创建Redis限流器
func TestNewRedisRateLimiter(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	assert.NotNil(t, limiter.client, "Redis客户端不应为nil")
}

// TestCheckRateLimit_SingleRule_Success 测试单个规则限流成功
func TestCheckRateLimit_SingleRule_Success(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "global",
		TargetID:    "",
		TimeWindow:  60,
		MaxRequests: 10,
	}

	// 第一次请求应该成功
	result, err := limiter.checkSingleRule(ctx, rule)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "第一次请求应该被允许")
	assert.Equal(t, 10, result.Limit, "限制数应该为10")
	assert.Equal(t, 9, result.Remaining, "剩余数应该为9")
	assert.Equal(t, "global", result.RateLimitType)
}

// TestCheckRateLimit_SingleRule_RateLimited 测试单个规则触发限流
func TestCheckRateLimit_SingleRule_RateLimited(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "api_key",
		TargetID:    "test-key-123",
		TimeWindow:  10,
		MaxRequests: 5,
	}

	// 发送5次请求
	for i := 0; i < 5; i++ {
		result, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)
		assert.True(t, result.Allowed, fmt.Sprintf("第%d次请求应该被允许", i+1))
		assert.Equal(t, 5-i-1, result.Remaining, fmt.Sprintf("第%d次请求剩余数应该为%d", i+1, 5-i-1))
	}

	// 第6次请求应该被限流
	result, err := limiter.checkSingleRule(ctx, rule)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "第6次请求应该被限流")
	assert.Equal(t, 0, result.Remaining, "剩余数应该为0")
	assert.Contains(t, result.Message, "API密钥限流限制")
}

// TestCheckRateLimit_MultipleRules_Priority 测试多层限流优先级
func TestCheckRateLimit_MultipleRules_Priority(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rules := []RateLimitRule{
		{Type: "global", TargetID: "", TimeWindow: 60, MaxRequests: 100},
		{Type: "api_key", TargetID: "key-123", TimeWindow: 60, MaxRequests: 50},
		{Type: "application", TargetID: "app-456", TimeWindow: 60, MaxRequests: 10},
	}

	// 应该按优先级检查：application > api_key > global
	// 发送10次请求
	for i := 0; i < 10; i++ {
		result, err := limiter.CheckRateLimit(ctx, rules)
		require.NoError(t, err)
		assert.True(t, result.Allowed, fmt.Sprintf("第%d次请求应该被允许", i+1))
	}

	// 第11次请求应该被应用层限流
	result, err := limiter.CheckRateLimit(ctx, rules)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "第11次请求应该被限流")
	assert.Equal(t, "application", result.RateLimitType, "应该是应用层触发限流")
}

// TestCheckRateLimit_NoRules 测试没有限流规则的情况
func TestCheckRateLimit_NoRules(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rules := []RateLimitRule{}

	result, err := limiter.CheckRateLimit(ctx, rules)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "没有限流规则应该允许通过")
	assert.Equal(t, "none", result.RateLimitType)
	assert.Equal(t, -1, result.Limit)
}

// TestCheckRateLimit_GlobalRateLimitUnique 测试全局限流Key唯一性
func TestCheckRateLimit_GlobalRateLimitUnique(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "global",
		TargetID:    "",
		TimeWindow:  60,
		MaxRequests: 5,
	}

	// 发送3次请求
	for i := 0; i < 3; i++ {
		_, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)
	}

	// 再次检查，应该累计计数
	result, err := limiter.checkSingleRule(ctx, rule)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Remaining, "全局限流应该累计计数，剩余1次")
}

// TestCheckRateLimit_ResetAfterWindow 测试时间窗口重置
func TestCheckRateLimit_ResetAfterWindow(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "api_key",
		TargetID:    "test-key-456",
		TimeWindow:  2, // 2秒时间窗口
		MaxRequests: 3,
	}

	// 发送3次请求，用完配额
	for i := 0; i < 3; i++ {
		result, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// 第4次请求应该被限流
	result, err := limiter.checkSingleRule(ctx, rule)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// 等待时间窗口重置
	time.Sleep(3 * time.Second)

	// 时间窗口重置后，应该可以再次请求
	result, err = limiter.checkSingleRule(ctx, rule)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "时间窗口重置后应该允许请求")
	assert.Equal(t, 2, result.Remaining, "重置后剩余数应该为2")
}

// TestGetStats 测试获取限流统计信息
func TestGetStats(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "application",
		TargetID:    "app-789",
		TimeWindow:  60,
		MaxRequests: 20,
	}

	// 发送5次请求
	for i := 0; i < 5; i++ {
		_, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)
	}

	// 获取统计信息
	stats, err := limiter.GetStats(ctx, rule)
	require.NoError(t, err)
	assert.Equal(t, "application", stats["type"])
	assert.Equal(t, "app-789", stats["target_id"])
	assert.Equal(t, 5, stats["current"], "当前计数应该为5")
	assert.Equal(t, 20, stats["limit"], "限制数应该为20")
	assert.Equal(t, 15, stats["remaining"], "剩余数应该为15")
}

// TestResetRateLimit 测试重置限流计数
func TestResetRateLimit(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "api_key",
		TargetID:    "reset-test-key",
		TimeWindow:  60,
		MaxRequests: 10,
	}

	// 发送5次请求
	for i := 0; i < 5; i++ {
		_, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)
	}

	// 验证计数
	stats, err := limiter.GetStats(ctx, rule)
	require.NoError(t, err)
	assert.Equal(t, 5, stats["current"])

	// 重置计数
	err = limiter.ResetRateLimit(ctx, rule)
	require.NoError(t, err)

	// 验证重置后计数为0
	stats, err = limiter.GetStats(ctx, rule)
	require.NoError(t, err)
	assert.Equal(t, 0, stats["current"], "重置后计数应该为0")
}

// TestSortRulesByPriority 测试规则优先级排序
func TestSortRulesByPriority(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	rules := []RateLimitRule{
		{Type: "global", TimeWindow: 60, MaxRequests: 1000},
		{Type: "api_key", TargetID: "key-1", TimeWindow: 60, MaxRequests: 100},
		{Type: "application", TargetID: "app-1", TimeWindow: 60, MaxRequests: 50},
	}

	sorted := limiter.sortRulesByPriority(rules)
	assert.Equal(t, "application", sorted[0].Type, "第一个应该是application")
	assert.Equal(t, "api_key", sorted[1].Type, "第二个应该是api_key")
	assert.Equal(t, "global", sorted[2].Type, "第三个应该是global")
}

// TestBuildRateLimitKey 测试限流Key构造
func TestBuildRateLimitKey(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	// 测试全局限流Key
	globalKey := limiter.buildRateLimitKey("global", "", 60)
	assert.Contains(t, globalKey, "rate_limit:global")

	// 测试API密钥限流Key
	keyKey := limiter.buildRateLimitKey("api_key", "key-123", 60)
	assert.Contains(t, keyKey, "rate_limit:api_key:key-123")

	// 测试应用限流Key
	appKey := limiter.buildRateLimitKey("application", "app-456", 60)
	assert.Contains(t, appKey, "rate_limit:application:app-456")
}

// === 性能测试 ===

// BenchmarkCheckRateLimit_SingleRule 基准测试：单个规则限流检查
func BenchmarkCheckRateLimit_SingleRule(b *testing.B) {
	limiter := setupTestRedis(&testing.T{})
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "global",
		TargetID:    "",
		TimeWindow:  60,
		MaxRequests: 1000000, // 设置足够大，避免触发限流
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.checkSingleRule(ctx, rule)
	}
}

// BenchmarkCheckRateLimit_MultipleRules 基准测试：多层规则限流检查
func BenchmarkCheckRateLimit_MultipleRules(b *testing.B) {
	limiter := setupTestRedis(&testing.T{})
	defer limiter.Close()

	ctx := context.Background()
	rules := []RateLimitRule{
		{Type: "global", TargetID: "", TimeWindow: 60, MaxRequests: 1000000},
		{Type: "api_key", TargetID: "bench-key", TimeWindow: 60, MaxRequests: 1000000},
		{Type: "application", TargetID: "bench-app", TimeWindow: 60, MaxRequests: 1000000},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.CheckRateLimit(ctx, rules)
	}
}

// TestConcurrentRateLimitCheck 并发测试：多个goroutine同时检查限流
func TestConcurrentRateLimitCheck(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "api_key",
		TargetID:    "concurrent-key",
		TimeWindow:  60,
		MaxRequests: 100,
	}

	var wg sync.WaitGroup
	allowedCount := 0
	deniedCount := 0
	var mu sync.Mutex

	// 启动200个goroutine并发请求
	concurrency := 200
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := limiter.checkSingleRule(ctx, rule)
			require.NoError(t, err)

			mu.Lock()
			if result.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 验证结果
	t.Logf("允许请求: %d, 拒绝请求: %d", allowedCount, deniedCount)
	assert.Equal(t, 100, allowedCount, "应该有100个请求被允许")
	assert.Equal(t, 100, deniedCount, "应该有100个请求被拒绝")
}

// TestHighThroughput 高吞吐量测试
func TestHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过高吞吐量测试")
	}

	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "global",
		TargetID:    "",
		TimeWindow:  10,
		MaxRequests: 10000, // 10秒内允许10000次请求
	}

	startTime := time.Now()
	successCount := 0
	failCount := 0

	// 快速发送10000次请求
	for i := 0; i < 10000; i++ {
		result, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)

		if result.Allowed {
			successCount++
		} else {
			failCount++
		}
	}

	duration := time.Since(startTime)
	qps := float64(10000) / duration.Seconds()

	t.Logf("吞吐量测试结果:")
	t.Logf("  总请求数: 10000")
	t.Logf("  成功请求: %d", successCount)
	t.Logf("  失败请求: %d", failCount)
	t.Logf("  耗时: %v", duration)
	t.Logf("  QPS: %.2f", qps)

	assert.Equal(t, 10000, successCount, "应该有10000个请求成功")
	assert.Equal(t, 0, failCount, "不应该有请求失败")
	assert.Greater(t, qps, 1000.0, "QPS应该大于1000")
}

// TestMemoryLeak 内存泄漏测试
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存泄漏测试")
	}

	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()

	// 创建大量不同的规则
	for i := 0; i < 1000; i++ {
		rule := RateLimitRule{
			Type:        "api_key",
			TargetID:    fmt.Sprintf("key-%d", i),
			TimeWindow:  5,
			MaxRequests: 10,
		}

		for j := 0; j < 10; j++ {
			_, err := limiter.checkSingleRule(ctx, rule)
			require.NoError(t, err)
		}
	}

	// 等待Redis自动过期
	time.Sleep(6 * time.Second)

	// 验证Redis中的Key数量（应该被清理）
	keys, err := limiter.client.Keys(ctx, "rate_limit:*").Result()
	require.NoError(t, err)
	t.Logf("Redis中剩余的Key数量: %d", len(keys))

	// 大部分Key应该已经过期被清理
	assert.Less(t, len(keys), 100, "大部分Key应该已经被Redis自动清理")
}

// BenchmarkConcurrentAccess 并发访问基准测试
func BenchmarkConcurrentAccess(b *testing.B) {
	limiter := setupTestRedis(&testing.T{})
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "api_key",
		TargetID:    "bench-concurrent-key",
		TimeWindow:  60,
		MaxRequests: 1000000,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = limiter.checkSingleRule(ctx, rule)
		}
	})
}

// TestRateLimitAccuracy 限流精确度测试
func TestRateLimitAccuracy(t *testing.T) {
	limiter := setupTestRedis(t)
	defer limiter.Close()

	ctx := context.Background()
	rule := RateLimitRule{
		Type:        "application",
		TargetID:    "accuracy-test-app",
		TimeWindow:  5,
		MaxRequests: 50,
	}

	allowedCount := 0
	deniedCount := 0

	// 发送100次请求
	for i := 0; i < 100; i++ {
		result, err := limiter.checkSingleRule(ctx, rule)
		require.NoError(t, err)

		if result.Allowed {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	t.Logf("精确度测试结果: 允许=%d, 拒绝=%d", allowedCount, deniedCount)
	assert.Equal(t, 50, allowedCount, "应该精确允许50次请求")
	assert.Equal(t, 50, deniedCount, "应该精确拒绝50次请求")
}
