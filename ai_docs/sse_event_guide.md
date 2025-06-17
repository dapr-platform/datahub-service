# SSE 事件功能使用指南

## 概述

数据底座服务现已支持 Server-Sent Events (SSE)事件推送和 PostgreSQL 数据库变更监听功能。前端页面可以通过 SSE 连接实时接收系统事件、数据变更通知等。

## 功能特性

### 1. SSE 事件推送

- **实时连接**：前端通过 `/sse/{user_name}` 建立长连接
- **事件类型**：支持系统通知、数据变更、用户消息、告警等多种事件类型
- **用户定向**：可向指定用户发送事件或广播给所有用户
- **连接管理**：自动管理连接状态，支持断线重连

### 2. 数据库事件监听

- **触发器监听**：通过 PostgreSQL 触发器监听数据表变化
- **事件过滤**：支持按表名、事件类型、条件过滤
- **实时推送**：数据库变更自动转换为 SSE 事件推送给前端
- **灵活配置**：可动态创建、更新、删除监听器配置

## API 接口

### SSE 连接

#### 建立 SSE 连接

```
GET /sse/{user_name}
```

**参数：**

- `user_name`: 用户名

**响应：**

- Content-Type: `text/event-stream`
- 持续的事件流数据

**示例：**

```javascript
const eventSource = new EventSource("/sse/admin");

eventSource.onmessage = function (event) {
  const data = JSON.parse(event.data);
  console.log("收到事件:", data);
};

eventSource.onerror = function (error) {
  console.error("SSE连接错误:", error);
};
```

### 事件发送

#### 发送事件给指定用户

```
POST /events/send
```

**请求体：**

```json
{
  "user_name": "admin",
  "event_type": "system_notification",
  "data": {
    "title": "系统通知",
    "message": "这是一个测试消息",
    "priority": "high"
  }
}
```

#### 广播事件给所有用户

```
POST /events/broadcast
```

**请求体：**

```json
{
  "event_type": "system_announcement",
  "data": {
    "title": "系统公告",
    "message": "系统将在30分钟后进行维护",
    "type": "maintenance"
  }
}
```

### 数据库事件监听器管理

#### 创建监听器

```
POST /events/db-listeners
```

**请求体：**

```json
{
  "name": "基础库变更监听",
  "table_name": "basic_libraries",
  "event_types": ["INSERT", "UPDATE", "DELETE"],
  "condition": { "status": "active" },
  "target_users": ["admin", "operator"]
}
```

#### 获取监听器列表

```
GET /events/db-listeners?page=1&page_size=10
```

#### 更新监听器

```
PUT /events/db-listeners/{id}
```

#### 删除监听器

```
DELETE /events/db-listeners/{id}
```

## 事件类型

### 预定义事件类型

- `data_change`: 数据变更事件
- `system_notification`: 系统通知
- `user_message`: 用户消息
- `alert`: 告警事件
- `status_update`: 状态更新

### 事件数据格式

```json
{
  "id": "event-uuid",
  "type": "system_notification",
  "data": {
    "title": "通知标题",
    "message": "通知内容",
    "priority": "high",
    "timestamp": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 数据库触发器

### 自动安装触发器

系统启动时会自动为以下表创建数据变更监听触发器：

- `basic_libraries` - 基础库
- `data_interfaces` - 数据接口
- `thematic_libraries` - 主题库
- `thematic_interfaces` - 主题接口
- `quality_rules` - 质量规则
- `metadata` - 元数据
- `data_masking_rules` - 脱敏规则
- `api_applications` - API 应用
- `data_subscriptions` - 数据订阅
- `data_sync_tasks` - 同步任务
- `system_logs` - 系统日志

### 手动安装触发器

如果需要手动安装或重新安装触发器：

```bash
# 连接到PostgreSQL数据库
psql -h localhost -U postgres -d postgres

# 执行触发器安装脚本
\i scripts/setup_db_triggers.sql
```

### 测试触发器

```sql
-- 测试通知功能
SELECT test_datahub_notification('Hello from PostgreSQL!');

-- 手动发送通知
NOTIFY datahub_changes, '{"table":"test","type":"MANUAL","record_id":"manual-test","message":"Manual test notification","timestamp":1234567890}';
```

## 测试工具

### 1. 命令行测试脚本

```bash
# 运行完整的SSE功能测试
./scripts/test_sse_events.sh

# 给脚本添加执行权限（如果需要）
chmod +x scripts/test_sse_events.sh
```

### 2. Web 测试页面

打开 `scripts/sse_test.html` 在浏览器中进行可视化测试：

**功能包括：**

- SSE 连接建立和断开
- 发送事件给指定用户
- 广播事件给所有用户
- 测试数据库触发器
- 实时事件日志显示
- 连接统计信息

### 3. 手动测试命令

```bash
# 测试SSE连接
curl -N http://localhost:8080/sse/admin

# 发送测试事件
curl -X POST http://localhost:8080/events/send \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "admin",
    "event_type": "system_notification",
    "data": {
      "title": "测试通知",
      "message": "这是一个测试消息"
    }
  }'

# 广播事件
curl -X POST http://localhost:8080/events/broadcast \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "system_announcement",
    "data": {
      "title": "系统公告",
      "message": "系统维护通知"
    }
  }'
```

## 配置说明

### 环境变量

SSE 事件功能使用与主服务相同的数据库配置：

```bash
# 数据库连接配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=things2024
DB_NAME=postgres
DB_SCHEMA=public
DB_SSLMODE=disable

# 或使用完整连接字符串
DATABASE_URL=postgres://postgres:things2024@localhost:5432/postgres?sslmode=disable&search_path=public
```

### 服务配置

事件服务在应用启动时自动初始化，无需额外配置。

## 使用场景

### 1. 实时数据监控

```javascript
// 监听数据变更事件
eventSource.onmessage = function (event) {
  const data = JSON.parse(event.data);
  if (data.type === "data_change") {
    // 更新前端数据显示
    updateDataDisplay(data.data);
  }
};
```

### 2. 系统通知推送

```javascript
// 显示系统通知
eventSource.onmessage = function (event) {
  const data = JSON.parse(event.data);
  if (data.type === "system_notification") {
    showNotification(data.data.title, data.data.message);
  }
};
```

### 3. 实时告警

```javascript
// 处理告警事件
eventSource.onmessage = function (event) {
  const data = JSON.parse(event.data);
  if (data.type === "alert") {
    showAlert(data.data);
  }
};
```

## 性能考虑

### 连接管理

- 每个用户可以建立多个 SSE 连接
- 系统自动清理断开的连接
- 建议限制单用户最大连接数

### 事件缓冲

- 每个连接有 100 个事件的缓冲队列
- 队列满时会跳过新事件发送
- 建议前端及时处理接收到的事件

### 数据库监听

- PostgreSQL LISTEN/NOTIFY 机制高效可靠
- 支持大量并发监听器
- 建议合理配置监听器数量和条件

## 故障排除

### 常见问题

1. **SSE 连接失败**

   - 检查服务是否正常运行
   - 确认用户名参数正确
   - 检查网络连接和防火墙设置

2. **未收到数据库事件**

   - 确认触发器已正确安装
   - 检查监听器配置是否匹配
   - 验证数据库连接是否正常

3. **事件发送失败**
   - 检查请求格式是否正确
   - 确认目标用户是否有活跃连接
   - 查看服务器日志获取详细错误信息

### 调试方法

1. **查看服务器日志**

   ```bash
   # 查看应用日志
   tail -f datahub-service.log
   ```

2. **检查数据库连接**

   ```sql
   -- 查看活跃的监听器
   SELECT * FROM pg_stat_activity WHERE query LIKE '%LISTEN%';
   ```

3. **测试触发器**
   ```sql
   -- 手动触发测试通知
   SELECT test_datahub_notification('Debug test');
   ```

## 扩展开发

### 自定义事件类型

可以根据业务需求定义新的事件类型：

```go
// 在事件服务中添加新的事件类型处理
func (s *EventService) SendCustomEvent(userName string, customData map[string]interface{}) error {
    event := &models.SSEEvent{
        EventType: "custom_business_event",
        UserName:  userName,
        Data:      customData,
        CreatedAt: time.Now(),
    }
    return s.SendEventToUser(userName, event)
}
```

### 添加新的监听表

为新的数据表添加变更监听：

```sql
-- 为新表创建触发器
DROP TRIGGER IF EXISTS new_table_notify ON new_table;
CREATE TRIGGER new_table_notify
    AFTER INSERT OR UPDATE OR DELETE ON new_table
    FOR EACH ROW EXECUTE FUNCTION notify_datahub_changes();
```

这样就完成了 SSE 事件功能的完整实现和文档说明。
