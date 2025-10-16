/*
 * @module service/thematic_sync/sql_query_executor
 * @description SQL查询执行器，专门处理自定义SQL查询的执行
 * @architecture 服务层 - 提供安全的SQL执行能力
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow SQL配置 -> 参数验证 -> 查询执行 -> 结果转换
 * @rules 确保SQL执行的安全性，防止SQL注入，支持超时和行数限制
 * @dependencies gorm.io/gorm, context
 * @refs types.go, data_fetcher.go
 */

package thematic_sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SQLQueryExecutor SQL查询执行器
type SQLQueryExecutor struct {
	db *gorm.DB
}

// NewSQLQueryExecutor 创建SQL查询执行器
func NewSQLQueryExecutor(db *gorm.DB) *SQLQueryExecutor {
	return &SQLQueryExecutor{
		db: db,
	}
}

// ExecuteQuery 执行SQL查询
// 返回: 查询结果记录列表, 错误信息
func (sqe *SQLQueryExecutor) ExecuteQuery(ctx context.Context, config *SQLQueryConfig) ([]map[string]interface{}, error) {
	// 参数验证
	if err := sqe.validateSQLQuery(config); err != nil {
		return nil, fmt.Errorf("SQL查询验证失败: %w", err)
	}

	// 设置默认值
	timeout := 30
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	maxRows := 10000
	if config.MaxRows > 0 {
		maxRows = config.MaxRows
	}

	// 创建带超时的上下文
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 处理参数化查询
	sqlQuery := config.SQLQuery
	args := make([]interface{}, 0)

	// 如果有参数，替换占位符
	if len(config.Parameters) > 0 {
		sqlQuery, args = sqe.processParameters(sqlQuery, config.Parameters)
	}

	// 执行查询
	fmt.Printf("[DEBUG] 执行SQL查询: %s\n", sqlQuery)
	fmt.Printf("[DEBUG] 查询参数: %v\n", args)
	fmt.Printf("[DEBUG] 超时时间: %d秒, 最大行数: %d\n", timeout, maxRows)

	// 使用GORM执行原始SQL查询
	var results []map[string]interface{}

	// 构建查询，添加行数限制
	limitedQuery := sqlQuery
	if !strings.Contains(strings.ToUpper(sqlQuery), "LIMIT") {
		limitedQuery = fmt.Sprintf("%s LIMIT %d", sqlQuery, maxRows)
	}

	// 执行查询
	rows, err := sqe.db.WithContext(queryCtx).Raw(limitedQuery, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("执行SQL查询失败: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	// 扫描结果
	for rows.Next() {
		// 检查是否已达到最大行数
		if len(results) >= maxRows {
			fmt.Printf("[WARNING] 查询结果超过最大行数限制(%d)，停止获取\n", maxRows)
			break
		}

		// 准备扫描目标
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// 扫描行
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("扫描数据失败: %w", err)
		}

		// 转换为map
		record := make(map[string]interface{})
		for i, column := range columns {
			record[column] = sqe.convertDatabaseValue(values[i])
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	fmt.Printf("[DEBUG] SQL查询完成，返回记录数: %d\n", len(results))

	return results, nil
}

// ExecuteMultipleQueries 执行多个SQL查询
// 返回: 合并的查询结果, 错误信息
func (sqe *SQLQueryExecutor) ExecuteMultipleQueries(ctx context.Context, configs []*SQLQueryConfig) ([]map[string]interface{}, error) {
	var allResults []map[string]interface{}

	for i, config := range configs {
		fmt.Printf("[DEBUG] 执行第 %d/%d 个SQL查询\n", i+1, len(configs))

		results, err := sqe.ExecuteQuery(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("执行第%d个SQL查询失败: %w", i+1, err)
		}

		allResults = append(allResults, results...)
	}

	fmt.Printf("[DEBUG] 多个SQL查询完成，总记录数: %d\n", len(allResults))

	return allResults, nil
}

// validateSQLQuery 验证SQL查询的安全性
func (sqe *SQLQueryExecutor) validateSQLQuery(config *SQLQueryConfig) error {
	if config == nil {
		return fmt.Errorf("SQL查询配置为空")
	}

	if config.SQLQuery == "" {
		return fmt.Errorf("SQL查询语句为空")
	}

	// 去除首尾空格
	sqlQuery := strings.TrimSpace(config.SQLQuery)

	// 检查是否为SELECT查询（只允许查询操作，不允许修改操作）
	upperSQL := strings.ToUpper(sqlQuery)
	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") {
		return fmt.Errorf("只允许执行SELECT查询或WITH子句查询")
	}

	// 检查危险的SQL关键字（防止恶意操作）
	dangerousKeywords := []string{
		"DROP", "DELETE", "UPDATE", "INSERT", "ALTER",
		"TRUNCATE", "CREATE", "GRANT", "REVOKE",
	}

	for _, keyword := range dangerousKeywords {
		// 检查是否包含危险关键字
		if strings.Contains(upperSQL, keyword) {
			// 进一步检查是否真的是危险操作（可能在注释或字符串中）
			if sqe.isDangerousOperation(upperSQL, keyword) {
				return fmt.Errorf("SQL查询包含不允许的操作: %s", keyword)
			}
		}
	}

	return nil
}

// isDangerousOperation 判断是否为危险操作
func (sqe *SQLQueryExecutor) isDangerousOperation(sql, keyword string) bool {
	// 简化实现：检查关键字是否在SQL主体中（不在字符串或注释中）
	// 这里做基础检查，实际生产环境建议使用更严格的SQL解析器

	// 移除单行注释
	sql = sqe.removeComments(sql)

	// 检查关键字
	return strings.Contains(sql, keyword)
}

// removeComments 移除SQL注释
func (sqe *SQLQueryExecutor) removeComments(sql string) string {
	// 移除单行注释 --
	lines := strings.Split(sql, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// processParameters 处理参数化查询
// 支持 {{param_name}} 格式的参数占位符
func (sqe *SQLQueryExecutor) processParameters(sqlQuery string, parameters map[string]interface{}) (string, []interface{}) {
	// 替换 {{param_name}} 格式的占位符为 ? 或 $1, $2, ...
	processedSQL := sqlQuery
	args := make([]interface{}, 0)

	for key, value := range parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if strings.Contains(processedSQL, placeholder) {
			// PostgreSQL使用 $1, $2, ... 作为占位符
			processedSQL = strings.Replace(processedSQL, placeholder, fmt.Sprintf("$%d", len(args)+1), -1)
			args = append(args, value)
		}
	}

	return processedSQL, args
}

// convertDatabaseValue 转换数据库值为Go类型
func (sqe *SQLQueryExecutor) convertDatabaseValue(value interface{}) interface{} {
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

// ValidateQuerySafety 验证查询安全性（导出方法，供外部使用）
func (sqe *SQLQueryExecutor) ValidateQuerySafety(sqlQuery string) error {
	return sqe.validateSQLQuery(&SQLQueryConfig{
		SQLQuery: sqlQuery,
	})
}
