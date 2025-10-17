/*
 * @module service/distributed_lock/redis_lock
 * @description Redis分布式锁实现，用于多实例环境下的同步任务调度防重
 * @architecture 工具层 - 提供分布式锁能力
 * @documentReference ai_docs/distributed_lock_design.md
 * @stateFlow 获取锁 -> 执行任务 -> 释放锁/自动过期
 * @rules 使用Redis SET NX实现，支持锁续期和自动过期
 * @dependencies github.com/redis/go-redis/v9
 * @refs service/init.go, service/basic_library/sync_task_service.go
 */

package distributed_lock

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Unlock 释放锁
	Unlock(ctx context.Context, key string) error
	// Refresh 刷新锁的过期时间
	Refresh(ctx context.Context, key string, ttl time.Duration) error
	// IsLocked 检查锁是否存在
	IsLocked(ctx context.Context, key string) (bool, error)
}

// RedisLock Redis分布式锁实现
type RedisLock struct {
	client     *redis.Client
	instanceID string // 实例ID，用于标识锁的持有者
}

// NewRedisLock 创建Redis分布式锁
func NewRedisLock() (*RedisLock, error) {
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
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	// 生成实例ID（使用主机名+进程ID）
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s:%d", hostname, os.Getpid())

	slog.Info("Redis分布式锁初始化成功",
		"instance_id", instanceID,
		"redis_host", host,
		"redis_port", port)

	return &RedisLock{
		client:     client,
		instanceID: instanceID,
	}, nil
}

// TryLock 尝试获取锁
// 使用SET NX命令，只有当key不存在时才会设置成功
func (r *RedisLock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// 构造锁的键
	lockKey := fmt.Sprintf("sync_task_scheduler:lock:%s", key)

	// 使用SET NX命令尝试获取锁
	result, err := r.client.SetNX(ctx, lockKey, r.instanceID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("获取锁失败: %w", err)
	}

	if result {
		slog.Debug("分布式锁: 成功获取锁",
			"key", key,
			"ttl", ttl,
			"instance", r.instanceID)
	}

	return result, nil
}

// Unlock 释放锁
// 使用Lua脚本确保只有锁的持有者才能释放锁
func (r *RedisLock) Unlock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("sync_task_scheduler:lock:%s", key)

	// Lua脚本：检查锁的持有者是否是当前实例，是则删除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, script, []string{lockKey}, r.instanceID).Result()
	if err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	if result.(int64) == 1 {
		slog.Debug("分布式锁: 成功释放锁",
			"key", key,
			"instance", r.instanceID)
	} else {
		slog.Warn("分布式锁: 锁不存在或已被其他实例持有",
			"key", key,
			"instance", r.instanceID)
	}

	return nil
}

// Refresh 刷新锁的过期时间
// 用于长时间运行的任务，防止锁过期
func (r *RedisLock) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	lockKey := fmt.Sprintf("sync_task_scheduler:lock:%s", key)

	// Lua脚本：检查锁的持有者是否是当前实例，是则刷新过期时间
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, script, []string{lockKey}, r.instanceID, int(ttl.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("刷新锁失败: %w", err)
	}

	if result.(int64) == 1 {
		slog.Debug("分布式锁: 成功刷新锁",
			"key", key,
			"ttl", ttl,
			"instance", r.instanceID)
		return nil
	}

	return fmt.Errorf("锁不存在或已被其他实例持有")
}

// IsLocked 检查锁是否存在
func (r *RedisLock) IsLocked(ctx context.Context, key string) (bool, error) {
	lockKey := fmt.Sprintf("sync_task_scheduler:lock:%s", key)

	exists, err := r.client.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, fmt.Errorf("检查锁状态失败: %w", err)
	}

	return exists > 0, nil
}

// Close 关闭Redis客户端
func (r *RedisLock) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LockExecutor 带锁执行器，用于简化锁的使用
type LockExecutor struct {
	lock DistributedLock
}

// NewLockExecutor 创建带锁执行器
func NewLockExecutor(lock DistributedLock) *LockExecutor {
	return &LockExecutor{lock: lock}
}

// ExecuteWithLock 在锁保护下执行函数
func (e *LockExecutor) ExecuteWithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	// 尝试获取锁
	locked, err := e.lock.TryLock(ctx, key, ttl)
	if err != nil {
		return fmt.Errorf("获取锁失败: %w", err)
	}

	if !locked {
		slog.Debug("分布式锁: 锁已被其他实例持有，跳过执行", "key", key)
		return nil // 不是错误，只是被其他实例执行了
	}

	// 确保函数执行完毕后释放锁
	defer func() {
		if unlockErr := e.lock.Unlock(ctx, key); unlockErr != nil {
			slog.Error("分布式锁: 释放锁失败", "key", key, "error", unlockErr)
		}
	}()

	// 执行函数
	return fn()
}

// ExecuteWithLockAndRefresh 在锁保护下执行函数，并自动续期
func (e *LockExecutor) ExecuteWithLockAndRefresh(ctx context.Context, key string, ttl time.Duration, refreshInterval time.Duration, fn func() error) error {
	// 尝试获取锁
	locked, err := e.lock.TryLock(ctx, key, ttl)
	if err != nil {
		return fmt.Errorf("获取锁失败: %w", err)
	}

	if !locked {
		slog.Debug("分布式锁: 锁已被其他实例持有，跳过执行", "key", key)
		return nil
	}

	// 创建一个可取消的上下文用于停止续期
	refreshCtx, cancelRefresh := context.WithCancel(ctx)
	defer cancelRefresh()

	// 启动自动续期
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-refreshCtx.Done():
				return
			case <-ticker.C:
				if refreshErr := e.lock.Refresh(ctx, key, ttl); refreshErr != nil {
					slog.Error("分布式锁: 续期失败", "key", key, "error", refreshErr)
				}
			}
		}
	}()

	// 确保函数执行完毕后释放锁
	defer func() {
		if unlockErr := e.lock.Unlock(ctx, key); unlockErr != nil {
			slog.Error("分布式锁: 释放锁失败", "key", key, "error", unlockErr)
		}
	}()

	// 执行函数
	return fn()
}
