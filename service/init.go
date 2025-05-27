/*
 * @module service/init
 * @description 服务初始化模块，负责数据库连接、配置加载等初始化工作
 * @architecture 分层架构 - 服务层
 * @documentReference dev_docs/backend_requirements.md
 * @stateFlow 应用启动时执行初始化流程
 * @rules 确保所有依赖服务正常启动后才提供API服务
 * @dependencies gorm.io/gorm, gorm.io/driver/postgres
 * @refs dev_docs/model.md
 */

package service

import (
	"datahub-service/service/database"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	initDatabase()
	runMigrations()
}

// initDatabase 初始化数据库连接
func initDatabase() {
	var dsn string

	// 优先使用DATABASE_URL环境变量
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		dsn = databaseURL
	} else {
		// 使用分离的环境变量构建连接字符串
		host := getEnvWithDefault("DB_HOST", "localhost")
		port := getEnvWithDefault("DB_PORT", "5432")
		user := getEnvWithDefault("DB_USER", "postgres")
		password := getEnvWithDefault("DB_PASSWORD", "things2024")
		dbname := getEnvWithDefault("DB_NAME", "postgres")
		sslmode := getEnvWithDefault("DB_SSLMODE", "disable")
		schema := getEnvWithDefault("DB_SCHEMA", "public")

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s TimeZone=Asia/Shanghai",
			host, port, user, password, dbname, sslmode, schema)
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	log.Println("数据库连接成功")
}

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// runMigrations 运行数据库迁移
func runMigrations() {
	if err := database.AutoMigrate(DB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	if err := database.InitializeData(DB); err != nil {
		log.Fatalf("基础数据初始化失败: %v", err)
	}
}
