/*
 * @module RedisConnector
 * @description Redis连接器，提供Redis客户端的封装，支持数据监听、缓存操作和发布订阅功能
 * @architecture 适配器模式 - 封装第三方Redis客户端，提供统一的接口
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 连接建立 -> 数据操作/监听 -> 连接断开
 * @rules 支持连接池、事务、流水线、发布订阅
 * @dependencies github.com/go-redis/redis/v8, encoding/json
 * @refs service/models/sync_models.go, service/utils/data_converter.go
 */
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConnector Redis连接器结构体
type RedisConnector struct {
	config        *RedisConfig
	client        *redis.Client
	clusterClient *redis.ClusterClient
	pubsub        *redis.PubSub
	logger        *log.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	isConnected   bool
	isCluster     bool
	subscribers   map[string]RedisMessageHandler // 频道订阅处理器映射
	mutex         sync.RWMutex
	stats         *RedisStats
}

// RedisConfig Redis配置信息
type RedisConfig struct {
	Addresses     []string          `json:"addresses"`      // Redis地址列表（集群模式）
	Address       string            `json:"address"`        // Redis地址（单机模式）
	Password      string            `json:"password"`       // 密码
	Database      int               `json:"database"`       // 数据库编号
	PoolSize      int               `json:"pool_size"`      // 连接池大小
	MinIdleConns  int               `json:"min_idle_conns"` // 最小空闲连接数
	MaxRetries    int               `json:"max_retries"`    // 最大重试次数
	RetryDelay    time.Duration     `json:"retry_delay"`    // 重试延迟
	DialTimeout   time.Duration     `json:"dial_timeout"`   // 连接超时时间
	ReadTimeout   time.Duration     `json:"read_timeout"`   // 读取超时时间
	WriteTimeout  time.Duration     `json:"write_timeout"`  // 写入超时时间
	PoolTimeout   time.Duration     `json:"pool_timeout"`   // 连接池超时时间
	IdleTimeout   time.Duration     `json:"idle_timeout"`   // 空闲连接超时时间
	MaxConnAge    time.Duration     `json:"max_conn_age"`   // 连接最大存活时间
	Channels      []string          `json:"channels"`       // 订阅的频道列表
	IsCluster     bool              `json:"is_cluster"`     // 是否集群模式
	TLSConfig     *RedisTLSConfig   `json:"tls_config"`     // TLS配置
	CustomOptions map[string]string `json:"custom_options"` // 自定义选项
}

// RedisTLSConfig Redis TLS配置
type RedisTLSConfig struct {
	EnableTLS    bool   `json:"enable_tls"`    // 是否启用TLS
	CertFile     string `json:"cert_file"`     // 证书文件
	KeyFile      string `json:"key_file"`      // 密钥文件
	CAFile       string `json:"ca_file"`       // CA证书文件
	InsecureSkip bool   `json:"insecure_skip"` // 是否跳过证书验证
}

// RedisMessage Redis消息结构体
type RedisMessage struct {
	Channel   string      `json:"channel"`   // 频道
	Pattern   string      `json:"pattern"`   // 模式（模式订阅时使用）
	Payload   interface{} `json:"payload"`   // 消息载荷
	Timestamp time.Time   `json:"timestamp"` // 时间戳
}

// RedisMessageHandler Redis消息处理函数类型
type RedisMessageHandler func(*RedisMessage) error

// RedisStats Redis连接器统计信息
type RedisStats struct {
	ConnectedAt      time.Time `json:"connected_at"`      // 连接时间
	CommandsExecuted int64     `json:"commands_executed"` // 执行命令数
	BytesRead        int64     `json:"bytes_read"`        // 读取字节数
	BytesWritten     int64     `json:"bytes_written"`     // 写入字节数
	MessagesReceived int64     `json:"messages_received"` // 接收消息数
	MessagesSent     int64     `json:"messages_sent"`     // 发送消息数
	ReconnectCount   int       `json:"reconnect_count"`   // 重连次数
	LastError        string    `json:"last_error"`        // 最后错误信息
	mutex            sync.RWMutex
}

// NewRedisConnector 创建新的Redis连接器
func NewRedisConnector(config *RedisConfig, logger *log.Logger) *RedisConnector {
	ctx, cancel := context.WithCancel(context.Background())

	connector := &RedisConnector{
		config:      config,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		isConnected: false,
		isCluster:   config.IsCluster,
		subscribers: make(map[string]RedisMessageHandler),
		stats:       &RedisStats{},
	}

	// 创建Redis客户端
	if config.IsCluster {
		// 集群模式
		connector.clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Addresses,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			IdleTimeout:  config.IdleTimeout,
			MaxConnAge:   config.MaxConnAge,
		})
	} else {
		// 单机模式
		connector.client = redis.NewClient(&redis.Options{
			Addr:         config.Address,
			Password:     config.Password,
			DB:           config.Database,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			IdleTimeout:  config.IdleTimeout,
			MaxConnAge:   config.MaxConnAge,
		})
	}

	return connector
}

// Connect 建立Redis连接
func (rc *RedisConnector) Connect() error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if rc.isConnected {
		return nil
	}

	rc.logger.Printf("正在连接Redis...")

	// 测试连接
	var err error
	if rc.isCluster {
		_, err = rc.clusterClient.Ping(rc.ctx).Result()
	} else {
		_, err = rc.client.Ping(rc.ctx).Result()
	}

	if err != nil {
		rc.updateError(fmt.Sprintf("Redis连接失败: %v", err))
		return fmt.Errorf("Redis连接失败: %v", err)
	}

	rc.isConnected = true
	rc.stats.ConnectedAt = time.Now()
	rc.logger.Printf("Redis连接器已连接")

	// 订阅预配置的频道
	if len(rc.config.Channels) > 0 {
		if err := rc.subscribeChannels(rc.config.Channels); err != nil {
			rc.logger.Printf("订阅频道失败: %v", err)
		}
	}

	return nil
}

// Disconnect 断开Redis连接
func (rc *RedisConnector) Disconnect() error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if !rc.isConnected {
		return nil
	}

	rc.logger.Println("正在断开Redis连接...")

	// 关闭发布订阅
	if rc.pubsub != nil {
		if err := rc.pubsub.Close(); err != nil {
			rc.logger.Printf("关闭发布订阅失败: %v", err)
		}
		rc.pubsub = nil
	}

	// 关闭客户端
	var err error
	if rc.isCluster {
		err = rc.clusterClient.Close()
	} else {
		err = rc.client.Close()
	}

	if err != nil {
		rc.logger.Printf("关闭Redis客户端失败: %v", err)
	}

	rc.cancel()
	rc.isConnected = false
	rc.subscribers = make(map[string]RedisMessageHandler)
	rc.logger.Println("Redis连接器已断开连接")

	return nil
}

// Set 设置键值
func (rc *RedisConnector) Set(key string, value interface{}, expiration time.Duration) error {
	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	// 序列化值
	data, err := rc.serializeValue(value)
	if err != nil {
		return fmt.Errorf("序列化值失败: %v", err)
	}

	// 执行SET命令
	var setErr error
	if rc.isCluster {
		setErr = rc.clusterClient.Set(rc.ctx, key, data, expiration).Err()
	} else {
		setErr = rc.client.Set(rc.ctx, key, data, expiration).Err()
	}

	if setErr != nil {
		rc.updateError(fmt.Sprintf("SET命令失败: %v", setErr))
		return fmt.Errorf("SET命令失败: %v", setErr)
	}

	rc.updateStats(1, int64(len(data)), 0)
	rc.logger.Printf("已设置键: %s", key)
	return nil
}

// Get 获取键值
func (rc *RedisConnector) Get(key string) (interface{}, error) {
	if !rc.isConnected {
		return nil, fmt.Errorf("Redis客户端未连接")
	}

	// 执行GET命令
	var result string
	var err error
	if rc.isCluster {
		result, err = rc.clusterClient.Get(rc.ctx, key).Result()
	} else {
		result, err = rc.client.Get(rc.ctx, key).Result()
	}

	if err != nil {
		if err == redis.Nil {
			return nil, nil // 键不存在
		}
		rc.updateError(fmt.Sprintf("GET命令失败: %v", err))
		return nil, fmt.Errorf("GET命令失败: %v", err)
	}

	// 反序列化值
	value, deserializeErr := rc.deserializeValue([]byte(result))
	if deserializeErr != nil {
		return result, nil // 如果反序列化失败，返回原始字符串
	}

	rc.updateStats(1, 0, int64(len(result)))
	return value, nil
}

// Delete 删除键
func (rc *RedisConnector) Delete(keys ...string) error {
	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	if len(keys) == 0 {
		return nil
	}

	// 执行DEL命令
	var err error
	if rc.isCluster {
		err = rc.clusterClient.Del(rc.ctx, keys...).Err()
	} else {
		err = rc.client.Del(rc.ctx, keys...).Err()
	}

	if err != nil {
		rc.updateError(fmt.Sprintf("DEL命令失败: %v", err))
		return fmt.Errorf("DEL命令失败: %v", err)
	}

	rc.updateStats(1, 0, 0)
	rc.logger.Printf("已删除 %d 个键", len(keys))
	return nil
}

// Exists 检查键是否存在
func (rc *RedisConnector) Exists(keys ...string) (int64, error) {
	if !rc.isConnected {
		return 0, fmt.Errorf("Redis客户端未连接")
	}

	// 执行EXISTS命令
	var count int64
	var err error
	if rc.isCluster {
		count, err = rc.clusterClient.Exists(rc.ctx, keys...).Result()
	} else {
		count, err = rc.client.Exists(rc.ctx, keys...).Result()
	}

	if err != nil {
		rc.updateError(fmt.Sprintf("EXISTS命令失败: %v", err))
		return 0, fmt.Errorf("EXISTS命令失败: %v", err)
	}

	rc.updateStats(1, 0, 0)
	return count, nil
}

// HSet 设置哈希字段
func (rc *RedisConnector) HSet(key string, values ...interface{}) error {
	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	// 执行HSET命令
	var err error
	if rc.isCluster {
		err = rc.clusterClient.HSet(rc.ctx, key, values...).Err()
	} else {
		err = rc.client.HSet(rc.ctx, key, values...).Err()
	}

	if err != nil {
		rc.updateError(fmt.Sprintf("HSET命令失败: %v", err))
		return fmt.Errorf("HSET命令失败: %v", err)
	}

	rc.updateStats(1, 0, 0)
	rc.logger.Printf("已设置哈希键: %s", key)
	return nil
}

// HGet 获取哈希字段值
func (rc *RedisConnector) HGet(key, field string) (string, error) {
	if !rc.isConnected {
		return "", fmt.Errorf("Redis客户端未连接")
	}

	// 执行HGET命令
	var result string
	var err error
	if rc.isCluster {
		result, err = rc.clusterClient.HGet(rc.ctx, key, field).Result()
	} else {
		result, err = rc.client.HGet(rc.ctx, key, field).Result()
	}

	if err != nil {
		if err == redis.Nil {
			return "", nil // 字段不存在
		}
		rc.updateError(fmt.Sprintf("HGET命令失败: %v", err))
		return "", fmt.Errorf("HGET命令失败: %v", err)
	}

	rc.updateStats(1, 0, int64(len(result)))
	return result, nil
}

// LPush 从列表左侧推入元素
func (rc *RedisConnector) LPush(key string, values ...interface{}) error {
	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	// 执行LPUSH命令
	var err error
	if rc.isCluster {
		err = rc.clusterClient.LPush(rc.ctx, key, values...).Err()
	} else {
		err = rc.client.LPush(rc.ctx, key, values...).Err()
	}

	if err != nil {
		rc.updateError(fmt.Sprintf("LPUSH命令失败: %v", err))
		return fmt.Errorf("LPUSH命令失败: %v", err)
	}

	rc.updateStats(1, 0, 0)
	rc.logger.Printf("已向列表推入 %d 个元素: %s", len(values), key)
	return nil
}

// RPop 从列表右侧弹出元素
func (rc *RedisConnector) RPop(key string) (string, error) {
	if !rc.isConnected {
		return "", fmt.Errorf("Redis客户端未连接")
	}

	// 执行RPOP命令
	var result string
	var err error
	if rc.isCluster {
		result, err = rc.clusterClient.RPop(rc.ctx, key).Result()
	} else {
		result, err = rc.client.RPop(rc.ctx, key).Result()
	}

	if err != nil {
		if err == redis.Nil {
			return "", nil // 列表为空
		}
		rc.updateError(fmt.Sprintf("RPOP命令失败: %v", err))
		return "", fmt.Errorf("RPOP命令失败: %v", err)
	}

	rc.updateStats(1, 0, int64(len(result)))
	return result, nil
}

// Publish 发布消息到频道
func (rc *RedisConnector) Publish(channel string, message interface{}) error {
	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	// 序列化消息
	data, err := rc.serializeValue(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 执行PUBLISH命令
	var publishErr error
	if rc.isCluster {
		publishErr = rc.clusterClient.Publish(rc.ctx, channel, data).Err()
	} else {
		publishErr = rc.client.Publish(rc.ctx, channel, data).Err()
	}

	if publishErr != nil {
		rc.updateError(fmt.Sprintf("PUBLISH命令失败: %v", publishErr))
		return fmt.Errorf("PUBLISH命令失败: %v", publishErr)
	}

	rc.stats.mutex.Lock()
	rc.stats.MessagesSent++
	rc.stats.mutex.Unlock()

	rc.logger.Printf("消息已发布到频道: %s", channel)
	return nil
}

// Subscribe 订阅频道
func (rc *RedisConnector) Subscribe(channel string, handler RedisMessageHandler) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if !rc.isConnected {
		return fmt.Errorf("Redis客户端未连接")
	}

	// 设置处理器
	rc.subscribers[channel] = handler

	// 订阅频道
	return rc.subscribeChannels([]string{channel})
}

// subscribeChannels 订阅频道列表
func (rc *RedisConnector) subscribeChannels(channels []string) error {
	if rc.pubsub != nil {
		// 如果已有订阅，先关闭
		if err := rc.pubsub.Close(); err != nil {
			rc.logger.Printf("关闭旧的订阅失败: %v", err)
		}
	}

	// 创建新的发布订阅
	if rc.isCluster {
		rc.pubsub = rc.clusterClient.Subscribe(rc.ctx, channels...)
	} else {
		rc.pubsub = rc.client.Subscribe(rc.ctx, channels...)
	}

	// 启动消息监听
	go rc.listenMessages()

	rc.logger.Printf("已订阅 %d 个频道", len(channels))
	return nil
}

// listenMessages 监听消息
func (rc *RedisConnector) listenMessages() {
	if rc.pubsub == nil {
		return
	}

	ch := rc.pubsub.Channel()
	for msg := range ch {
		rc.stats.mutex.Lock()
		rc.stats.MessagesReceived++
		rc.stats.mutex.Unlock()

		// 构建消息对象
		redisMsg := &RedisMessage{
			Channel:   msg.Channel,
			Pattern:   msg.Pattern,
			Timestamp: time.Now(),
		}

		// 反序列化消息载荷
		payload, err := rc.deserializeValue([]byte(msg.Payload))
		if err != nil {
			rc.logger.Printf("反序列化消息载荷失败 channel=%s: %v", msg.Channel, err)
			redisMsg.Payload = msg.Payload // 使用原始字符串
		} else {
			redisMsg.Payload = payload
		}

		// 查找并调用处理器
		rc.mutex.RLock()
		handler, exists := rc.subscribers[msg.Channel]
		rc.mutex.RUnlock()

		if exists && handler != nil {
			if err := handler(redisMsg); err != nil {
				rc.logger.Printf("处理消息失败 channel=%s: %v", msg.Channel, err)
			} else {
				rc.logger.Printf("消息处理成功 channel=%s", msg.Channel)
			}
		} else {
			rc.logger.Printf("接收到消息但无处理器 channel=%s", msg.Channel)
		}
	}
}

// Unsubscribe 取消订阅频道
func (rc *RedisConnector) Unsubscribe(channels ...string) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if !rc.isConnected || rc.pubsub == nil {
		return fmt.Errorf("Redis客户端未连接或未订阅")
	}

	// 取消订阅
	if err := rc.pubsub.Unsubscribe(rc.ctx, channels...); err != nil {
		return fmt.Errorf("取消订阅失败: %v", err)
	}

	// 删除处理器
	for _, channel := range channels {
		delete(rc.subscribers, channel)
	}

	rc.logger.Printf("已取消订阅 %d 个频道", len(channels))
	return nil
}

// Pipeline 创建流水线
func (rc *RedisConnector) Pipeline() (redis.Pipeliner, error) {
	if !rc.isConnected {
		return nil, fmt.Errorf("Redis客户端未连接")
	}

	if rc.isCluster {
		return rc.clusterClient.Pipeline(), nil
	}
	return rc.client.Pipeline(), nil
}

// serializeValue 序列化值
func (rc *RedisConnector) serializeValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int, int32, int64:
		return fmt.Sprintf("%v", v), nil
	case float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// deserializeValue 反序列化值
func (rc *RedisConnector) deserializeValue(data []byte) (interface{}, error) {
	// 尝试解析为JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue, nil
	}

	// 如果不是JSON，返回字符串
	return string(data), nil
}

// updateStats 更新统计信息
func (rc *RedisConnector) updateStats(commands int64, bytesWritten, bytesRead int64) {
	rc.stats.mutex.Lock()
	rc.stats.CommandsExecuted += commands
	rc.stats.BytesWritten += bytesWritten
	rc.stats.BytesRead += bytesRead
	rc.stats.mutex.Unlock()
}

// updateError 更新错误信息
func (rc *RedisConnector) updateError(errMsg string) {
	rc.stats.mutex.Lock()
	rc.stats.LastError = errMsg
	rc.stats.mutex.Unlock()
}

// IsConnected 检查连接状态
func (rc *RedisConnector) IsConnected() bool {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	return rc.isConnected
}

// GetSubscribedChannels 获取已订阅的频道列表
func (rc *RedisConnector) GetSubscribedChannels() []string {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	channels := make([]string, 0, len(rc.subscribers))
	for channel := range rc.subscribers {
		channels = append(channels, channel)
	}
	return channels
}

// GetStatistics 获取连接器统计信息
func (rc *RedisConnector) GetStatistics() map[string]interface{} {
	rc.stats.mutex.RLock()
	defer rc.stats.mutex.RUnlock()

	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	stats := map[string]interface{}{
		"connected":           rc.isConnected,
		"is_cluster":          rc.isCluster,
		"address":             rc.config.Address,
		"addresses":           rc.config.Addresses,
		"database":            rc.config.Database,
		"connected_at":        rc.stats.ConnectedAt,
		"commands_executed":   rc.stats.CommandsExecuted,
		"bytes_read":          rc.stats.BytesRead,
		"bytes_written":       rc.stats.BytesWritten,
		"messages_received":   rc.stats.MessagesReceived,
		"messages_sent":       rc.stats.MessagesSent,
		"reconnect_count":     rc.stats.ReconnectCount,
		"subscribed_channels": len(rc.subscribers),
		"configured_channels": rc.config.Channels,
		"last_error":          rc.stats.LastError,
	}

	return stats
}

// GetPoolStats 获取连接池统计信息
func (rc *RedisConnector) GetPoolStats() map[string]interface{} {
	if !rc.isConnected {
		return map[string]interface{}{"error": "not connected"}
	}

	var poolStats *redis.PoolStats
	if rc.isCluster {
		poolStats = rc.clusterClient.PoolStats()
	} else {
		poolStats = rc.client.PoolStats()
	}

	return map[string]interface{}{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"timeouts":    poolStats.Timeouts,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}
}
