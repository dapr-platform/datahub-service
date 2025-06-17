/*
 * @module MQTTConnector
 * @description MQTT连接器，提供MQTT客户端的封装，支持主题订阅/发布和消息处理
 * @architecture 适配器模式 - 封装第三方MQTT客户端，提供统一的接口
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 连接建立 -> 主题订阅/发布 -> 消息处理 -> 连接断开
 * @rules 支持自动重连、QoS控制、消息确认、遗嘱消息
 * @dependencies github.com/eclipse/paho.mqtt.golang, encoding/json
 * @refs service/models/sync_models.go, service/utils/data_converter.go
 */
package connectors

import (
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTConnector MQTT连接器结构体
type MQTTConnector struct {
	config        *models.MQTTConfig
	client        mqtt.Client
	logger        *log.Logger
	subscribers   map[string]MQTTMessageHandler // 主题订阅处理器映射
	mutex         sync.RWMutex
	isConnected   bool
	reconnectChan chan bool
	stats         *MQTTStats
}

// MQTTMessageHandler MQTT消息处理函数类型
type MQTTMessageHandler func(*models.MQTTMessage) error

// MQTTStats MQTT连接器统计信息
type MQTTStats struct {
	ConnectedAt      time.Time `json:"connected_at"`      // 连接时间
	MessagesSent     int64     `json:"messages_sent"`     // 发送消息数
	MessagesReceived int64     `json:"messages_received"` // 接收消息数
	BytesSent        int64     `json:"bytes_sent"`        // 发送字节数
	BytesReceived    int64     `json:"bytes_received"`    // 接收字节数
	ReconnectCount   int       `json:"reconnect_count"`   // 重连次数
	LastError        string    `json:"last_error"`        // 最后错误信息
	mutex            sync.RWMutex
}

// NewMQTTConnector 创建新的MQTT连接器
func NewMQTTConnector(config *models.MQTTConfig, logger *log.Logger) *MQTTConnector {
	connector := &MQTTConnector{
		config:        config,
		logger:        logger,
		subscribers:   make(map[string]MQTTMessageHandler),
		isConnected:   false,
		reconnectChan: make(chan bool, 1),
		stats:         &MQTTStats{},
	}

	// 配置MQTT客户端选项
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)

	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}

	opts.SetCleanSession(config.CleanSession)
	opts.SetKeepAlive(config.KeepAlive)

	// 设置连接处理器
	opts.SetOnConnectHandler(connector.onConnected)
	opts.SetConnectionLostHandler(connector.onConnectionLost)

	connector.client = mqtt.NewClient(opts)
	return connector
}

// Connect 建立MQTT连接
func (mc *MQTTConnector) Connect() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.isConnected {
		return nil
	}

	mc.logger.Printf("正在连接MQTT broker: %s", mc.config.Broker)

	// 连接到MQTT broker
	if token := mc.client.Connect(); token.Wait() && token.Error() != nil {
		mc.updateError(fmt.Sprintf("MQTT连接失败: %v", token.Error()))
		return fmt.Errorf("MQTT连接失败: %v", token.Error())
	}

	mc.isConnected = true
	mc.stats.ConnectedAt = time.Now()
	mc.logger.Printf("MQTT连接器已连接到broker: %s", mc.config.Broker)

	// 订阅预配置的主题
	for _, topic := range mc.config.Topics {
		if err := mc.subscribeInternal(topic, mc.config.QoS[topic], nil); err != nil {
			mc.logger.Printf("订阅主题失败 topic=%s: %v", topic, err)
		}
	}

	return nil
}

// Disconnect 断开MQTT连接
func (mc *MQTTConnector) Disconnect() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if !mc.isConnected {
		return nil
	}

	mc.logger.Println("正在断开MQTT连接...")

	// 取消所有订阅
	for topic := range mc.subscribers {
		if token := mc.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			mc.logger.Printf("取消订阅失败 topic=%s: %v", topic, token.Error())
		}
	}

	// 断开连接
	mc.client.Disconnect(250) // 等待250ms让消息发送完成

	mc.isConnected = false
	mc.subscribers = make(map[string]MQTTMessageHandler)
	mc.logger.Println("MQTT连接器已断开连接")

	return nil
}

// Subscribe 订阅主题
func (mc *MQTTConnector) Subscribe(topic string, qos byte, handler MQTTMessageHandler) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	return mc.subscribeInternal(topic, qos, handler)
}

// subscribeInternal 内部订阅方法
func (mc *MQTTConnector) subscribeInternal(topic string, qos byte, handler MQTTMessageHandler) error {
	if !mc.isConnected {
		return fmt.Errorf("MQTT客户端未连接")
	}

	// 设置消息处理器
	if handler != nil {
		mc.subscribers[topic] = handler
	}

	// 订阅主题
	token := mc.client.Subscribe(topic, qos, mc.messageHandler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("订阅主题失败 topic=%s: %v", topic, token.Error())
	}

	mc.logger.Printf("已订阅主题: %s (QoS: %d)", topic, qos)
	return nil
}

// Unsubscribe 取消订阅主题
func (mc *MQTTConnector) Unsubscribe(topic string) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if !mc.isConnected {
		return fmt.Errorf("MQTT客户端未连接")
	}

	// 取消订阅
	if token := mc.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return fmt.Errorf("取消订阅失败 topic=%s: %v", topic, token.Error())
	}

	// 删除处理器
	delete(mc.subscribers, topic)

	mc.logger.Printf("已取消订阅主题: %s", topic)
	return nil
}

// Publish 发布消息
func (mc *MQTTConnector) Publish(message *models.MQTTMessage) error {
	mc.mutex.RLock()
	isConnected := mc.isConnected
	mc.mutex.RUnlock()

	if !isConnected {
		return fmt.Errorf("MQTT客户端未连接")
	}

	// 序列化消息载荷
	payload, err := mc.serializePayload(message.Payload)
	if err != nil {
		return fmt.Errorf("序列化消息载荷失败: %v", err)
	}

	// 发布消息
	token := mc.client.Publish(message.Topic, message.QoS, message.Retained, payload)
	if token.Wait() && token.Error() != nil {
		mc.updateError(fmt.Sprintf("发布消息失败: %v", token.Error()))
		return fmt.Errorf("发布消息失败: %v", token.Error())
	}

	// 更新统计信息
	mc.stats.mutex.Lock()
	mc.stats.MessagesSent++
	mc.stats.BytesSent += int64(len(payload))
	mc.stats.mutex.Unlock()

	mc.logger.Printf("消息已发布到主题: %s (QoS: %d, Retained: %t)",
		message.Topic, message.QoS, message.Retained)
	return nil
}

// PublishBatch 批量发布消息
func (mc *MQTTConnector) PublishBatch(messages []*models.MQTTMessage) error {
	if len(messages) == 0 {
		return nil
	}

	mc.mutex.RLock()
	isConnected := mc.isConnected
	mc.mutex.RUnlock()

	if !isConnected {
		return fmt.Errorf("MQTT客户端未连接")
	}

	var publishErrors []error
	sentCount := 0
	totalBytes := int64(0)

	for _, message := range messages {
		// 序列化消息载荷
		payload, err := mc.serializePayload(message.Payload)
		if err != nil {
			publishErrors = append(publishErrors,
				fmt.Errorf("序列化消息载荷失败 topic=%s: %v", message.Topic, err))
			continue
		}

		// 发布消息
		token := mc.client.Publish(message.Topic, message.QoS, message.Retained, payload)
		if token.Wait() && token.Error() != nil {
			publishErrors = append(publishErrors,
				fmt.Errorf("发布消息失败 topic=%s: %v", message.Topic, token.Error()))
			continue
		}

		sentCount++
		totalBytes += int64(len(payload))
	}

	// 更新统计信息
	mc.stats.mutex.Lock()
	mc.stats.MessagesSent += int64(sentCount)
	mc.stats.BytesSent += totalBytes
	mc.stats.mutex.Unlock()

	mc.logger.Printf("批量发布完成: 成功=%d, 失败=%d", sentCount, len(publishErrors))

	if len(publishErrors) > 0 {
		return fmt.Errorf("批量发布部分失败: %d个错误", len(publishErrors))
	}

	return nil
}

// messageHandler 消息处理器
func (mc *MQTTConnector) messageHandler(client mqtt.Client, msg mqtt.Message) {
	mc.stats.mutex.Lock()
	mc.stats.MessagesReceived++
	mc.stats.BytesReceived += int64(len(msg.Payload()))
	mc.stats.mutex.Unlock()

	// 构建消息对象
	message := &models.MQTTMessage{
		Topic:     msg.Topic(),
		QoS:       msg.Qos(),
		Retained:  msg.Retained(),
		MessageID: msg.MessageID(),
		Timestamp: time.Now(),
	}

	// 反序列化消息载荷
	payload, err := mc.deserializePayload(msg.Payload())
	if err != nil {
		mc.logger.Printf("反序列化消息载荷失败 topic=%s: %v", msg.Topic(), err)
		return
	}
	message.Payload = payload.([]byte)

	// 查找并调用处理器
	mc.mutex.RLock()
	handler, exists := mc.subscribers[msg.Topic()]
	mc.mutex.RUnlock()

	if exists && handler != nil {
		if err := handler(message); err != nil {
			mc.logger.Printf("处理消息失败 topic=%s: %v", msg.Topic(), err)
		} else {
			mc.logger.Printf("消息处理成功 topic=%s", msg.Topic())
		}
	} else {
		mc.logger.Printf("接收到消息但无处理器 topic=%s", msg.Topic())
	}
}

// onConnected 连接建立处理器
func (mc *MQTTConnector) onConnected(client mqtt.Client) {
	mc.mutex.Lock()
	mc.isConnected = true
	mc.stats.ConnectedAt = time.Now()
	mc.mutex.Unlock()

	mc.logger.Printf("MQTT连接已建立")

	// 重新订阅主题
	for topic := range mc.subscribers {
		if err := mc.subscribeInternal(topic, mc.config.QoS[topic], nil); err != nil {
			mc.logger.Printf("重新订阅主题失败 topic=%s: %v", topic, err)
		}
	}
}

// onConnectionLost 连接丢失处理器
func (mc *MQTTConnector) onConnectionLost(client mqtt.Client, err error) {
	mc.mutex.Lock()
	mc.isConnected = false
	mc.stats.ReconnectCount++
	mc.mutex.Unlock()

	mc.updateError(fmt.Sprintf("MQTT连接丢失: %v", err))
	mc.logger.Printf("MQTT连接丢失: %v", err)

	// 触发重连通知
	select {
	case mc.reconnectChan <- true:
	default:
	}
}

// serializePayload 序列化消息载荷
func (mc *MQTTConnector) serializePayload(payload interface{}) ([]byte, error) {
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		return []byte(p), nil
	default:
		return json.Marshal(p)
	}
}

// deserializePayload 反序列化消息载荷
func (mc *MQTTConnector) deserializePayload(data []byte) (interface{}, error) {
	// 尝试解析为JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue, nil
	}

	// 如果不是JSON，返回字符串
	return string(data), nil
}

// updateError 更新错误信息
func (mc *MQTTConnector) updateError(errMsg string) {
	mc.stats.mutex.Lock()
	mc.stats.LastError = errMsg
	mc.stats.mutex.Unlock()
}

// IsConnected 检查连接状态
func (mc *MQTTConnector) IsConnected() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.isConnected
}

// GetSubscribedTopics 获取已订阅的主题列表
func (mc *MQTTConnector) GetSubscribedTopics() []string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	topics := make([]string, 0, len(mc.subscribers))
	for topic := range mc.subscribers {
		topics = append(topics, topic)
	}
	return topics
}

// GetStatistics 获取连接器统计信息
func (mc *MQTTConnector) GetStatistics() map[string]interface{} {
	mc.stats.mutex.RLock()
	defer mc.stats.mutex.RUnlock()

	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	stats := map[string]interface{}{
		"connected":         mc.isConnected,
		"broker":            mc.config.Broker,
		"client_id":         mc.config.ClientID,
		"connected_at":      mc.stats.ConnectedAt,
		"messages_sent":     mc.stats.MessagesSent,
		"messages_received": mc.stats.MessagesReceived,
		"bytes_sent":        mc.stats.BytesSent,
		"bytes_received":    mc.stats.BytesReceived,
		"reconnect_count":   mc.stats.ReconnectCount,
		"subscribed_topics": len(mc.subscribers),
		"configured_topics": mc.config.Topics,
		"last_error":        mc.stats.LastError,
	}

	return stats
}

// WaitForReconnect 等待重连信号
func (mc *MQTTConnector) WaitForReconnect() <-chan bool {
	return mc.reconnectChan
}
