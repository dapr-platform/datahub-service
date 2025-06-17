# 数据同步任务 API 实现说明

## 概述

本文档描述了在 `sync_controller.go` 中新实现的 SyncTask CRUD 功能，以及与 `schedule_service` 和 `sync_engine` 的集成。

## 架构设计

```
API请求 → SyncController → ScheduleService (任务CRUD) → SyncEngine (任务执行)
```

### 职责分工

- **SyncController**: 处理 HTTP 请求，参数验证，响应格式化
- **ScheduleService**: 负责同步任务的增删改查操作
- **SyncEngine**: 负责同步任务的实际执行和状态管理

## 新增 API 端点

### 1. 创建同步任务

- **路径**: `POST /api/sync/tasks`
- **功能**: 创建新的数据同步任务并提交到执行引擎
- **流程**:
  1. 验证请求参数
  2. 调用 `ScheduleService.CreateSyncTask()` 创建任务记录
  3. 调用 `SyncEngine.SubmitSyncTask()` 提交任务执行

### 2. 获取任务列表

- **路径**: `GET /api/sync/tasks`
- **功能**: 分页获取同步任务列表，支持筛选
- **参数**: `page`,`size`,`data_source_id`,`status`

### 3. 获取任务详情

- **路径**: `GET /api/sync/tasks/{id}`
- **功能**: 获取指定任务的详细信息
- **特性**: 对运行中的任务会合并实时进度信息

### 4. 更新任务配置

- **路径**: `PUT /api/sync/tasks/{id}`
- **功能**: 更新待执行任务的配置
- **限制**: 只能更新 `pending` 状态的任务

### 5. 取消任务

- **路径**: `POST /api/sync/tasks/{id}/cancel`
- **功能**: 取消同步任务
- **流程**:
  1. 调用 `ScheduleService.CancelSyncTask()` 更新数据库状态
  2. 调用 `SyncEngine.CancelTask()` 停止执行

### 6. 重试任务

- **路径**: `POST /api/sync/tasks/{id}/retry`
- **功能**: 重试失败的任务
- **流程**:
  1. 调用 `ScheduleService.RetryTask()` 创建重试任务
  2. 调用 `SyncEngine.SubmitSyncTask()` 提交执行

### 7. 获取统计信息

- **路径**: `GET /api/sync/tasks/statistics`
- **功能**: 获取任务统计信息
- **数据源**: 合并数据库统计和引擎实时统计

## 新增的服务方法

### ScheduleService 新增方法

```go
// 更新同步任务配置
func (s *ScheduleService) UpdateSyncTask(taskID string, config map[string]interface{}) (*models.SyncTask, error)
```

### SyncController 新增依赖

```go
type SyncController struct {
    scheduleService *basic_library.ScheduleService
    syncEngine      *sync_engine.SyncEngine
}
```

## 请求/响应结构体

### CreateSyncTaskRequest

```go
type CreateSyncTaskRequest struct {
    DataSourceID string                 // 数据源ID (必需)
    InterfaceID  string                 // 接口ID (可选)
    TaskType     string                 // 任务类型 (必需)
    Parameters   map[string]interface{} // 任务参数 (可选)
}
```

### UpdateSyncTaskRequest

```go
type UpdateSyncTaskRequest struct {
    Config map[string]interface{} // 更新的配置
}
```

### GetSyncTasksResponse

```go
type GetSyncTasksResponse struct {
    Tasks      []models.SyncTask
    Pagination PaginationResponse
}
```

## 集成流程

### 任务创建流程

1. 客户端调用 `POST /api/sync/tasks`
2. `SyncController.CreateSyncTask()` 验证参数
3. `ScheduleService.CreateSyncTask()` 创建任务记录
4. `SyncEngine.SubmitSyncTask()` 提交任务到执行队列
5. 返回任务信息给客户端

### 任务执行流程

1. `SyncEngine` 从任务队列获取任务
2. 根据任务类型选择相应的处理器
3. 执行同步操作并更新进度
4. 完成后更新任务状态到数据库

## 注意事项

1. **URL 参数提取**: 当前代码中使用 `temp_task_id` 占位符，实际部署时需要使用路由库（如 chi）提取路径参数

2. **错误处理**: 实现了详细的错误分类和 HTTP 状态码映射

3. **状态同步**: 运行中的任务会从 `SyncEngine` 获取实时状态信息

4. **事务管理**: 任务创建和提交执行是分离的，确保即使执行失败也能保留任务记录

## 使用示例

### 创建全量同步任务

```bash
curl -X POST http://localhost:8080/api/sync/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "data_source_id": "ds_123",
    "task_type": "full_sync",
    "parameters": {
      "batch_size": 1000,
      "timeout": 300
    }
  }'
```

### 获取任务列表

```bash
curl "http://localhost:8080/api/sync/tasks?page=1&size=10&status=running"
```

### 取消任务

```bash
curl -X POST http://localhost:8080/api/sync/tasks/task_123/cancel
```

## 扩展建议

1. **路由集成**: 集成到主路由器中，配置正确的路径参数提取
2. **认证授权**: 添加必要的身份验证和权限控制
3. **监控日志**: 添加详细的操作日志和监控指标
4. **批量操作**: 支持批量取消、重试等操作
5. **WebSocket**: 为长时间运行的任务提供实时状态更新
