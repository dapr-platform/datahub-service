package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

func CheckSchemaExists(db *gorm.DB, schemaName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?", schemaName).Scan(&count)
	return count > 0
}

func CreateSchema(db *gorm.DB, schemaName string) error {
	log.Printf("开始创建 schema: %s", schemaName)

	// 1. 创建 schema (使用双引号避免保留关键字问题)
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\";", schemaName)
	if err := db.Exec(createSchemaSQL).Error; err != nil {
		return fmt.Errorf("创建 schema %s 失败: %v", schemaName, err)
	}

	// 2. 在 postgrest.schema_config 表中插入记录
	insertConfigSQL := `
		INSERT INTO postgrest.schema_config (schema_name, db_schemas) 
		VALUES (?, ?) 	
		ON CONFLICT (schema_name) DO NOTHING;`

	if err := db.Exec(insertConfigSQL, schemaName, schemaName).Error; err != nil {
		log.Printf("插入 postgrest.schema_config 记录失败: %v", err)
		// 不返回错误，因为 schema 已经创建成功
	}

	log.Printf("成功创建 schema: %s", schemaName)
	return nil
}

// deleteSchema 删除 schema 和 postgrest 配置
func DeleteSchema(db *gorm.DB, schemaName string) error {
	log.Printf("开始删除 schema: %s", schemaName)

	// 1. 从 postgrest.schema_config 表中删除记录
	deleteConfigSQL := "DELETE FROM postgrest.schema_config WHERE schema_name = ?;"
	if err := db.Exec(deleteConfigSQL, schemaName).Error; err != nil {
		log.Printf("删除 postgrest.schema_config 记录失败: %v", err)
		// 继续执行 schema 删除
	}

	// 2. 强制删除 schema（CASCADE 会删除 schema 中的所有对象，使用双引号避免保留关键字问题）
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE;", schemaName)
	if err := db.Exec(dropSchemaSQL).Error; err != nil {
		return fmt.Errorf("删除 schema %s 失败: %v", schemaName, err)
	}

	log.Printf("成功删除 schema: %s", schemaName)
	return nil
}

// UpdateUserSchemas 更新用户的schema权限
func UpdateUserSchemas(db *gorm.DB, userName, newSchemas string) error {
	log.Printf("开始更新用户 %s 的 schema 权限: %s", userName, newSchemas)

	// 调用 postgrest.update_user_schemas 函数
	sql := `SELECT postgrest.update_user_schemas($1, $2)`

	var result string
	if err := db.Raw(sql, userName, newSchemas).Scan(&result).Error; err != nil {
		return fmt.Errorf("调用 postgrest.update_user_schemas 失败: %v", err)
	}

	log.Printf("用户 schema 权限更新结果: %s", result)
	return nil
}

// DeletePostgRESTUser 调用PostgREST的delete_user函数删除用户
func DeletePostgRESTUser(db *gorm.DB, userName string) error {
	log.Printf("开始删除 PostgREST 用户: %s", userName)

	// 调用 postgrest.delete_user 函数
	sql := `SELECT postgrest.delete_user($1, true)`

	var result string
	if err := db.Raw(sql, userName).Scan(&result).Error; err != nil {
		return fmt.Errorf("调用 postgrest.delete_user 失败: %v", err)
	}

	log.Printf("PostgREST 用户删除结果: %s", result)
	return nil
}
