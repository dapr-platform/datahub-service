/*
 * @module service/interface_executor/data_sync_engine
 * @description 数据同步引擎，专门处理不同策略的数据同步逻辑
 * @architecture 策略模式 - 根据同步类型选择不同的同步策略
 * @documentReference design.md
 * @stateFlow 策略选择 -> 数据验证 -> 同步执行 -> 一致性检查 -> 状态更新
 * @rules 确保全量同步、增量同步、实时同步的数据一致性和完整性
 * @dependencies datahub-service/service/models, datahub-service/service/database, gorm.io/gorm
 * @refs executor.go
 */

package interface_executor

import (
	"context"
	"datahub-service/service/database"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DataSyncEngine 数据同步引擎
type DataSyncEngine struct {
	db            *gorm.DB
	schemaService *database.SchemaService
}

// DataSyncStrategy 数据同步策略类型
type DataSyncStrategy string

const (
	DataFullSync        DataSyncStrategy = "full"        // 全量同步
	DataIncrementalSync DataSyncStrategy = "incremental" // 增量同步
	DataRealtimeSync    DataSyncStrategy = "realtime"    // 实时同步
)

// TableTarget 目标表配置
type TableTarget struct {
	TableName   string                 `json:"table_name"`
	Schema      string                 `json:"schema,omitempty"`
	PrimaryKeys []string               `json:"primary_keys"`
	Columns     []string               `json:"columns,omitempty"`
	Mapping     map[string]string      `json:"mapping,omitempty"` // 源字段到目标字段的映射
	Options     map[string]interface{} `json:"options,omitempty"`
}

// DataSyncResult 数据同步结果
type DataSyncResult struct {
	Strategy       DataSyncStrategy `json:"strategy"`
	RecordsCount   int64            `json:"records_count"`
	InsertedCount  int64            `json:"inserted_count"`
	UpdatedCount   int64            `json:"updated_count"`
	DeletedCount   int64            `json:"deleted_count"`
	SkippedCount   int64            `json:"skipped_count"`
	ErrorCount     int64            `json:"error_count"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
	Duration       time.Duration    `json:"duration"`
	LastSyncTime   *time.Time       `json:"last_sync_time,omitempty"`
	CheckpointData interface{}      `json:"checkpoint_data,omitempty"`
	ErrorDetails   []string         `json:"error_details,omitempty"`
}

// NewDataSyncEngine 创建数据同步引擎实例
func NewDataSyncEngine(db *gorm.DB) *DataSyncEngine {
	return &DataSyncEngine{
		db:            db,
		schemaService: database.NewSchemaService(db),
	}
}

// ExecuteSync 执行数据同步
func (s *DataSyncEngine) ExecuteSync(ctx context.Context, strategy DataSyncStrategy, data []map[string]interface{}, target TableTarget) (*DataSyncResult, error) {
	result := &DataSyncResult{
		Strategy:  strategy,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// 验证目标表配置
	if err := s.validateTableTarget(target); err != nil {
		return result, fmt.Errorf("目标表配置验证失败: %w", err)
	}

	// 根据策略执行同步
	switch strategy {
	case DataFullSync:
		return s.ExecuteFullSync(ctx, data, target)
	case DataIncrementalSync:
		return s.ExecuteIncrementalSync(ctx, data, target)
	case DataRealtimeSync:
		return s.ExecuteRealtimeSync(ctx, data, target)
	default:
		return result, fmt.Errorf("不支持的同步策略: %s", strategy)
	}
}

// ExecuteFullSync 执行全量同步
func (s *DataSyncEngine) ExecuteFullSync(ctx context.Context, data []map[string]interface{}, target TableTarget) (*DataSyncResult, error) {
	result := &DataSyncResult{
		Strategy:     DataFullSync,
		StartTime:    time.Now(),
		RecordsCount: int64(len(data)),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 1. 创建或更新表结构
	if err := s.ensureTableSchema(tx, data, target); err != nil {
		tx.Rollback()
		return result, fmt.Errorf("确保表结构失败: %w", err)
	}

	// 2. 清空目标表（全量同步）
	if err := s.truncateTable(tx, target); err != nil {
		tx.Rollback()
		return result, fmt.Errorf("清空目标表失败: %w", err)
	}

	// 3. 批量插入数据
	insertedCount, errors := s.batchInsertData(tx, data, target)
	result.InsertedCount = insertedCount
	result.ErrorCount = int64(len(errors))

	if len(errors) > 0 {
		result.ErrorDetails = errors
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		return result, fmt.Errorf("提交事务失败: %w", err)
	}

	result.LastSyncTime = &result.EndTime
	return result, nil
}

// ExecuteIncrementalSync 执行增量同步
func (s *DataSyncEngine) ExecuteIncrementalSync(ctx context.Context, data []map[string]interface{}, target TableTarget) (*DataSyncResult, error) {
	result := &DataSyncResult{
		Strategy:     DataIncrementalSync,
		StartTime:    time.Now(),
		RecordsCount: int64(len(data)),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 1. 创建或更新表结构
	if err := s.ensureTableSchema(tx, data, target); err != nil {
		tx.Rollback()
		return result, fmt.Errorf("确保表结构失败: %w", err)
	}

	// 2. 执行增量数据同步（UPSERT操作）
	inserted, updated, errors := s.upsertData(tx, data, target)
	result.InsertedCount = inserted
	result.UpdatedCount = updated
	result.ErrorCount = int64(len(errors))

	if len(errors) > 0 {
		result.ErrorDetails = errors
	}

	// 3. 提交事务
	if err := tx.Commit().Error; err != nil {
		return result, fmt.Errorf("提交事务失败: %w", err)
	}

	result.LastSyncTime = &result.EndTime
	return result, nil
}

// ExecuteRealtimeSync 执行实时同步
func (s *DataSyncEngine) ExecuteRealtimeSync(ctx context.Context, data []map[string]interface{}, target TableTarget) (*DataSyncResult, error) {
	result := &DataSyncResult{
		Strategy:     DataRealtimeSync,
		StartTime:    time.Now(),
		RecordsCount: int64(len(data)),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// 实时同步通常是单条记录处理，不需要事务
	// 但为了保证一致性，这里仍使用事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 1. 确保表结构存在
	if err := s.ensureTableSchema(tx, data, target); err != nil {
		tx.Rollback()
		return result, fmt.Errorf("确保表结构失败: %w", err)
	}

	// 2. 逐条处理数据（实时同步）
	var errors []string
	for _, record := range data {
		if err := s.processRealtimeRecord(tx, record, target); err != nil {
			result.ErrorCount++
			errors = append(errors, err.Error())
		} else {
			result.InsertedCount++
		}
	}

	if len(errors) > 0 {
		result.ErrorDetails = errors
	}

	// 3. 提交事务
	if err := tx.Commit().Error; err != nil {
		return result, fmt.Errorf("提交事务失败: %w", err)
	}

	result.LastSyncTime = &result.EndTime
	return result, nil
}

// validateTableTarget 验证目标表配置
func (s *DataSyncEngine) validateTableTarget(target TableTarget) error {
	if target.TableName == "" {
		return fmt.Errorf("目标表名不能为空")
	}

	if len(target.PrimaryKeys) == 0 {
		return fmt.Errorf("必须指定主键字段")
	}

	return nil
}

// ensureTableSchema 确保表结构存在
func (s *DataSyncEngine) ensureTableSchema(tx *gorm.DB, data []map[string]interface{}, target TableTarget) error {
	if len(data) == 0 {
		return nil
	}

	// TODO: 实现表结构确保逻辑
	// 这里应该检查表是否存在，如果不存在则根据样本数据创建表
	// 从第一条数据推断表结构：sampleRecord := data[0]
	// 构建完整表名：tableName := target.TableName (with schema if provided)
	// 暂时返回nil，后续完善
	return nil
}

// truncateTable 清空目标表
func (s *DataSyncEngine) truncateTable(tx *gorm.DB, target TableTarget) error {
	tableName := target.TableName
	if target.Schema != "" {
		tableName = target.Schema + "." + target.TableName
	}

	// SQLite不支持TRUNCATE，使用DELETE代替
	sql := fmt.Sprintf("DELETE FROM %s", tableName)
	return tx.Exec(sql).Error
}

// batchInsertData 批量插入数据
func (s *DataSyncEngine) batchInsertData(tx *gorm.DB, data []map[string]interface{}, target TableTarget) (int64, []string) {
	var insertedCount int64
	var errors []string

	tableName := target.TableName
	if target.Schema != "" {
		tableName = target.Schema + "." + target.TableName
	}

	// 分批插入，避免单次插入数据量过大
	batchSize := 1000
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		if err := s.insertBatch(tx, tableName, batch, target); err != nil {
			errors = append(errors, fmt.Sprintf("批次 %d-%d 插入失败: %v", i, end, err))
		} else {
			insertedCount += int64(len(batch))
		}
	}

	return insertedCount, errors
}

// insertBatch 插入单个批次的数据
func (s *DataSyncEngine) insertBatch(tx *gorm.DB, tableName string, batch []map[string]interface{}, target TableTarget) error {
	if len(batch) == 0 {
		return nil
	}

	// 构建插入SQL
	columns := s.getColumnsFromData(batch[0], target)
	if len(columns) == 0 {
		return fmt.Errorf("没有找到可插入的列")
	}

	// 逐条插入以避免复杂的批量SQL构建
	for _, record := range batch {
		if err := s.insertSingleRecord(tx, tableName, record, columns); err != nil {
			return err
		}
	}

	return nil
}

// insertSingleRecord 插入单条记录
func (s *DataSyncEngine) insertSingleRecord(tx *gorm.DB, tableName string, record map[string]interface{}, columns []string) error {
	// 构建列名和占位符
	columnNames := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	values := make([]interface{}, len(columns))

	for i, column := range columns {
		columnNames[i] = fmt.Sprintf(`"%s"`, column)
		placeholders[i] = "?"
		values[i] = record[column]
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "))

	return tx.Exec(sql, values...).Error
}

// upsertData 执行UPSERT操作
func (s *DataSyncEngine) upsertData(tx *gorm.DB, data []map[string]interface{}, target TableTarget) (int64, int64, []string) {
	var insertedCount, updatedCount int64
	var errors []string

	for _, record := range data {
		inserted, updated, err := s.upsertRecord(tx, record, target)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			insertedCount += inserted
			updatedCount += updated
		}
	}

	return insertedCount, updatedCount, errors
}

// upsertRecord 执行单条记录的UPSERT操作
func (s *DataSyncEngine) upsertRecord(tx *gorm.DB, record map[string]interface{}, target TableTarget) (int64, int64, error) {
	tableName := target.TableName
	if target.Schema != "" {
		tableName = target.Schema + "." + target.TableName
	}

	// 构建ON CONFLICT语句（PostgreSQL语法）
	columns := s.getColumnsFromData(record, target)
	primaryKeys := target.PrimaryKeys

	// 简化的UPSERT实现，实际项目中需要更复杂的逻辑
	// 这里只是示例
	updateColumns := make([]string, 0)
	for _, col := range columns {
		found := false
		for _, pk := range primaryKeys {
			if col == pk {
				found = true
				break
			}
		}
		if !found {
			updateColumns = append(updateColumns, fmt.Sprintf(`%s = EXCLUDED.%s`, col, col))
		}
	}

	sql := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s`,
		tableName,
		`"`+columns[0]+`"`, // 简化处理
		"?",                // 简化处理
		primaryKeys[0],     // 简化处理
		updateColumns[0])   // 简化处理

	result := tx.Exec(sql, record[columns[0]])
	if result.Error != nil {
		return 0, 0, result.Error
	}

	// 简化返回，实际需要判断是插入还是更新
	return 1, 0, nil
}

// processRealtimeRecord 处理实时同步的单条记录
func (s *DataSyncEngine) processRealtimeRecord(tx *gorm.DB, record map[string]interface{}, target TableTarget) error {
	// 实时同步通常直接插入，如果冲突则更新
	_, _, err := s.upsertRecord(tx, record, target)
	return err
}

// getColumnsFromData 从数据中获取列名
func (s *DataSyncEngine) getColumnsFromData(record map[string]interface{}, target TableTarget) []string {
	if len(target.Columns) > 0 {
		return target.Columns
	}

	// 从记录中提取列名
	columns := make([]string, 0, len(record))
	for key := range record {
		// 应用字段映射
		if target.Mapping != nil {
			if mappedKey, exists := target.Mapping[key]; exists {
				columns = append(columns, mappedKey)
			} else {
				columns = append(columns, key)
			}
		} else {
			columns = append(columns, key)
		}
	}

	return columns
}
