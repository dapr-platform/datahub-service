/*
 * @module service/basic_library/datasource/base
 * @description 数据源基础实现，提供通用功能和抽象基类
 * @architecture 模板方法模式 - 定义数据源操作的通用流程
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 数据源状态管理：初始化 -> 启动 -> 运行 -> 停止
 * @rules 所有具体数据源继承基础实现，重写特定方法
 * @dependencies github.com/traefik/yaegi, sync, context
 * @refs interface.go, service/models/basic_library.go
 */

package datasource

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"datahub-service/service/models"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// BaseDataSource 基础数据源实现
type BaseDataSource struct {
	mu             sync.RWMutex
	id             string
	dsType         string
	dataSource     *models.DataSource
	isInitialized  bool
	isStarted      bool
	isResident     bool // 可以被修改的常驻状态
	lastHealthTime time.Time
	scriptExecutor ScriptExecutor
}

// NewBaseDataSource 创建基础数据源实例
func NewBaseDataSource(dsType string, isResident bool) *BaseDataSource {
	return &BaseDataSource{
		dsType:         dsType,
		isResident:     isResident,
		scriptExecutor: NewYaegiScriptExecutor(),
	}
}

// Init 初始化数据源
func (b *BaseDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ds == nil {
		return fmt.Errorf("数据源配置不能为空")
	}

	if b.isInitialized {
		return fmt.Errorf("数据源 %s 已经初始化", ds.ID)
	}

	b.id = ds.ID
	b.dataSource = ds
	b.isInitialized = true

	return nil
}

// Start 启动数据源（基础实现为空，子类重写）
func (b *BaseDataSource) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isInitialized {
		return fmt.Errorf("数据源 %s 未初始化", b.id)
	}

	if b.isStarted {
		return fmt.Errorf("数据源 %s 已经启动", b.id)
	}

	b.isStarted = true
	return nil
}

// Execute 执行操作（基础实现，子类重写）
func (b *BaseDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.isInitialized {
		return nil, fmt.Errorf("数据源 %s 未初始化", b.id)
	}

	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	// 如果启用了脚本执行，先执行脚本
	if b.dataSource.ScriptEnabled && b.dataSource.Script != "" {
		scriptResult, err := b.executeScript(ctx, request)
		if err != nil {
			response.Error = fmt.Sprintf("脚本执行失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}

		// 如果脚本返回了结果，直接返回
		if scriptResult != nil {
			response.Success = true
			response.Data = scriptResult
			response.Duration = time.Since(startTime)
			return response, nil
		}
	}

	// 默认实现返回未实现错误
	response.Error = "Execute方法未实现"
	response.Duration = time.Since(startTime)
	return response, fmt.Errorf("Execute方法未实现")
}

// Stop 停止数据源
func (b *BaseDataSource) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isStarted {
		return nil
	}

	b.isStarted = false
	return nil
}

// HealthCheck 健康检查
func (b *BaseDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	startTime := time.Now()
	status := &HealthStatus{
		LastCheck:    startTime,
		ResponseTime: 0,
		Details:      make(map[string]interface{}),
	}

	if !b.isInitialized {
		status.Status = "offline"
		status.Message = "数据源未初始化"
		status.ResponseTime = time.Since(startTime)
		return status, nil
	}

	if !b.isStarted && b.isResident {
		status.Status = "offline"
		status.Message = "数据源未启动"
		status.ResponseTime = time.Since(startTime)
		return status, nil
	}

	status.Status = "online"
	status.Message = "数据源正常"
	status.ResponseTime = time.Since(startTime)
	status.Details["type"] = b.dsType
	status.Details["resident"] = b.isResident
	status.Details["initialized"] = b.isInitialized
	status.Details["started"] = b.isStarted

	b.lastHealthTime = startTime
	return status, nil
}

// GetType 获取数据源类型
func (b *BaseDataSource) GetType() string {
	return b.dsType
}

// GetID 获取数据源ID
func (b *BaseDataSource) GetID() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.id
}

// IsResident 是否为常驻数据源
func (b *BaseDataSource) IsResident() bool {
	return b.isResident
}

// SetResident 设置常驻状态（用于测试场景）
func (b *BaseDataSource) SetResident(isResident bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.isResident = isResident
}

// GetDataSource 获取数据源模型（受保护方法，供子类使用）
func (b *BaseDataSource) GetDataSource() *models.DataSource {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.dataSource
}

// IsInitialized 检查是否已初始化
func (b *BaseDataSource) IsInitialized() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isInitialized
}

// IsStarted 检查是否已启动
func (b *BaseDataSource) IsStarted() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isStarted
}

// executeScript 执行脚本
func (b *BaseDataSource) executeScript(ctx context.Context, request *ExecuteRequest) (interface{}, error) {
	if b.scriptExecutor == nil {
		return nil, fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := make(map[string]interface{})
	params["request"] = request
	params["dataSource"] = b.dataSource
	params["connectionConfig"] = b.dataSource.ConnectionConfig
	params["paramsConfig"] = b.dataSource.ParamsConfig

	return b.scriptExecutor.Execute(ctx, b.dataSource.Script, params)
}

// YaegiScriptExecutor Yaegi脚本执行器实现 - 优化版，支持缓存和参数注入
type YaegiScriptExecutor struct {
	mu    sync.RWMutex
	cache map[string]*CompiledScript
}

// CompiledScript 编译后的脚本，保存可执行函数
type CompiledScript struct {
	fn       func(map[string]interface{}) (interface{}, error)
	compiled time.Time // 编译时间
	hash     string    // 脚本哈希
}

// NewYaegiScriptExecutor 创建Yaegi脚本执行器
func NewYaegiScriptExecutor() *YaegiScriptExecutor {
	return &YaegiScriptExecutor{
		cache: make(map[string]*CompiledScript),
	}
}

// Execute 执行脚本（带参数注入和缓存优化）
func (y *YaegiScriptExecutor) Execute(ctx context.Context, script string, params map[string]interface{}) (interface{}, error) {
	// 使用脚本内容的哈希作为缓存key
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(script)))

	// 先查缓存
	y.mu.RLock()
	compiled, ok := y.cache[hash]
	y.mu.RUnlock()

	if !ok {
		// 没有缓存则编译
		var err error
		compiled, err = y.compile(script, hash)
		if err != nil {
			return nil, fmt.Errorf("脚本编译失败: %v", err)
		}

		// 存入缓存
		y.mu.Lock()
		y.cache[hash] = compiled
		y.mu.Unlock()
	}

	// 调用编译后的函数
	return compiled.fn(params)
}

// compile 编译脚本为可执行函数
func (y *YaegiScriptExecutor) compile(script, hash string) (*CompiledScript, error) {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("加载标准库失败: %w", err)
	}

	// 包装脚本：要求脚本必须实现一个 Run 函数
	wrapped := fmt.Sprintf(`
package main

import (
	"fmt"
	"context"
	"time"
	"encoding/json"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
)

// 必须提供一个 Run 函数作为入口
func Run(params map[string]interface{}) (interface{}, error) {
	// 从参数中提取常用变量，方便脚本使用
	var operation interface{}
	if op, exists := params["operation"]; exists {
		operation = op
	}
	
	var request interface{}
	if req, exists := params["request"]; exists {
		request = req
	}
	
	var dataSource interface{}
	if ds, exists := params["dataSource"]; exists {
		dataSource = ds
	}
	
	var baseURL interface{}
	if url, exists := params["baseURL"]; exists {
		baseURL = url
	}
	
	var credentials interface{}
	if creds, exists := params["credentials"]; exists {
		credentials = creds
	}
	
	var updateSessionData interface{}
	if fn, exists := params["updateSessionData"]; exists {
		updateSessionData = fn
	}
	
	var getSessionData interface{}
	if fn, exists := params["getSessionData"]; exists {
		getSessionData = fn
	}
	
	var httpPost interface{}
	if fn, exists := params["httpPost"]; exists {
		httpPost = fn
	}
	
	var httpGet interface{}
	if fn, exists := params["httpGet"]; exists {
		httpGet = fn
	}

	// 脚本内容
%s
}
`, script)

	_, err := i.Eval(wrapped)
	if err != nil {
		return nil, fmt.Errorf("脚本编译失败: %w", err)
	}

	// 获取 Run 函数
	v, err := i.Eval("Run")
	if err != nil {
		return nil, fmt.Errorf("脚本缺少 Run 函数: %w", err)
	}

	runFunc, ok := v.Interface().(func(map[string]interface{}) (interface{}, error))
	if !ok {
		return nil, fmt.Errorf("Run 函数签名必须是 func(map[string]interface{}) (interface{}, error)")
	}

	return &CompiledScript{
		fn:       runFunc,
		compiled: time.Now(),
		hash:     hash,
	}, nil
}

// GetCacheStats 获取缓存统计信息
func (y *YaegiScriptExecutor) GetCacheStats() map[string]interface{} {
	y.mu.RLock()
	defer y.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["cache_size"] = len(y.cache)

	if len(y.cache) > 0 {
		oldestTime := time.Now()
		newestTime := time.Time{}

		for _, compiled := range y.cache {
			if compiled.compiled.Before(oldestTime) {
				oldestTime = compiled.compiled
			}
			if compiled.compiled.After(newestTime) {
				newestTime = compiled.compiled
			}
		}

		stats["oldest_compiled"] = oldestTime
		stats["newest_compiled"] = newestTime
	}

	return stats
}

// ClearCache 清理缓存
func (y *YaegiScriptExecutor) ClearCache() {
	y.mu.Lock()
	defer y.mu.Unlock()
	y.cache = make(map[string]*CompiledScript)
}

// Validate 验证脚本语法（快速校验）
func (y *YaegiScriptExecutor) Validate(script string) error {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return fmt.Errorf("加载标准库符号失败: %v", err)
	}

	// 包装脚本进行语法检查，使用与compile相同的包装逻辑
	wrapped := fmt.Sprintf(`
package main

import (
	"fmt"
	"context"
	"time"
	"encoding/json"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
)

func Run(params map[string]interface{}) (interface{}, error) {
	// 从参数中提取常用变量，方便脚本使用
	var operation interface{}
	if op, exists := params["operation"]; exists {
		operation = op
	}
	
	var request interface{}
	if req, exists := params["request"]; exists {
		request = req
	}
	
	var dataSource interface{}
	if ds, exists := params["dataSource"]; exists {
		dataSource = ds
	}
	
	var baseURL interface{}
	if url, exists := params["baseURL"]; exists {
		baseURL = url
	}
	
	var credentials interface{}
	if creds, exists := params["credentials"]; exists {
		credentials = creds
	}
	
	var updateSessionData interface{}
	if fn, exists := params["updateSessionData"]; exists {
		updateSessionData = fn
	}
	
	var getSessionData interface{}
	if fn, exists := params["getSessionData"]; exists {
		getSessionData = fn
	}
	
	var httpPost interface{}
	if fn, exists := params["httpPost"]; exists {
		httpPost = fn
	}
	
	var httpGet interface{}
	if fn, exists := params["httpGet"]; exists {
		httpGet = fn
	}

	// 脚本内容
%s
}
`, script)

	_, err := i.Compile(wrapped)
	return err
}

// DefaultDataSourceFactory 默认数据源工厂实现
type DefaultDataSourceFactory struct {
	mu       sync.RWMutex
	creators map[string]DataSourceCreator
}

// NewDefaultDataSourceFactory 创建默认数据源工厂
func NewDefaultDataSourceFactory() *DefaultDataSourceFactory {
	return &DefaultDataSourceFactory{
		creators: make(map[string]DataSourceCreator),
	}
}

// Create 创建数据源实例
func (f *DefaultDataSourceFactory) Create(dsType string) (DataSourceInterface, error) {
	f.mu.RLock()
	creator, exists := f.creators[dsType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("不支持的数据源类型: %s", dsType)
	}

	return creator(), nil
}

// GetSupportedTypes 获取支持的数据源类型列表
func (f *DefaultDataSourceFactory) GetSupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.creators))
	for dsType := range f.creators {
		types = append(types, dsType)
	}
	return types
}

// RegisterType 注册新的数据源类型
func (f *DefaultDataSourceFactory) RegisterType(dsType string, creator DataSourceCreator) error {
	if dsType == "" {
		return fmt.Errorf("数据源类型不能为空")
	}
	if creator == nil {
		return fmt.Errorf("数据源创建器不能为空")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[dsType] = creator
	return nil
}
