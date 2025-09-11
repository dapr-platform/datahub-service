/*
 * @module service/basic_library/datasource/postgresql
 * @description PostgreSQL数据源实现，支持连接池和SQL查询操作
 * @architecture 连接池模式 - 管理数据库连接的生命周期
 * @documentReference ai_docs/datasource_req.md, service/meta/datasource.go
 * @stateFlow PostgreSQL连接生命周期：初始化连接池 -> 获取连接 -> 执行SQL -> 归还连接 -> 关闭连接池
 * @rules 常驻数据源，维护连接池，支持事务和批量操作
 * @dependencies database/sql, github.com/lib/pq, context
 * @refs interface.go, base.go
 */

package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"datahub-service/service/meta"
	"datahub-service/service/models"

	_ "github.com/lib/pq" // PostgreSQL驱动
)

// PostgreSQLDataSource PostgreSQL数据源实现
type PostgreSQLDataSource struct {
	*BaseDataSource
	db           *sql.DB
	connStr      string
	maxConns     int
	maxIdleConns int
	connTimeout  time.Duration
}

// NewPostgreSQLDataSource 创建PostgreSQL数据源
func NewPostgreSQLDataSource() DataSourceInterface {
	base := NewBaseDataSource(meta.DataSourceTypeDBPostgreSQL, true) // PostgreSQL是常驻数据源
	return &PostgreSQLDataSource{
		BaseDataSource: base,
		maxConns:       100,
		maxIdleConns:   10,
		connTimeout:    30 * time.Second,
	}
}

// Init 初始化PostgreSQL数据源
func (p *PostgreSQLDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	if err := p.BaseDataSource.Init(ctx, ds); err != nil {
		return err
	}

	// 解析连接配置
	config := ds.ConnectionConfig
	if config == nil {
		return fmt.Errorf("连接配置不能为空")
	}

	// 构建连接字符串
	connStr, err := p.buildConnectionString(config)
	if err != nil {
		return fmt.Errorf("构建连接字符串失败: %v", err)
	}
	p.connStr = connStr

	// 解析参数配置
	if params := ds.ParamsConfig; params != nil {
		p.parseParamsConfig(params)
	}

	return nil
}

// Start 启动PostgreSQL数据源
func (p *PostgreSQLDataSource) Start(ctx context.Context) error {
	if err := p.BaseDataSource.Start(ctx); err != nil {
		return err
	}

	// 创建数据库连接
	db, err := sql.Open("postgres", p.connStr)
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(p.maxConns)
	db.SetMaxIdleConns(p.maxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	ctx, cancel := context.WithTimeout(ctx, p.connTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("数据库连接测试失败: %v", err)
	}

	p.db = db
	return nil
}

// Execute 执行PostgreSQL操作
func (p *PostgreSQLDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// 检查数据源状态
	if !p.IsInitialized() || !p.IsStarted() {
		response.Error = "数据源未启动"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("数据源未启动")
	}

	// 如果启用了脚本执行，优先使用脚本
	ds := p.GetDataSource()
	if ds.ScriptEnabled && ds.Script != "" {
		return p.BaseDataSource.Execute(ctx, request)
	}

	// 默认SQL执行处理
	return p.executeSQLQuery(ctx, request)
}

// Stop 停止PostgreSQL数据源
func (p *PostgreSQLDataSource) Stop(ctx context.Context) error {
	if p.db != nil {
		if err := p.db.Close(); err != nil {
			return fmt.Errorf("关闭数据库连接失败: %v", err)
		}
		p.db = nil
	}

	return p.BaseDataSource.Stop(ctx)
}

// HealthCheck PostgreSQL健康检查
func (p *PostgreSQLDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	baseStatus, err := p.BaseDataSource.HealthCheck(ctx)
	if err != nil {
		return baseStatus, err
	}

	// 如果基础检查失败，直接返回
	if baseStatus.Status != "online" {
		return baseStatus, nil
	}

	// 执行数据库连接测试
	startTime := time.Now()

	// 如果是常驻数据源且已启动，使用现有连接
	if p.db != nil {
		if err := p.db.PingContext(ctx); err != nil {
			baseStatus.Status = "error"
			baseStatus.Message = fmt.Sprintf("数据库连接测试失败: %v", err)
		} else {
			// 获取连接池统计信息
			stats := p.db.Stats()
			baseStatus.Details["connection_pool"] = map[string]interface{}{
				"max_open_connections": stats.MaxOpenConnections,
				"open_connections":     stats.OpenConnections,
				"in_use_connections":   stats.InUse,
				"idle_connections":     stats.Idle,
				"wait_count":           stats.WaitCount,
				"wait_duration":        stats.WaitDuration,
				"max_idle_closed":      stats.MaxIdleClosed,
				"max_idle_time_closed": stats.MaxIdleTimeClosed,
				"max_lifetime_closed":  stats.MaxLifetimeClosed,
			}
		}
	} else {
		// 对于非常驻数据源（测试模式），创建临时连接进行测试
		if !p.IsResident() && p.connStr != "" {
			// 添加调试信息
			baseStatus.Details["connection_string_length"] = len(p.connStr)
			baseStatus.Details["connection_timeout"] = p.connTimeout.String()

			tempDB, err := sql.Open("postgres", p.connStr)
			if err != nil {
				baseStatus.Status = "error"
				baseStatus.Message = fmt.Sprintf("创建临时数据库连接失败: %v", err)
				baseStatus.Details["error_type"] = "sql_open_failed"
			} else {
				// 设置更短的超时时间进行快速测试
				testTimeout := 10 * time.Second
				if p.connTimeout < testTimeout {
					testTimeout = p.connTimeout
				}
				testCtx, cancel := context.WithTimeout(ctx, testTimeout)
				defer cancel()

				// 设置连接参数，减少连接开销
				tempDB.SetMaxOpenConns(1)
				tempDB.SetMaxIdleConns(0)
				tempDB.SetConnMaxLifetime(testTimeout)

				// 测试连接
				if err := tempDB.PingContext(testCtx); err != nil {
					baseStatus.Status = "error"
					// 提供更详细的错误信息
					if err.Error() == "EOF" {
						baseStatus.Message = fmt.Sprintf("数据库连接被意外关闭（EOF）- 可能原因：1）服务器不可达 2）端口被防火墙阻止 3）SSL配置问题 4）连接超时。连接字符串长度: %d, 超时设置: %v", len(p.connStr), testTimeout)
						baseStatus.Details["error_type"] = "connection_eof"
						baseStatus.Details["possible_causes"] = []string{
							"服务器不可达或服务未启动",
							"端口35432被防火墙阻止",
							"SSL配置问题（当前使用sslmode=disable）",
							"网络连接不稳定",
							"服务器拒绝连接",
						}
					} else {
						baseStatus.Message = fmt.Sprintf("数据库连接测试失败: %v", err)
						baseStatus.Details["error_type"] = "ping_failed"
					}
					baseStatus.Details["original_error"] = err.Error()
				} else {
					baseStatus.Message = "数据库连接测试成功（临时连接）"
					baseStatus.Details["test_mode"] = "temporary_connection"
					baseStatus.Details["test_timeout"] = testTimeout.String()
				}

				// 立即关闭临时连接
				tempDB.Close()
			}
		} else {
			// 常驻数据源但未启动的情况
			baseStatus.Status = "offline"
			baseStatus.Message = "数据库连接未建立"
		}
	}

	baseStatus.ResponseTime = time.Since(startTime)
	return baseStatus, nil
}

// buildConnectionString 构建连接字符串
func (p *PostgreSQLDataSource) buildConnectionString(config map[string]interface{}) (string, error) {
	var parts []string

	// 主机
	if host, ok := config[meta.DataSourceFieldHost].(string); ok && host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", host))
	} else {
		return "", fmt.Errorf("主机地址不能为空")
	}

	// 端口
	if port, ok := config[meta.DataSourceFieldPort].(float64); ok {
		parts = append(parts, fmt.Sprintf("port=%d", int(port)))
	}

	// 数据库名
	if database, ok := config[meta.DataSourceFieldDatabase].(string); ok && database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", database))
	} else {
		return "", fmt.Errorf("数据库名不能为空")
	}

	// 用户名
	if username, ok := config[meta.DataSourceFieldUsername].(string); ok && username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", username))
	} else {
		return "", fmt.Errorf("用户名不能为空")
	}

	// 密码
	if password, ok := config[meta.DataSourceFieldPassword].(string); ok && password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", password))
	} else {
		return "", fmt.Errorf("密码不能为空")
	}

	// Schema
	if schema, ok := config[meta.DataSourceFieldSchema].(string); ok && schema != "" {
		parts = append(parts, fmt.Sprintf("search_path=%s", schema))
	}

	// SSL模式
	if sslMode, ok := config[meta.DataSourceFieldSSLMode].(string); ok && sslMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", sslMode))
	}

	return strings.Join(parts, " "), nil
}

// parseParamsConfig 解析参数配置
func (p *PostgreSQLDataSource) parseParamsConfig(params map[string]interface{}) {
	if timeout, ok := params[meta.DataSourceFieldTimeout].(float64); ok {
		p.connTimeout = time.Duration(timeout) * time.Second
	}

	if maxConns, ok := params[meta.DataSourceFieldMaxConnections].(float64); ok {
		p.maxConns = int(maxConns)
		p.maxIdleConns = p.maxConns / 10 // 设置为最大连接数的10%
		if p.maxIdleConns < 1 {
			p.maxIdleConns = 1
		}
	}
}

// executeSQLQuery 执行SQL查询
func (p *PostgreSQLDataSource) executeSQLQuery(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	if request.Query == "" {
		response.Error = "SQL查询语句不能为空"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("SQL查询语句不能为空")
	}

	// 设置查询超时
	queryTimeout := 30 * time.Second
	if request.Timeout > 0 {
		queryTimeout = request.Timeout
	}
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// 根据操作类型执行不同的SQL操作
	switch strings.ToLower(request.Operation) {
	case "query", "select", "":
		return p.executeSelectQuery(queryCtx, request.Query, response, startTime)
	case "insert", "update", "delete":
		return p.executeModifyQuery(queryCtx, request.Query, response, startTime)
	case "batch":
		return p.executeBatchQuery(queryCtx, request, response, startTime)
	default:
		response.Error = fmt.Sprintf("不支持的操作类型: %s", request.Operation)
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("不支持的操作类型: %s", request.Operation)
	}
}

// executeSelectQuery 执行查询操作
func (p *PostgreSQLDataSource) executeSelectQuery(ctx context.Context, query string, response *ExecuteResponse, startTime time.Time) (*ExecuteResponse, error) {
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		response.Error = fmt.Sprintf("执行查询失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		response.Error = fmt.Sprintf("获取列信息失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	// 读取数据
	var results []map[string]interface{}
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			response.Error = fmt.Sprintf("扫描行数据失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}

		// 构建结果行
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		response.Error = fmt.Sprintf("读取数据时发生错误: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	response.Success = true
	response.Data = results
	response.RowCount = int64(len(results))
	response.Duration = time.Since(startTime)
	response.Metadata["columns"] = columns
	response.Metadata["query"] = query

	return response, nil
}

// executeModifyQuery 执行修改操作
func (p *PostgreSQLDataSource) executeModifyQuery(ctx context.Context, query string, response *ExecuteResponse, startTime time.Time) (*ExecuteResponse, error) {
	result, err := p.db.ExecContext(ctx, query)
	if err != nil {
		response.Error = fmt.Sprintf("执行修改操作失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		response.Error = fmt.Sprintf("获取影响行数失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	response.Success = true
	response.RowCount = rowsAffected
	response.Duration = time.Since(startTime)
	response.Metadata["query"] = query
	response.Message = fmt.Sprintf("成功执行，影响 %d 行", rowsAffected)

	return response, nil
}

// executeBatchQuery 执行批量操作
func (p *PostgreSQLDataSource) executeBatchQuery(ctx context.Context, request *ExecuteRequest, response *ExecuteResponse, startTime time.Time) (*ExecuteResponse, error) {
	// 开始事务
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		response.Error = fmt.Sprintf("开始事务失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}
	defer tx.Rollback()

	var totalRowsAffected int64
	queries, ok := request.Data.([]interface{})
	if !ok {
		response.Error = "批量操作数据格式错误"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("批量操作数据格式错误")
	}

	for i, queryData := range queries {
		queryStr, ok := queryData.(string)
		if !ok {
			response.Error = fmt.Sprintf("第 %d 个查询格式错误", i+1)
			response.Duration = time.Since(startTime)
			return response, fmt.Errorf("第 %d 个查询格式错误", i+1)
		}

		result, err := tx.ExecContext(ctx, queryStr)
		if err != nil {
			response.Error = fmt.Sprintf("执行第 %d 个查询失败: %v", i+1, err)
			response.Duration = time.Since(startTime)
			return response, err
		}

		rowsAffected, _ := result.RowsAffected()
		totalRowsAffected += rowsAffected
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		response.Error = fmt.Sprintf("提交事务失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	response.Success = true
	response.RowCount = totalRowsAffected
	response.Duration = time.Since(startTime)
	response.Message = fmt.Sprintf("批量操作成功，共影响 %d 行", totalRowsAffected)
	response.Metadata["batch_size"] = len(queries)

	return response, nil
}
