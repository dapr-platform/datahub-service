# SyncTask 同步任务重构计划

## 当前问题分析

### 1. 模型层面的问题

- **SyncTask 模型位置不当**：目前定义在 `service/models/basic_library.go` 中，但应该是通用模型
- **缺少库类型区分**：只有 `DataSourceID` 和 `InterfaceID`，无法区分基础库和主题库
- **关联关系单一**：只能关联到基础库的数据源和接口

### 2. 控制器层面的问题

- **路径耦合**：API 路径硬编码为 `/basic-libraries/sync/tasks`，无法复用
- **业务逻辑耦合**：控制器直接依赖 `basic_library.ScheduleService`
- **缺少抽象**：无法处理主题库的同步需求

### 3. 服务层面的问题

- **包位置限制**：`ScheduleService` 位于 `service/basic_library` 包下
- **业务逻辑绑定**：只处理基础库相关的调度和同步逻辑
- **扩展性差**：添加主题库支持需要大量重复代码

### 4. 引擎层面的问题

- **假设单一**：`SyncEngine` 假设处理的都是基础库数据源
- **配置耦合**：任务配置和基础库强耦合

## 重构目标

### 1. 通用性

- 支持基础库和主题库的同步任务
- 提供统一的 API 接口
- 保持代码复用性

### 2. 可扩展性

- 易于添加新的库类型
- 支持不同库类型的特定业务逻辑
- 保持向后兼容性

### 3. 模块化

- 清晰的分层架构
- 低耦合高内聚
- 易于测试和维护

## 重构方案

### 阶段一：模型重构

#### 1.1 创建通用同步任务模型

**文件：`service/models/sync_task.go`**

```go
// SyncTask 通用同步任务模型
type SyncTask struct {
    ID             string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
    LibraryType    string                 `json:"library_type" gorm:"not null;size:20;index"`     // basic_library, thematic_library
    LibraryID      string                 `json:"library_id" gorm:"not null;type:varchar(36);index"` // 基础库ID或主题库ID
    DataSourceID   string                 `json:"data_source_id" gorm:"not null;type:varchar(36);index"`
    InterfaceID    *string                `json:"interface_id,omitempty" gorm:"type:varchar(36);index"`
    TaskType       string                 `json:"task_type" gorm:"not null;size:20"`
    Status         string                 `json:"status" gorm:"not null;size:20;default:'pending'"`
    StartTime      *time.Time             `json:"start_time,omitempty"`
    EndTime        *time.Time             `json:"end_time,omitempty"`
    Progress       int                    `json:"progress" gorm:"default:0"`
    ProcessedRows  int64                  `json:"processed_rows" gorm:"default:0"`
    TotalRows      int64                  `json:"total_rows" gorm:"default:0"`
    ErrorCount     int                    `json:"error_count" gorm:"default:0"`
    ErrorMessage   string                 `json:"error_message,omitempty" gorm:"type:text"`
    Config         map[string]interface{} `json:"config,omitempty" gorm:"type:jsonb"`
    Result         map[string]interface{} `json:"result,omitempty" gorm:"type:jsonb"`
    CreatedAt      time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
    CreatedBy      string                 `json:"created_by" gorm:"not null;default:'system';size:100"`
    UpdatedAt      time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

    // 动态关联
    BasicLibrary   *BasicLibrary   `json:"basic_library,omitempty" gorm:"-"`
    ThematicLibrary *ThematicLibrary `json:"thematic_library,omitempty" gorm:"-"`
    DataSource     DataSource      `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
    DataInterface  *DataInterface  `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}
```

#### 1.2 添加库类型常量

**文件：`service/meta/library_types.go`**

```go
// 库类型常量
const (
    LibraryTypeBasic    = "basic_library"
    LibraryTypeThematic = "thematic_library"
)

// 库类型验证
func IsValidLibraryType(libraryType string) bool {
    return libraryType == LibraryTypeBasic || libraryType == LibraryTypeThematic
}
```

### 阶段二：服务层重构

#### 2.1 创建通用同步任务服务

**文件：`service/sync_task_service.go`**

```go
// SyncTaskService 通用同步任务服务
type SyncTaskService struct {
    db             *gorm.DB
    basicLibService *basic_library.Service
    thematicLibService *thematic_library.Service
}

// LibraryHandler 库类型处理器接口
type LibraryHandler interface {
    ValidateLibrary(libraryID string) error
    ValidateDataSource(libraryID, dataSourceID string) error
    ValidateInterface(libraryID, interfaceID string) error
    GetLibraryInfo(libraryID string) (interface{}, error)
    PrepareTaskConfig(libraryID string, config map[string]interface{}) (map[string]interface{}, error)
}

// BasicLibraryHandler 基础库处理器
type BasicLibraryHandler struct {
    service *basic_library.Service
}

// ThematicLibraryHandler 主题库处理器
type ThematicLibraryHandler struct {
    service *thematic_library.Service
}
```

#### 2.2 重构调度服务

将 `ScheduleService` 从 `basic_library` 包移出，改为通用的调度服务，支持不同库类型。

### 阶段三：控制器重构

#### 3.1 创建通用同步控制器

**文件：`api/controllers/sync_task_controller.go`**

```go
// SyncTaskController 通用同步任务控制器
type SyncTaskController struct {
    syncTaskService *service.SyncTaskService
    syncEngine      *sync_engine.SyncEngine
}

// 路由设计：
// GET    /sync/tasks                    - 获取同步任务列表（支持library_type过滤）
// POST   /sync/tasks                    - 创建同步任务
// GET    /sync/tasks/{id}               - 获取同步任务详情
// PUT    /sync/tasks/{id}               - 更新同步任务
// DELETE /sync/tasks/{id}               - 删除同步任务
// POST   /sync/tasks/{id}/start         - 启动任务
// POST   /sync/tasks/{id}/stop          - 停止任务
// POST   /sync/tasks/{id}/cancel        - 取消任务
// POST   /sync/tasks/{id}/retry         - 重试任务
// GET    /sync/tasks/{id}/status        - 获取任务状态
// GET    /sync/tasks/statistics         - 获取统计信息
// POST   /sync/tasks/batch-delete       - 批量删除
// POST   /sync/tasks/cleanup            - 清理历史任务

// 保持兼容性的路由：
// GET    /basic-libraries/sync/tasks    - 重定向到 /sync/tasks?library_type=basic_library
// POST   /basic-libraries/sync/tasks    - 重定向到 /sync/tasks（自动设置library_type）
```

### 阶段四：引擎适配

#### 4.1 更新 SyncEngine

- 支持不同库类型的数据源处理
- 增强任务上下文信息
- 支持库特定的处理逻辑

#### 4.2 处理器增强

更新各个处理器以支持库类型区分：

- `BatchProcessor`
- `RealtimeProcessor`
- `DataTransformer`
- `IncrementalSync`

### 阶段五：数据库迁移

#### 5.1 数据迁移脚本

```sql
-- 添加新字段
ALTER TABLE sync_tasks ADD COLUMN library_type VARCHAR(20) NOT NULL DEFAULT 'basic_library';
ALTER TABLE sync_tasks ADD COLUMN library_id VARCHAR(36);

-- 数据迁移：将现有数据标记为基础库类型
UPDATE sync_tasks st
SET library_type = 'basic_library',
    library_id = (
        SELECT ds.library_id
        FROM data_sources ds
        WHERE ds.id = st.data_source_id
    );

-- 添加索引
CREATE INDEX idx_sync_tasks_library ON sync_tasks(library_type, library_id);
CREATE INDEX idx_sync_tasks_library_status ON sync_tasks(library_type, status);

-- 添加约束
ALTER TABLE sync_tasks ADD CONSTRAINT chk_library_type
CHECK (library_type IN ('basic_library', 'thematic_library'));
```

## 实施计划

### 第 1 天：模型和元数据重构 ✅ **已完成**

1. ✅ 创建 `service/models/sync_task.go` - 通用同步任务模型，支持基础库和主题库
2. ✅ 创建 `service/meta/library_types.go` - 库类型常量和验证函数
3. ✅ 更新元数据常量和验证函数 - 使用 meta 包统一管理
4. ✅ 运行单元测试 - 编译检查通过

### 第 2 天：服务层重构 ✅ **已完成**

1. ✅ 创建 `service/sync_task_service.go` - 通用同步任务服务
2. ✅ 实现 `LibraryHandler` 接口和具体实现 - 基础库和主题库处理器
3. ⏳ 重构调度服务 - 待实现（将在后续优化）
4. ⏳ 更新服务初始化逻辑 - 待与控制器一起更新

### 第 3 天：控制器重构 ✅ **已完成**

1. ✅ 创建 `api/controllers/sync_task_controller.go` - 通用同步任务控制器
2. ✅ 实现新的 API 接口 - CreateSyncTask 和 GetSyncTask 等核心接口
3. ✅ 添加兼容性路由 - CreateBasicLibrarySyncTask 向后兼容接口
4. ✅ 更新路由注册 - 在 routes.go 中添加 /sync/tasks 通用路由

### 第 4 天：引擎和处理器适配

1. 更新 `SyncEngine` 支持库类型
2. 增强各个处理器
3. 更新任务执行逻辑
4. 测试同步流程

### 第 5 天：数据库迁移和测试

1. 编写数据库迁移脚本
2. 执行数据迁移
3. 全面测试基础库功能
4. 测试主题库功能
5. 性能测试和优化

## 风险评估

### 高风险

- **数据迁移风险**：现有同步任务数据可能丢失
- **向后兼容性**：现有 API 可能失效
- **业务中断**：重构期间可能影响现有功能

### 缓解措施

1. **数据备份**：迁移前完整备份数据库
2. **渐进式重构**：保持现有 API 正常工作
3. **充分测试**：每个阶段都进行全面测试
4. **回滚计划**：准备快速回滚方案

## 测试策略

### 单元测试

- 新增模型的 CRUD 操作
- 服务层的业务逻辑
- 控制器的 API 接口

### 集成测试

- 基础库同步任务完整流程
- 主题库同步任务完整流程
- 跨库类型的操作

### 性能测试

- 大量同步任务的处理能力
- 数据库查询性能
- 并发同步任务处理

## 后续优化

### 功能增强

1. **任务依赖管理**：支持任务间的依赖关系
2. **分布式调度**：支持多节点任务调度
3. **监控告警**：增强任务监控和告警机制

### 架构优化

1. **微服务拆分**：将同步服务独立为微服务
2. **事件驱动**：引入事件驱动架构
3. **缓存优化**：添加适当的缓存机制

---

## 结论

通过以上重构方案，我们可以：

1. **统一同步任务模型**，支持基础库和主题库
2. **提供通用 API 接口**，保持向后兼容
3. **模块化设计**，易于扩展和维护
4. **保持高性能**，不影响现有功能

重构完成后，系统将具备更好的扩展性和维护性，为未来支持更多库类型奠定基础。
