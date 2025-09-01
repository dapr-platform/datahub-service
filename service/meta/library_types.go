/*
 * @module service/meta/library_types
 * @description 库类型常量定义和验证函数，支持基础库和主题库的统一管理
 * @architecture 常量层 - 元数据定义
 * @documentReference ai_docs/refactor_sync_task.md
 * @stateFlow 常量定义 -> 验证函数 -> 业务逻辑使用
 * @rules 统一管理所有库类型相关的常量，确保类型安全
 * @dependencies 无外部依赖
 * @refs service/models, service/sync_task_service
 */

package meta

// 库类型常量
const (
	// LibraryTypeBasic 基础库类型
	LibraryTypeBasic = "basic_library"

	// LibraryTypeThematic 主题库类型
	LibraryTypeThematic = "thematic_library"
)

// 库类型显示名称映射
var LibraryTypeDisplayNames = map[string]string{
	LibraryTypeBasic:    "基础库",
	LibraryTypeThematic: "主题库",
}

// 库类型描述映射
var LibraryTypeDescriptions = map[string]string{
	LibraryTypeBasic:    "存储基础数据的库，如用户信息、组织架构等",
	LibraryTypeThematic: "按主题组织的数据库，如业务主题、分析主题等",
}

// IsValidLibraryType 验证库类型是否有效
func IsValidLibraryType(libraryType string) bool {
	validTypes := map[string]bool{
		LibraryTypeBasic:    true,
		LibraryTypeThematic: true,
	}
	return validTypes[libraryType]
}

// GetLibraryTypeDisplayName 获取库类型的显示名称
func GetLibraryTypeDisplayName(libraryType string) string {
	if displayName, exists := LibraryTypeDisplayNames[libraryType]; exists {
		return displayName
	}
	return "未知类型"
}

// GetLibraryTypeDescription 获取库类型的描述
func GetLibraryTypeDescription(libraryType string) string {
	if description, exists := LibraryTypeDescriptions[libraryType]; exists {
		return description
	}
	return "未知库类型"
}

// GetAllLibraryTypes 获取所有支持的库类型
func GetAllLibraryTypes() []string {
	return []string{
		LibraryTypeBasic,
		LibraryTypeThematic,
	}
}

// GetLibraryTypeInfo 获取库类型的完整信息
func GetLibraryTypeInfo(libraryType string) map[string]interface{} {
	return map[string]interface{}{
		"type":         libraryType,
		"display_name": GetLibraryTypeDisplayName(libraryType),
		"description":  GetLibraryTypeDescription(libraryType),
		"is_valid":     IsValidLibraryType(libraryType),
	}
}

// GetAllLibraryTypeInfo 获取所有库类型的信息
func GetAllLibraryTypeInfo() []map[string]interface{} {
	var result []map[string]interface{}
	for _, libraryType := range GetAllLibraryTypes() {
		result = append(result, GetLibraryTypeInfo(libraryType))
	}
	return result
}
