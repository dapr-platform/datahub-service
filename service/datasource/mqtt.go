/*
 * @module service/datasource/mqtt
 * @description MQTT数据源实现，作为客户端订阅消息
 * @architecture 发布订阅模式 - 连接MQTT broker并订阅主题
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow MQTT客户端生命周期：连接 -> 订阅主题 -> 接收消息 -> 处理数据 -> 断开连接
 * @rules 支持QoS、自动重连、消息持久化
 * @dependencies github.com/eclipse/paho.mqtt.golang, context, sync, time
 * @refs interface.go, base.go
 */

package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"datahub-service/service/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTDataSource MQTT数据源实现
type MQTTDataSource struct {
	*BaseDataSource
	client         mqtt.Client
	broker         string
	port           int
	clientID       string
	username       string
	password       string
	topics         []string // 订阅的主题列表
	qos            byte     // QoS级别
	timeout        time.Duration
	keepAlive      time.Duration
	cleanSession   bool
	receivedMsgs   []MQTTMessage      // 存储接收到的消息
	mu             sync.RWMutex       // 保护receivedMsgs的并发访问
	msgChannel     chan MQTTMessage   // 消息通道
	subscribers    []chan MQTTMessage // 订阅者列表
	subscribersMu  sync.RWMutex       // 保护subscribers的并发访问
	reconnectDelay time.Duration
	maxReconnects  int
	reconnectCount int

	// 实时数据处理
	realtimeProcessor RealtimeDataProcessor // 实时数据处理器
	enableAutoWrite   bool                  // 是否启用自动写入
}

// MQTTMessage MQTT消息结构
type MQTTMessage struct {
	Topic      string                 `json:"topic"`
	Payload    string                 `json:"payload"`
	QoS        byte                   `json:"qos"`
	Retained   bool                   `json:"retained"`
	MessageID  uint16                 `json:"message_id"`
	ReceivedAt time.Time              `json:"received_at"`
	ParsedData map[string]interface{} `json:"parsed_data,omitempty"`
}

// NewMQTTDataSource 创建MQTT数据源
func NewMQTTDataSource() DataSourceInterface {
	return &MQTTDataSource{
		BaseDataSource: NewBaseDataSource("mqtt", true), // 常驻数据源
		qos:            0,                               // 默认QoS 0
		timeout:        30 * time.Second,
		keepAlive:      60 * time.Second,
		cleanSession:   true,
		receivedMsgs:   make([]MQTTMessage, 0),
		msgChannel:     make(chan MQTTMessage, 1000), // 缓冲通道
		subscribers:    make([]chan MQTTMessage, 0),
		reconnectDelay: 5 * time.Second,
		maxReconnects:  10,
	}
}

// Init 初始化MQTT数据源
func (m *MQTTDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	if err := m.BaseDataSource.Init(ctx, ds); err != nil {
		return err
	}

	// 解析连接配置
	config := ds.ConnectionConfig
	if config == nil {
		return fmt.Errorf("连接配置不能为空")
	}

	// 解析broker地址
	if host, exists := config["host"]; exists {
		if hostStr, ok := host.(string); ok {
			m.broker = hostStr
		} else {
			return fmt.Errorf("broker地址格式错误")
		}
	} else {
		return fmt.Errorf("缺少broker地址配置")
	}

	// 解析端口
	if portVal, exists := config["port"]; exists {
		switch v := portVal.(type) {
		case float64:
			m.port = int(v)
		case int:
			m.port = v
		case string:
			port, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("端口配置格式错误: %v", err)
			}
			m.port = port
		default:
			return fmt.Errorf("端口配置类型错误")
		}
	} else {
		m.port = 1883 // 默认MQTT端口
	}

	// 解析客户端ID
	if clientID, exists := config["client_id"]; exists {
		if idStr, ok := clientID.(string); ok {
			m.clientID = idStr
		} else {
			return fmt.Errorf("客户端ID格式错误")
		}
	} else {
		m.clientID = fmt.Sprintf("datahub-mqtt-%d", time.Now().Unix())
	}

	// 解析用户名和密码（可选）
	if username, exists := config["username"]; exists {
		if userStr, ok := username.(string); ok {
			m.username = userStr
		}
	}

	if password, exists := config["password"]; exists {
		if passStr, ok := password.(string); ok {
			m.password = passStr
		}
	}

	// 解析参数配置
	if ds.ParamsConfig != nil {
		m.parseParamsConfig(ds.ParamsConfig)
	}

	// 默认订阅主题（如果没有在参数中指定）
	if len(m.topics) == 0 {
		m.topics = []string{"datahub/+", "#"} // 默认订阅所有主题
	}

	// 获取全局实时处理器
	m.realtimeProcessor = GetGlobalRealtimeProcessor()
	m.enableAutoWrite = true // 默认启用自动写入

	return nil
}

// parseParamsConfig 解析参数配置
func (m *MQTTDataSource) parseParamsConfig(params map[string]interface{}) {
	// 超时时间
	if timeout, exists := params["timeout"]; exists {
		switch v := timeout.(type) {
		case float64:
			m.timeout = time.Duration(v) * time.Second
		case int:
			m.timeout = time.Duration(v) * time.Second
		default:
			m.timeout = 30 * time.Second
		}
	}

	// Keep Alive间隔
	if keepAlive, exists := params["keep_alive"]; exists {
		switch v := keepAlive.(type) {
		case float64:
			m.keepAlive = time.Duration(v) * time.Second
		case int:
			m.keepAlive = time.Duration(v) * time.Second
		default:
			m.keepAlive = 60 * time.Second
		}
	}

	// QoS级别
	if qos, exists := params["qos"]; exists {
		switch v := qos.(type) {
		case float64:
			if v >= 0 && v <= 2 {
				m.qos = byte(v)
			}
		case int:
			if v >= 0 && v <= 2 {
				m.qos = byte(v)
			}
		}
	}

	// 清理会话
	if cleanSession, exists := params["clean_session"]; exists {
		if clean, ok := cleanSession.(bool); ok {
			m.cleanSession = clean
		}
	}

	// 订阅主题列表
	if topics, exists := params["topics"]; exists {
		switch v := topics.(type) {
		case string:
			m.topics = []string{v}
		case []interface{}:
			m.topics = make([]string, 0, len(v))
			for _, topic := range v {
				if topicStr, ok := topic.(string); ok {
					m.topics = append(m.topics, topicStr)
				}
			}
		case []string:
			m.topics = v
		}
	}

	// 重连配置
	if reconnectDelay, exists := params["reconnect_delay"]; exists {
		switch v := reconnectDelay.(type) {
		case float64:
			m.reconnectDelay = time.Duration(v) * time.Second
		case int:
			m.reconnectDelay = time.Duration(v) * time.Second
		}
	}

	if maxReconnects, exists := params["max_reconnects"]; exists {
		switch v := maxReconnects.(type) {
		case float64:
			m.maxReconnects = int(v)
		case int:
			m.maxReconnects = v
		}
	}

	// 是否启用自动写入
	if enableAutoWrite, exists := params["enable_auto_write"]; exists {
		if enabled, ok := enableAutoWrite.(bool); ok {
			m.enableAutoWrite = enabled
		}
	}
}

// Start 启动MQTT数据源
func (m *MQTTDataSource) Start(ctx context.Context) error {
	if err := m.BaseDataSource.Start(ctx); err != nil {
		return err
	}

	// 创建MQTT客户端选项
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", m.broker, m.port))
	opts.SetClientID(m.clientID)
	opts.SetKeepAlive(m.keepAlive)
	opts.SetCleanSession(m.cleanSession)
	opts.SetConnectTimeout(m.timeout)

	// 设置用户名和密码
	if m.username != "" {
		opts.SetUsername(m.username)
	}
	if m.password != "" {
		opts.SetPassword(m.password)
	}

	// 设置回调函数
	opts.SetDefaultPublishHandler(m.messageHandler)
	opts.SetConnectionLostHandler(m.connectionLostHandler)
	opts.SetOnConnectHandler(m.onConnectHandler)

	// 设置自动重连
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(m.reconnectDelay)

	// 创建客户端
	m.client = mqtt.NewClient(opts)

	// 连接到broker
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("连接MQTT broker失败: %v", token.Error())
	}

	// 订阅主题
	for _, topic := range m.topics {
		if token := m.client.Subscribe(topic, m.qos, m.messageHandler); token.Wait() && token.Error() != nil {
			return fmt.Errorf("订阅主题 %s 失败: %v", topic, token.Error())
		}
		slog.Info("MQTT数据源已订阅主题: %s\n", topic)
	}

	// 启动消息处理协程
	go m.processMessages()

	slog.Info("MQTT数据源已启动，连接到: %s:%d，客户端ID: %s\n", m.broker, m.port, m.clientID)
	return nil
}

// messageHandler MQTT消息处理器
func (m *MQTTDataSource) messageHandler(client mqtt.Client, msg mqtt.Message) {
	message := MQTTMessage{
		Topic:      msg.Topic(),
		Payload:    string(msg.Payload()),
		QoS:        msg.Qos(),
		Retained:   msg.Retained(),
		MessageID:  msg.MessageID(),
		ReceivedAt: time.Now(),
	}

	// 尝试解析JSON数据
	var parsedData map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &parsedData); err == nil {
		message.ParsedData = parsedData
	}

	// 发送到消息通道
	select {
	case m.msgChannel <- message:
		// 消息发送成功
	default:
		// 通道满了，记录警告但不阻塞
		slog.Error("MQTT数据源消息通道已满，丢弃消息: %s\n", msg.Topic())
	}
}

// connectionLostHandler 连接丢失处理器
func (m *MQTTDataSource) connectionLostHandler(client mqtt.Client, err error) {
	slog.Error("MQTT连接丢失: %v，尝试重连...\n", err.Error())
	m.reconnectCount++
}

// onConnectHandler 连接成功处理器
func (m *MQTTDataSource) onConnectHandler(client mqtt.Client) {
	slog.Info("MQTT连接成功，重连次数: %d\n", m.reconnectCount)
	m.reconnectCount = 0
}

// processMessages 处理接收到的消息
func (m *MQTTDataSource) processMessages() {
	for msg := range m.msgChannel {
		// 存储消息
		m.mu.Lock()
		m.receivedMsgs = append(m.receivedMsgs, msg)

		// 限制存储的消息量，只保留最近的5000条
		if len(m.receivedMsgs) > 5000 {
			m.receivedMsgs = m.receivedMsgs[len(m.receivedMsgs)-5000:]
		}
		m.mu.Unlock()

		// 通知所有订阅者
		m.notifySubscribers(msg)

		// 自动写入到关联的数据接口表
		if m.enableAutoWrite && m.realtimeProcessor != nil && msg.ParsedData != nil {
			ctx := context.Background()
			if err := m.realtimeProcessor.ProcessRealtimeData(ctx, m.GetID(), msg.ParsedData); err != nil {
				slog.Error("MQTT实时处理数据失败",
					"datasource_id", m.GetID(),
					"topic", msg.Topic,
					"error", err)
			}
		}
	}
}

// notifySubscribers 通知所有订阅者
func (m *MQTTDataSource) notifySubscribers(msg MQTTMessage) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()

	for _, subscriber := range m.subscribers {
		select {
		case subscriber <- msg:
			// 消息发送成功
		default:
			// 订阅者通道满了，跳过
		}
	}
}

// Execute 执行操作
func (m *MQTTDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	if !m.IsInitialized() {
		response.Error = "数据源未初始化"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("数据源未初始化")
	}

	// 如果启用了脚本，先尝试执行脚本
	if m.GetDataSource().ScriptEnabled && m.GetDataSource().Script != "" {
		scriptResult, err := m.executeScript(ctx, request)
		if err == nil && scriptResult != nil {
			response.Success = true
			response.Data = scriptResult
			response.Duration = time.Since(startTime)
			return response, nil
		}
	}

	switch request.Operation {
	case "query", "read":
		return m.handleQuery(ctx, request, startTime)
	case "publish":
		return m.handlePublish(ctx, request, startTime)
	case "subscribe":
		return m.handleSubscribe(ctx, request, startTime)
	case "unsubscribe":
		return m.handleUnsubscribe(ctx, request, startTime)
	case "status":
		return m.handleStatus(ctx, request, startTime)
	default:
		response.Error = fmt.Sprintf("不支持的操作: %s", request.Operation)
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("不支持的操作: %s", request.Operation)
	}
}

// handleQuery 处理查询操作
func (m *MQTTDataSource) handleQuery(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   true,
		Timestamp: startTime,
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取查询参数
	limit := 100 // 默认限制
	offset := 0  // 默认偏移
	topic := ""  // 主题过滤

	if request.Params != nil {
		if l, exists := request.Params["limit"]; exists {
			if limitInt, ok := l.(int); ok {
				limit = limitInt
			}
		}
		if o, exists := request.Params["offset"]; exists {
			if offsetInt, ok := o.(int); ok {
				offset = offsetInt
			}
		}
		if t, exists := request.Params["topic"]; exists {
			if topicStr, ok := t.(string); ok {
				topic = topicStr
			}
		}
	}

	// 过滤消息
	var filteredMsgs []MQTTMessage
	if topic == "" {
		filteredMsgs = m.receivedMsgs
	} else {
		for _, msg := range m.receivedMsgs {
			if msg.Topic == topic {
				filteredMsgs = append(filteredMsgs, msg)
			}
		}
	}

	// 计算数据范围
	total := len(filteredMsgs)
	start := offset
	end := offset + limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var data []MQTTMessage
	if start < end {
		data = filteredMsgs[start:end]
	} else {
		data = make([]MQTTMessage, 0)
	}

	response.Data = data
	response.RowCount = int64(len(data))
	response.Metadata = map[string]interface{}{
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"topic_filter": topic,
		"broker":       fmt.Sprintf("%s:%d", m.broker, m.port),
		"client_id":    m.clientID,
		"topics":       m.topics,
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handlePublish 处理发布操作
func (m *MQTTDataSource) handlePublish(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	if !m.IsStarted() || m.client == nil || !m.client.IsConnected() {
		response.Error = "MQTT客户端未连接"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("MQTT客户端未连接")
	}

	// 获取发布参数
	var topic string
	var payload []byte
	var qos byte = m.qos
	var retained bool = false

	if request.Params != nil {
		if t, exists := request.Params["topic"]; exists {
			if topicStr, ok := t.(string); ok {
				topic = topicStr
			} else {
				response.Error = "主题参数格式错误"
				response.Duration = time.Since(startTime)
				return response, fmt.Errorf("主题参数格式错误")
			}
		} else {
			response.Error = "缺少主题参数"
			response.Duration = time.Since(startTime)
			return response, fmt.Errorf("缺少主题参数")
		}

		if q, exists := request.Params["qos"]; exists {
			if qosInt, ok := q.(int); ok && qosInt >= 0 && qosInt <= 2 {
				qos = byte(qosInt)
			}
		}

		if r, exists := request.Params["retained"]; exists {
			if retainedBool, ok := r.(bool); ok {
				retained = retainedBool
			}
		}
	}

	// 准备payload
	if request.Data != nil {
		switch v := request.Data.(type) {
		case string:
			payload = []byte(v)
		case []byte:
			payload = v
		default:
			// 尝试JSON序列化
			jsonData, err := json.Marshal(v)
			if err != nil {
				response.Error = fmt.Sprintf("数据序列化失败: %v", err)
				response.Duration = time.Since(startTime)
				return response, fmt.Errorf("数据序列化失败: %v", err)
			}
			payload = jsonData
		}
	} else {
		payload = []byte("")
	}

	// 发布消息
	token := m.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		response.Error = fmt.Sprintf("发布消息失败: %v", token.Error())
		response.Duration = time.Since(startTime)
		return response, token.Error()
	}

	response.Success = true
	response.Message = "消息发布成功"
	response.Metadata = map[string]interface{}{
		"topic":    topic,
		"qos":      qos,
		"retained": retained,
		"size":     len(payload),
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handleSubscribe 处理订阅操作
func (m *MQTTDataSource) handleSubscribe(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	if !m.IsStarted() || m.client == nil || !m.client.IsConnected() {
		response.Error = "MQTT客户端未连接"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("MQTT客户端未连接")
	}

	// 获取订阅参数
	var topic string
	var qos byte = m.qos

	if request.Params != nil {
		if t, exists := request.Params["topic"]; exists {
			if topicStr, ok := t.(string); ok {
				topic = topicStr
			} else {
				response.Error = "主题参数格式错误"
				response.Duration = time.Since(startTime)
				return response, fmt.Errorf("主题参数格式错误")
			}
		} else {
			response.Error = "缺少主题参数"
			response.Duration = time.Since(startTime)
			return response, fmt.Errorf("缺少主题参数")
		}

		if q, exists := request.Params["qos"]; exists {
			if qosInt, ok := q.(int); ok && qosInt >= 0 && qosInt <= 2 {
				qos = byte(qosInt)
			}
		}
	}

	// 订阅主题
	token := m.client.Subscribe(topic, qos, m.messageHandler)
	if token.Wait() && token.Error() != nil {
		response.Error = fmt.Sprintf("订阅主题失败: %v", token.Error())
		response.Duration = time.Since(startTime)
		return response, token.Error()
	}

	// 添加到主题列表
	m.topics = append(m.topics, topic)

	response.Success = true
	response.Message = "主题订阅成功"
	response.Metadata = map[string]interface{}{
		"topic":        topic,
		"qos":          qos,
		"total_topics": len(m.topics),
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handleUnsubscribe 处理取消订阅操作
func (m *MQTTDataSource) handleUnsubscribe(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	if !m.IsStarted() || m.client == nil || !m.client.IsConnected() {
		response.Error = "MQTT客户端未连接"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("MQTT客户端未连接")
	}

	// 获取取消订阅参数
	var topic string

	if request.Params != nil {
		if t, exists := request.Params["topic"]; exists {
			if topicStr, ok := t.(string); ok {
				topic = topicStr
			} else {
				response.Error = "主题参数格式错误"
				response.Duration = time.Since(startTime)
				return response, fmt.Errorf("主题参数格式错误")
			}
		} else {
			response.Error = "缺少主题参数"
			response.Duration = time.Since(startTime)
			return response, fmt.Errorf("缺少主题参数")
		}
	}

	// 取消订阅主题
	token := m.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		response.Error = fmt.Sprintf("取消订阅主题失败: %v", token.Error())
		response.Duration = time.Since(startTime)
		return response, token.Error()
	}

	// 从主题列表中移除
	newTopics := make([]string, 0)
	for _, t := range m.topics {
		if t != topic {
			newTopics = append(newTopics, t)
		}
	}
	m.topics = newTopics

	response.Success = true
	response.Message = "取消订阅成功"
	response.Metadata = map[string]interface{}{
		"topic":        topic,
		"total_topics": len(m.topics),
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handleStatus 处理状态查询
func (m *MQTTDataSource) handleStatus(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   true,
		Timestamp: startTime,
	}

	m.mu.RLock()
	msgCount := len(m.receivedMsgs)
	m.mu.RUnlock()

	m.subscribersMu.RLock()
	subscriberCount := len(m.subscribers)
	m.subscribersMu.RUnlock()

	var connected bool
	if m.client != nil {
		connected = m.client.IsConnected()
	}

	response.Data = map[string]interface{}{
		"broker":           fmt.Sprintf("%s:%d", m.broker, m.port),
		"client_id":        m.clientID,
		"connected":        connected,
		"topics":           m.topics,
		"qos":              m.qos,
		"clean_session":    m.cleanSession,
		"message_count":    msgCount,
		"subscriber_count": subscriberCount,
		"reconnect_count":  m.reconnectCount,
		"max_reconnects":   m.maxReconnects,
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// Stop 停止MQTT数据源
func (m *MQTTDataSource) Stop(ctx context.Context) error {
	if err := m.BaseDataSource.Stop(ctx); err != nil {
		return err
	}

	// 断开MQTT连接
	if m.client != nil && m.client.IsConnected() {
		// 取消所有订阅
		for _, topic := range m.topics {
			m.client.Unsubscribe(topic)
		}

		// 断开连接
		m.client.Disconnect(250)
	}

	// 关闭消息通道
	close(m.msgChannel)

	// 关闭所有订阅者通道
	m.subscribersMu.Lock()
	for _, subscriber := range m.subscribers {
		close(subscriber)
	}
	m.subscribers = make([]chan MQTTMessage, 0)
	m.subscribersMu.Unlock()

	slog.Info("MQTT数据源已停止\n")
	return nil
}

// HealthCheck 健康检查
func (m *MQTTDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status, err := m.BaseDataSource.HealthCheck(ctx)
	if err != nil {
		return status, err
	}

	// 检查MQTT连接状态
	if m.client != nil && m.client.IsConnected() {
		status.Status = "online"
		status.Message = "MQTT客户端已连接"

		m.mu.RLock()
		msgCount := len(m.receivedMsgs)
		m.mu.RUnlock()

		status.Details["broker"] = fmt.Sprintf("%s:%d", m.broker, m.port)
		status.Details["client_id"] = m.clientID
		status.Details["topics"] = m.topics
		status.Details["message_count"] = msgCount
		status.Details["reconnect_count"] = m.reconnectCount
	} else {
		status.Status = "offline"
		status.Message = "MQTT客户端未连接"
	}

	return status, nil
}

// GetReceivedMessages 获取接收到的消息（用于测试）
func (m *MQTTDataSource) GetReceivedMessages() []MQTTMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回消息副本
	msgs := make([]MQTTMessage, len(m.receivedMsgs))
	copy(msgs, m.receivedMsgs)
	return msgs
}

// ClearReceivedMessages 清空接收到的消息（用于测试）
func (m *MQTTDataSource) ClearReceivedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMsgs = make([]MQTTMessage, 0)
}

// IsConnected 检查是否已连接（用于测试）
func (m *MQTTDataSource) IsConnected() bool {
	if m.client == nil {
		return false
	}
	return m.client.IsConnected()
}

// executeScript 执行脚本
func (m *MQTTDataSource) executeScript(ctx context.Context, request *ExecuteRequest) (interface{}, error) {
	if m.scriptExecutor == nil {
		return nil, fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := make(map[string]interface{})
	params["request"] = request
	params["dataSource"] = m.GetDataSource()
	params["connectionConfig"] = m.GetDataSource().ConnectionConfig
	params["paramsConfig"] = m.GetDataSource().ParamsConfig
	params["operation"] = request.Operation
	params["receivedMessages"] = m.GetReceivedMessages()
	params["mqttClient"] = m.client
	params["topics"] = m.topics

	return m.scriptExecutor.Execute(ctx, m.GetDataSource().Script, params)
}
