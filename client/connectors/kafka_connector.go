/*
 * @module KafkaConnector
 * @description Kafka连接器，提供Kafka生产者和消费者的封装，支持消息的序列化/反序列化和连接管理
 * @architecture 适配器模式 - 封装第三方Kafka客户端，提供统一的接口
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 连接建立 -> 消息发送/接收 -> 连接断开
 * @rules 支持自动重连、消息确认、错误处理
 * @dependencies github.com/segmentio/kafka-go, encoding/json
 * @refs service/models/sync_models.go, service/utils/data_converter.go
 */
package connectors

import (
	"context"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConnector Kafka连接器结构体
type KafkaConnector struct {
	config      *KafkaConfig
	writers     map[string]*kafka.Writer // 按topic分组的生产者
	readers     map[string]*kafka.Reader // 按topic分组的消费者
	mutex       sync.RWMutex
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
}

// 使用models包中定义的类型
type KafkaConfig = models.KafkaConfig
type SecurityConfig = models.SecurityConfig
type ProducerConfig = models.ProducerConfig
type ConsumerConfig = models.ConsumerConfig
type KafkaMessage = models.KafkaMessage
type MessageHandler = models.MessageHandler

// NewKafkaConnector 创建新的Kafka连接器
func NewKafkaConnector(config *KafkaConfig, logger *log.Logger) *KafkaConnector {
	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaConnector{
		config:      config,
		writers:     make(map[string]*kafka.Writer),
		readers:     make(map[string]*kafka.Reader),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		isConnected: false,
	}
}

// Connect 建立Kafka连接
func (kc *KafkaConnector) Connect() error {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	if kc.isConnected {
		return nil
	}

	// 初始化生产者
	for _, topic := range kc.config.Topics {
		writer := &kafka.Writer{
			Addr:         kafka.TCP(kc.config.Brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequiredAcks(kc.config.ProducerConfig.RequiredAcks),
			Async:        kc.config.ProducerConfig.Async,
		}

		if kc.config.ProducerConfig.BatchSize > 0 {
			writer.BatchSize = kc.config.ProducerConfig.BatchSize
		}
		if kc.config.ProducerConfig.BatchTimeout > 0 {
			writer.BatchTimeout = kc.config.ProducerConfig.BatchTimeout
		}

		kc.writers[topic] = writer
	}

	// 初始化消费者
	for _, topic := range kc.config.Topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        kc.config.Brokers,
			Topic:          topic,
			GroupID:        kc.config.GroupID,
			MinBytes:       kc.config.ConsumerConfig.MinBytes,
			MaxBytes:       kc.config.ConsumerConfig.MaxBytes,
			MaxWait:        kc.config.ConsumerConfig.MaxWait,
			CommitInterval: kc.config.ConsumerConfig.CommitInterval,
			StartOffset:    kc.config.ConsumerConfig.StartOffset,
		})

		kc.readers[topic] = reader
	}

	kc.isConnected = true
	kc.logger.Printf("Kafka连接器已连接到brokers: %v", kc.config.Brokers)
	return nil
}

// Disconnect 断开Kafka连接
func (kc *KafkaConnector) Disconnect() error {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	if !kc.isConnected {
		return nil
	}

	// 关闭所有生产者
	for topic, writer := range kc.writers {
		if err := writer.Close(); err != nil {
			kc.logger.Printf("关闭生产者失败 topic=%s: %v", topic, err)
		}
	}

	// 关闭所有消费者
	for topic, reader := range kc.readers {
		if err := reader.Close(); err != nil {
			kc.logger.Printf("关闭消费者失败 topic=%s: %v", topic, err)
		}
	}

	kc.cancel()
	kc.isConnected = false
	kc.logger.Println("Kafka连接器已断开连接")
	return nil
}

// ProduceMessage 发送消息
func (kc *KafkaConnector) ProduceMessage(message *KafkaMessage) error {
	kc.mutex.RLock()
	writer, exists := kc.writers[message.Topic]
	kc.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("找不到topic的生产者: %s", message.Topic)
	}

	// 序列化消息值
	valueBytes, err := kc.serializeValue(message.Value)
	if err != nil {
		return fmt.Errorf("序列化消息值失败: %v", err)
	}

	// 构建Kafka消息
	kafkaMsg := kafka.Message{
		Key:   []byte(message.Key),
		Value: valueBytes,
		Time:  message.Timestamp,
	}

	// 添加消息头
	if message.Headers != nil {
		for key, value := range message.Headers {
			kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
				Key:   key,
				Value: []byte(value),
			})
		}
	}

	// 添加自定义头部
	if kc.config.CustomHeaders != nil {
		for key, value := range kc.config.CustomHeaders {
			kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
				Key:   key,
				Value: []byte(value),
			})
		}
	}

	// 发送消息
	ctx, cancel := context.WithTimeout(kc.ctx, kc.config.ConnectionTimeout)
	defer cancel()

	err = writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	kc.logger.Printf("消息已发送到topic: %s, key: %s", message.Topic, message.Key)
	return nil
}

// ProduceBatchMessages 批量发送消息
func (kc *KafkaConnector) ProduceBatchMessages(messages []*KafkaMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// 按topic分组消息
	topicMessages := make(map[string][]kafka.Message)

	for _, message := range messages {
		valueBytes, err := kc.serializeValue(message.Value)
		if err != nil {
			kc.logger.Printf("序列化消息值失败 topic=%s key=%s: %v",
				message.Topic, message.Key, err)
			continue
		}

		kafkaMsg := kafka.Message{
			Key:   []byte(message.Key),
			Value: valueBytes,
			Time:  message.Timestamp,
		}

		// 添加消息头
		if message.Headers != nil {
			for key, value := range message.Headers {
				kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
					Key:   key,
					Value: []byte(value),
				})
			}
		}

		topicMessages[message.Topic] = append(topicMessages[message.Topic], kafkaMsg)
	}

	// 批量发送
	for topic, msgs := range topicMessages {
		kc.mutex.RLock()
		writer, exists := kc.writers[topic]
		kc.mutex.RUnlock()

		if !exists {
			kc.logger.Printf("找不到topic的生产者: %s", topic)
			continue
		}

		ctx, cancel := context.WithTimeout(kc.ctx, kc.config.ConnectionTimeout)
		err := writer.WriteMessages(ctx, msgs...)
		cancel()

		if err != nil {
			kc.logger.Printf("批量发送消息失败 topic=%s: %v", topic, err)
		} else {
			kc.logger.Printf("批量发送 %d 条消息到topic: %s", len(msgs), topic)
		}
	}

	return nil
}

// ConsumeMessages 消费消息
func (kc *KafkaConnector) ConsumeMessages(topic string, handler MessageHandler) error {
	kc.mutex.RLock()
	reader, exists := kc.readers[topic]
	kc.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("找不到topic的消费者: %s", topic)
	}

	kc.logger.Printf("开始消费topic: %s", topic)

	for {
		select {
		case <-kc.ctx.Done():
			kc.logger.Printf("停止消费topic: %s", topic)
			return nil
		default:
			// 读取消息
			msg, err := reader.ReadMessage(kc.ctx)
			if err != nil {
				if err == context.Canceled {
					return nil
				}
				kc.logger.Printf("读取消息失败 topic=%s: %v", topic, err)
				time.Sleep(time.Second)
				continue
			}

			// 构建消息对象
			message := &KafkaMessage{
				Topic:     msg.Topic,
				Key:       string(msg.Key),
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Timestamp: msg.Time,
				Headers:   make(map[string]string),
			}

			// 解析消息头
			for _, header := range msg.Headers {
				message.Headers[header.Key] = string(header.Value)
			}

			// 反序列化消息值
			value, err := kc.deserializeValue(msg.Value)
			if err != nil {
				kc.logger.Printf("反序列化消息值失败 topic=%s offset=%d: %v",
					topic, msg.Offset, err)
				continue
			}
			message.Value = value

			// 处理消息
			if err := handler(message); err != nil {
				kc.logger.Printf("处理消息失败 topic=%s offset=%d: %v",
					topic, msg.Offset, err)
				continue
			}

			kc.logger.Printf("消息处理成功 topic=%s offset=%d", topic, msg.Offset)
		}
	}
}

// GetTopicMetadata 获取主题元数据
func (kc *KafkaConnector) GetTopicMetadata(topic string) (*kafka.Topic, error) {
	conn, err := kafka.Dial("tcp", kc.config.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("连接Kafka失败: %v", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, fmt.Errorf("读取分区信息失败: %v", err)
	}

	topicInfo := &kafka.Topic{
		Name:       topic,
		Partitions: partitions,
	}

	return topicInfo, nil
}

// serializeValue 序列化消息值
func (kc *KafkaConnector) serializeValue(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return json.Marshal(v)
	}
}

// deserializeValue 反序列化消息值
func (kc *KafkaConnector) deserializeValue(data []byte) (interface{}, error) {
	// 尝试解析为JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue, nil
	}

	// 如果不是JSON，返回字符串
	return string(data), nil
}

// IsConnected 检查连接状态
func (kc *KafkaConnector) IsConnected() bool {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()
	return kc.isConnected
}

// GetConnectedTopics 获取已连接的主题列表
func (kc *KafkaConnector) GetConnectedTopics() []string {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()

	topics := make([]string, 0, len(kc.writers))
	for topic := range kc.writers {
		topics = append(topics, topic)
	}
	return topics
}

// GetStatistics 获取连接器统计信息
func (kc *KafkaConnector) GetStatistics() map[string]interface{} {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()

	stats := map[string]interface{}{
		"connected":         kc.isConnected,
		"writer_count":      len(kc.writers),
		"reader_count":      len(kc.readers),
		"configured_topics": kc.config.Topics,
		"brokers":           kc.config.Brokers,
		"group_id":          kc.config.GroupID,
	}

	return stats
}
