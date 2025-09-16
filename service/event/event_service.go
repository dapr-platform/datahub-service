/*
 * @module service/event_service
 * @description 事件管理服务，提供SSE事件推送和数据库变更监听功能
 * @architecture 事件驱动架构 - 业务服务层
 * @documentReference ai_docs/patch_db_event.md
 * @stateFlow 事件监听 -> 事件处理 -> 事件分发 -> 客户端推送
 * @rules 确保事件的实时性和可靠性
 * @dependencies datahub-service/service/models, gorm.io/gorm, github.com/lib/pq
 * @refs ai_docs/requirements.md
 */

package event

import (
	"context"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// EventService 事件管理服务
type EventService struct {
	db                *gorm.DB
	connections       map[string]map[string]*SSEClient // userName -> connectionID -> client
	mu                sync.RWMutex
	dbEventProcessors map[string]models.DBEventProcessor
	dbListener        *pq.Listener
	ctx               context.Context
	cancel            context.CancelFunc
	httpClient        *http.Client
	functionCreated   bool // 标记通知函数是否已创建
}

// SSEClient SSE客户端连接
type SSEClient struct {
	ID       string
	UserName string
	Channel  chan *models.SSEEvent
	Done     chan bool
	ClientIP string
}

// TriggerInfo PostgreSQL触发器信息
type TriggerInfo struct {
	ID          int      `json:"id"`
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	TableID     int      `json:"table_id"`
	TableName   string   `json:"table"`
	Function    string   `json:"function"`
	Condition   string   `json:"condition"`
	Orientation string   `json:"orientation"`
	Activation  string   `json:"activation"`
	Timing      string   `json:"timing"`
	Events      []string `json:"events"`
	EnabledMode string   `json:"enabled_mode"`
}

// NewEventService 创建事件服务实例
func NewEventService(db *gorm.DB) *EventService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &EventService{
		db:                db,
		connections:       make(map[string]map[string]*SSEClient),
		dbEventProcessors: make(map[string]models.DBEventProcessor),
		ctx:               ctx,
		cancel:            cancel,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
		functionCreated:   false,
	}

	// 启动数据库监听器
	go service.startDBListener()

	// 启动事件处理器
	go service.startEventProcessor()

	// 检查数据库触发器
	go service.checkDatabaseTriggers()

	return service
}

// === SSE连接管理 ===

// AddSSEConnection 添加SSE连接
func (s *EventService) AddSSEConnection(userName, connectionID, clientIP string) *SSEClient {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connections[userName] == nil {
		s.connections[userName] = make(map[string]*SSEClient)
	}

	client := &SSEClient{
		ID:       connectionID,
		UserName: userName,
		Channel:  make(chan *models.SSEEvent, 100), // 缓冲100个事件
		Done:     make(chan bool),
		ClientIP: clientIP,
	}

	s.connections[userName][connectionID] = client

	// 记录连接到数据库
	connection := &models.SSEConnection{
		UserName:     userName,
		ConnectionID: connectionID,
		ClientIP:     clientIP,
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		IsActive:     true,
	}
	s.db.Create(connection)

	log.Printf("SSE连接已建立: 用户=%s, 连接ID=%s, IP=%s", userName, connectionID, clientIP)
	return client
}

// RemoveSSEConnection 移除SSE连接
func (s *EventService) RemoveSSEConnection(userName, connectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if userConnections, exists := s.connections[userName]; exists {
		if client, exists := userConnections[connectionID]; exists {
			close(client.Done)
			delete(userConnections, connectionID)

			if len(userConnections) == 0 {
				delete(s.connections, userName)
			}

			// 更新数据库连接状态
			s.db.Model(&models.SSEConnection{}).
				Where("connection_id = ?", connectionID).
				Update("is_active", false)

			log.Printf("SSE连接已断开: 用户=%s, 连接ID=%s", userName, connectionID)
		}
	}
}

// SendEventToUser 向指定用户发送事件
func (s *EventService) SendEventToUser(userName string, event *models.SSEEvent) error {
	s.mu.RLock()
	userConnections, exists := s.connections[userName]
	s.mu.RUnlock()

	if !exists {
		log.Printf("用户 %s 没有活跃的SSE连接", userName)
		return fmt.Errorf("用户 %s 没有活跃的SSE连接", userName)
	}

	// 保存事件到数据库
	if err := s.db.Create(event).Error; err != nil {
		log.Printf("保存SSE事件失败: %v", err)
		return err
	}

	// 向所有连接发送事件
	for _, client := range userConnections {
		select {
		case client.Channel <- event:
			log.Printf("事件已发送到用户 %s 的连接 %s", userName, client.ID)
		default:
			log.Printf("用户 %s 的连接 %s 事件队列已满，跳过发送", userName, client.ID)
		}
	}

	return nil
}

// BroadcastEvent 广播事件给所有用户
func (s *EventService) BroadcastEvent(event *models.SSEEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 保存事件到数据库
	if err := s.db.Create(event).Error; err != nil {
		log.Printf("保存广播事件失败: %v", err)
		return err
	}

	for userName, userConnections := range s.connections {
		for _, client := range userConnections {
			eventCopy := *event
			eventCopy.UserName = userName

			select {
			case client.Channel <- &eventCopy:
				log.Printf("广播事件已发送到用户 %s 的连接 %s", userName, client.ID)
			default:
				log.Printf("用户 %s 的连接 %s 事件队列已满，跳过广播", userName, client.ID)
			}
		}
	}

	return nil
}

// === 数据库监听管理 ===

// CreateDBEventListener 创建数据库事件监听器
func (s *EventService) RegisterDBEventProcessor(processor models.DBEventProcessor) error {

	s.mu.Lock()
	s.dbEventProcessors[processor.TableName()] = processor
	s.mu.Unlock()

	log.Printf("数据库事件监听器已创建: %s", processor.TableName())
	s.checkTableTrigger(processor.TableName())
	return nil
}

// === 数据库监听实现 ===

// startDBListener 启动数据库监听器
func (s *EventService) startDBListener() {
	// 获取数据库连接信息
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// 从环境变量构建连接字符串
		host := getEnvWithDefault("DB_HOST", "localhost")
		port := getEnvWithDefault("DB_PORT", "5432")
		user := getEnvWithDefault("DB_USER", "postgres")
		password := getEnvWithDefault("DB_PASSWORD", "things2024")
		dbname := getEnvWithDefault("DB_NAME", "postgres")
		sslmode := getEnvWithDefault("DB_SSLMODE", "disable")

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	// 创建PostgreSQL监听器
	s.dbListener = pq.NewListener(connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("PostgreSQL监听器事件: %v, 错误: %v", ev, err)
		}
		log.Printf("PostgreSQL监听器事件: %v, 错误: %v", ev, err)
	})

	// 监听数据库通知
	if err := s.dbListener.Listen("datahub_changes"); err != nil {
		log.Printf("监听数据库通知失败: %v", err)
		return
	}

	log.Println("数据库监听器已启动")

	// 处理数据库通知
	for {
		select {
		case notification := <-s.dbListener.Notify:
			if notification != nil {
				s.handleDBNotification(notification)
			}
		case <-s.ctx.Done():
			log.Println("数据库监听器已停止")
			return
		}
	}
}

// handleDBNotification 处理数据库通知
func (s *EventService) handleDBNotification(notification *pq.Notification) {
	var changeData map[string]interface{}
	if err := json.Unmarshal([]byte(notification.Extra), &changeData); err != nil {
		log.Printf("解析数据库通知失败: %v", err)
		return
	}

	tableName, _ := changeData["table"].(string)
	eventType, _ := changeData["type"].(string)
	recordID, _ := changeData["record_id"].(string)

	log.Printf("收到数据库变更通知: 表=%s, 类型=%s, 记录ID=%s", tableName, eventType, recordID)

	processor, ok := s.dbEventProcessors[tableName]
	if !ok {
		log.Printf("未找到匹配的事件处理器: %s", tableName)
		return
	}

	// 处理匹配的监听器
	processor.ProcessDBChangeEvent(changeData)
}

// containsEventType 检查事件类型是否在监听列表中
func (s *EventService) containsEventType(eventTypes []string, eventType string) bool {
	for _, et := range eventTypes {
		if et == eventType {
			return true
		}
	}
	return false
}

// startEventProcessor 启动事件处理器
func (s *EventService) startEventProcessor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupInactiveConnections()
		case <-s.ctx.Done():
			log.Println("事件处理器已停止")
			return
		}
	}
}

// cleanupInactiveConnections 清理不活跃的连接
func (s *EventService) cleanupInactiveConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userName, userConnections := range s.connections {
		for connectionID, client := range userConnections {
			select {
			case <-client.Done:
				delete(userConnections, connectionID)
				log.Printf("清理已断开的连接: 用户=%s, 连接ID=%s", userName, connectionID)
			default:
				// 连接仍然活跃
			}
		}

		if len(userConnections) == 0 {
			delete(s.connections, userName)
		}
	}
}

// Stop 停止事件服务
func (s *EventService) Stop() {
	s.cancel()

	if s.dbListener != nil {
		s.dbListener.Close()
	}

	// 关闭所有SSE连接
	s.mu.Lock()
	for _, userConnections := range s.connections {
		for _, client := range userConnections {
			close(client.Done)
		}
	}
	s.connections = make(map[string]map[string]*SSEClient)
	s.mu.Unlock()

	log.Println("事件服务已停止")
}

// checkDatabaseTriggers 检查数据库触发器
func (s *EventService) checkDatabaseTriggers() {
	requiredTables := make([]string, 0)
	for _, processor := range s.dbEventProcessors {
		requiredTables = append(requiredTables, processor.TableName())
	}

	for _, tableName := range requiredTables {
		if err := s.checkTableTrigger(tableName); err != nil {
			log.Printf("检查表 %s 的触发器失败: %v", tableName, err)
		}
	}
}

// checkTableTrigger 检查指定表的触发器
func (s *EventService) checkTableTrigger(tableName string) error {
	triggers, err := s.getTriggers()
	if err != nil {
		return fmt.Errorf("获取触发器列表失败: %v", err)
	}
	log.Println("triggers", triggers)

	// 检查是否存在对应的触发器
	triggerName := tableName + "_notify"
	found := false
	activation := "BEFORE"
	for _, trigger := range triggers {
		if trigger.Name == triggerName && trigger.TableName == tableName && trigger.Activation == activation {
			found = true
			log.Printf("表 %s 的触发器 %s 已存在", tableName, triggerName)

			break
		}
	}

	if !found {
		log.Printf("警告: 表 %s 缺少触发器 %s，正在创建...", tableName, triggerName)
		if err := s.createTableTrigger(tableName, activation); err != nil {
			return fmt.Errorf("创建表 %s 的触发器失败: %v", tableName, err)
		}
		log.Printf("成功创建表 %s 的触发器 %s", tableName, triggerName)
	}

	return nil
}

// createTableTrigger 为指定表创建触发器
func (s *EventService) createTableTrigger(tableName string, activation string) error {
	triggerName := tableName + "_notify"

	// 首先确保通知函数存在
	if err := s.createNotifyFunction(); err != nil {
		return fmt.Errorf("创建通知函数失败: %v", err)
	}

	// 构建创建触发器的SQL语句
	createTriggerSQL := fmt.Sprintf(`
		CREATE OR REPLACE TRIGGER %s
		%s INSERT OR UPDATE OR DELETE ON %s
		FOR EACH ROW
		EXECUTE FUNCTION notify_datahub_changes();
	`, triggerName, activation, tableName)

	// 执行SQL创建触发器
	if err := s.db.Exec(createTriggerSQL).Error; err != nil {
		return fmt.Errorf("执行创建触发器SQL失败: %v", err)
	}

	return nil
}

// createNotifyFunction 创建数据库通知函数
func (s *EventService) createNotifyFunction() error {
	if s.functionCreated {
		return nil
	}

	createFunctionSQL := `
CREATE OR REPLACE FUNCTION notify_datahub_changes()
RETURNS TRIGGER AS $$
DECLARE
    record_id TEXT;
    table_name TEXT;
    event_type TEXT;
    payload JSON;
BEGIN
    -- 获取表名
    table_name := TG_TABLE_NAME;
    
    -- 获取事件类型
    event_type := TG_OP;
    
    -- 根据操作类型获取记录ID和数据
    IF TG_OP = 'DELETE' THEN
        record_id := OLD.id;
        payload := json_build_object(
            'table', table_name,
            'type', event_type,
            'record_id', record_id,
            'old_data', row_to_json(OLD),
            'timestamp', extract(epoch from now())
        );
    ELSIF TG_OP = 'INSERT' THEN
        record_id := NEW.id;
        payload := json_build_object(
            'table', table_name,
            'type', event_type,
            'record_id', record_id,
            'new_data', row_to_json(NEW),
            'timestamp', extract(epoch from now())
        );
    ELSIF TG_OP = 'UPDATE' THEN
        record_id := NEW.id;
        payload := json_build_object(
            'table', table_name,
            'type', event_type,
            'record_id', record_id,
            'old_data', row_to_json(OLD),
            'new_data', row_to_json(NEW),
            'timestamp', extract(epoch from now())
        );
    END IF;
    
    -- 发送通知
    PERFORM pg_notify('datahub_changes', payload::text);
    
	RAISE NOTICE 'payload: %', payload;
    -- 返回适当的记录
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;`

	if err := s.db.Exec(createFunctionSQL).Error; err != nil {
		return fmt.Errorf("执行创建函数SQL失败: %v", err)
	}

	log.Println("数据库通知函数 notify_datahub_changes() 已创建")
	s.functionCreated = true
	return nil
}

// getTriggers 获取数据库触发器列表
func (s *EventService) getTriggers() ([]TriggerInfo, error) {
	// 检查是否在 Dapr 环境中
	if s.isDaprEnvironment() {
		return s.getTriggersViaDapr()
	}

	// 本地调试环境，使用 HTTP 客户端
	return s.getTriggersViaHTTP()
}

// isDaprEnvironment 检查是否在 Dapr 环境中
func (s *EventService) isDaprEnvironment() bool {
	// 检查 Dapr 相关环境变量
	daprHTTPPort := os.Getenv("DAPR_HTTP_PORT")
	daprGRPCPort := os.Getenv("DAPR_GRPC_PORT")

	return daprHTTPPort != "" || daprGRPCPort != ""
}

// getTriggersViaDapr 通过 Dapr 客户端获取触发器
func (s *EventService) getTriggersViaDapr() ([]TriggerInfo, error) {
	// 获取 Dapr HTTP 端口，默认为 3500
	daprPort := os.Getenv("DAPR_HTTP_PORT")
	if daprPort == "" {
		daprPort = "3500"
	}

	// 构建 Dapr 服务调用 URL
	url := fmt.Sprintf("http://localhost:%s/v1.0/invoke/postgre-meta/method/triggers/", daprPort)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加必要的头部
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pg", "default") // PostgreSQL 连接标识

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	var triggers []TriggerInfo
	if err := json.NewDecoder(resp.Body).Decode(&triggers); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	log.Printf("通过 Dapr 获取到 %d 个触发器", len(triggers))
	return triggers, nil
}

// getTriggersViaHTTP 通过 HTTP 客户端获取触发器
func (s *EventService) getTriggersViaHTTP() ([]TriggerInfo, error) {
	// 本地调试环境的 postgres-meta 服务地址
	url := "http://localhost:3001/triggers/"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加必要的头部
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pg", "default") // PostgreSQL 连接标识

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	var triggers []TriggerInfo
	if err := json.NewDecoder(resp.Body).Decode(&triggers); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	log.Printf("通过 HTTP 获取到 %d 个触发器", len(triggers))
	return triggers, nil
}

// GetSSEConnectionList 获取SSE连接列表
func (s *EventService) GetSSEConnectionList(page, pageSize int, userName, clientIP string, isActive *bool) ([]models.SSEConnection, int64, error) {
	var connections []models.SSEConnection
	var total int64

	query := s.db.Model(&models.SSEConnection{})

	// 添加过滤条件
	if userName != "" {
		query = query.Where("user_name = ?", userName)
	}
	if clientIP != "" {
		query = query.Where("client_ip = ?", clientIP)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("connected_at DESC").
		Offset(offset).Limit(pageSize).Find(&connections).Error

	return connections, total, err
}

// GetEventHistoryList 获取事件历史列表
func (s *EventService) GetEventHistoryList(page, pageSize int, userName, eventType string, sent, read *bool) ([]models.SSEEvent, int64, error) {
	var events []models.SSEEvent
	var total int64

	query := s.db.Model(&models.SSEEvent{})

	// 添加过滤条件
	if userName != "" {
		query = query.Where("user_name = ?", userName)
	}
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if sent != nil {
		query = query.Where("sent = ?", *sent)
	}
	if read != nil {
		query = query.Where("read = ?", *read)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&events).Error

	return events, total, err
}
