/*
 * @module service/thematic_sync/sql_data_source
 * @description SQL数据源管理，支持通过SQL配置获取复杂数据
 * @architecture 策略模式 - 支持多种数据库和SQL执行策略
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow SQL解析 -> 参数绑定 -> 查询执行 -> 结果映射 -> 数据返回
 * @rules 确保SQL执行的安全性和性能，支持参数化查询
 * @dependencies gorm.io/gorm, database/sql
 * @refs models/thematic_sync.go, sync_engine.go
 */

package thematic_sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SQLDataSourceConfig SQL数据源配置
type SQLDataSourceConfig struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	SQLQuery    string                 `json:"sql_query"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     int                    `json:"timeout"`  // 查询超时时间（秒）
	MaxRows     int                    `json:"max_rows"` // 最大返回行数
}

// SQLDataSource SQL数据源
type SQLDataSource struct {
	db        *gorm.DB
	sqlDB     *sql.DB
	validator *SQLValidator
}

// NewSQLDataSource 创建SQL数据源
func NewSQLDataSource(db *gorm.DB) (*SQLDataSource, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取SQL DB失败: %w", err)
	}

	return &SQLDataSource{
		db:        db,
		sqlDB:     sqlDB,
		validator: NewSQLValidator(),
	}, nil
}

// ExecuteQuery 执行SQL查询
func (sds *SQLDataSource) ExecuteQuery(ctx context.Context, config SQLDataSourceConfig) ([]map[string]interface{}, error) {
	// 验证SQL安全性
	if err := sds.validator.ValidateSQL(config.SQLQuery); err != nil {
		return nil, fmt.Errorf("SQL验证失败: %w", err)
	}

	// 设置查询超时
	timeout := 30 * time.Second // 默认30秒
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 准备SQL语句
	preparedSQL, args, err := sds.prepareSQL(config.SQLQuery, config.Parameters)
	if err != nil {
		return nil, fmt.Errorf("SQL预处理失败: %w", err)
	}

	// 执行查询
	rows, err := sds.sqlDB.QueryContext(queryCtx, preparedSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("SQL执行失败: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	// 解析结果
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// 检查最大行数限制
		if config.MaxRows > 0 && len(results) >= config.MaxRows {
			break
		}

		// 创建扫描目标
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 构建结果映射
		record := make(map[string]interface{})
		for i, column := range columns {
			record[column] = sds.convertValue(values[i])
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果失败: %w", err)
	}

	return results, nil
}

// prepareSQL 预处理SQL语句
func (sds *SQLDataSource) prepareSQL(sqlQuery string, parameters map[string]interface{}) (string, []interface{}, error) {
	if len(parameters) == 0 {
		return sqlQuery, nil, nil
	}

	// 简单的参数替换实现
	// 实际项目中应该使用更安全的参数化查询
	preparedSQL := sqlQuery
	args := make([]interface{}, 0)

	for key, value := range parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if strings.Contains(preparedSQL, placeholder) {
			preparedSQL = strings.ReplaceAll(preparedSQL, placeholder, "?")
			args = append(args, value)
		}
	}

	return preparedSQL, args, nil
}

// convertValue 转换数据库值
func (sds *SQLDataSource) convertValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

// SQLValidator SQL验证器
type SQLValidator struct {
	forbiddenKeywords []string
	allowedOperations []string
}

// NewSQLValidator 创建SQL验证器
func NewSQLValidator() *SQLValidator {
	return &SQLValidator{
		forbiddenKeywords: []string{
			"DROP", "DELETE", "TRUNCATE", "UPDATE", "INSERT",
			"CREATE", "ALTER", "GRANT", "REVOKE", "EXEC", "EXECUTE",
		},
		allowedOperations: []string{
			"SELECT", "WITH", "UNION", "INTERSECT", "EXCEPT",
		},
	}
}

// ValidateSQL 验证SQL安全性
func (sv *SQLValidator) ValidateSQL(sqlQuery string) error {
	upperSQL := strings.ToUpper(strings.TrimSpace(sqlQuery))

	// 检查是否以允许的操作开头
	validStart := false
	for _, op := range sv.allowedOperations {
		if strings.HasPrefix(upperSQL, op) {
			validStart = true
			break
		}
	}

	if !validStart {
		return fmt.Errorf("SQL必须以允许的操作开头: %v", sv.allowedOperations)
	}

	// 检查禁用关键字
	for _, keyword := range sv.forbiddenKeywords {
		if strings.Contains(upperSQL, keyword) {
			return fmt.Errorf("SQL包含禁用关键字: %s", keyword)
		}
	}

	return nil
}

// BatchSQLDataSource 批量SQL数据源
type BatchSQLDataSource struct {
	dataSources map[string]*SQLDataSource
}

// NewBatchSQLDataSource 创建批量SQL数据源
func NewBatchSQLDataSource() *BatchSQLDataSource {
	return &BatchSQLDataSource{
		dataSources: make(map[string]*SQLDataSource),
	}
}

// AddDataSource 添加数据源
func (bsds *BatchSQLDataSource) AddDataSource(libraryID string, dataSource *SQLDataSource) {
	bsds.dataSources[libraryID] = dataSource
}

// ExecuteBatchQuery 执行批量查询
func (bsds *BatchSQLDataSource) ExecuteBatchQuery(ctx context.Context, configs []SQLDataSourceConfig) (map[string][]map[string]interface{}, error) {
	results := make(map[string][]map[string]interface{})

	for _, config := range configs {
		dataSource, exists := bsds.dataSources[config.LibraryID]
		if !exists {
			return nil, fmt.Errorf("数据源不存在: %s", config.LibraryID)
		}

		data, err := dataSource.ExecuteQuery(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("查询库 %s 失败: %w", config.LibraryID, err)
		}

		key := fmt.Sprintf("%s_%s", config.LibraryID, config.InterfaceID)
		results[key] = data
	}

	return results, nil
}
