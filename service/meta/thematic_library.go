package meta

const (
	ThematicLibraryCategoryBusiness  = "business"
	ThematicLibraryCategoryTechnical = "technical"
	ThematicLibraryCategoryAnalysis  = "analysis"
	ThematicLibraryCategoryReport    = "report"
)

type ThematicLibraryCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicLibraryCategories = []ThematicLibraryCategory{
	{ID: ThematicLibraryCategoryBusiness, Name: "业务", Description: "业务"},
	{ID: ThematicLibraryCategoryTechnical, Name: "技术", Description: "技术"},
	{ID: ThematicLibraryCategoryAnalysis, Name: "分析", Description: "分析"},
	{ID: ThematicLibraryCategoryReport, Name: "报告", Description: "报告"},
}

const (
	ThematicLibraryDomainUser            = "user"
	ThematicLibraryDomainOrder           = "order"
	ThematicLibraryDomainProduct         = "product"
	ThematicLibraryDomainFinance         = "finance"
	ThematicLibraryDomainMarketing       = "marketing"
	ThematicLibraryDomainAsset           = "asset"
	ThematicLibraryDomainSupplyChain     = "supply_chain"
	ThematicLibraryDomainParkOperation   = "park_operation"
	ThematicLibraryDomainParkManagement  = "park_management"
	ThematicLibraryDomainEmergencySafety = "emergency_safety"
	ThematicLibraryDomainEnergy          = "energy"
)

type ThematicLibraryDomain struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicLibraryDomains = []ThematicLibraryDomain{
	{ID: ThematicLibraryDomainUser, Name: "用户", Description: "用户"},
	{ID: ThematicLibraryDomainOrder, Name: "订单", Description: "订单"},
	{ID: ThematicLibraryDomainProduct, Name: "产品", Description: "产品"},
	{ID: ThematicLibraryDomainFinance, Name: "财务", Description: "财务"},
	{ID: ThematicLibraryDomainMarketing, Name: "营销", Description: "营销"},
	{ID: ThematicLibraryDomainAsset, Name: "资产", Description: "资产"},
	{ID: ThematicLibraryDomainSupplyChain, Name: "供应链", Description: "供应链"},
	{ID: ThematicLibraryDomainParkOperation, Name: "园区运营", Description: "园区运营"},
	{ID: ThematicLibraryDomainParkManagement, Name: "园区管理", Description: "园区管理"},
	{ID: ThematicLibraryDomainEmergencySafety, Name: "应急安全", Description: "应急安全"},
	{ID: ThematicLibraryDomainEnergy, Name: "能源", Description: "能源"},
}

const (
	ThematicLibraryAccessLevelPublic   = "public"
	ThematicLibraryAccessLevelInternal = "internal"
	ThematicLibraryAccessLevelPrivate  = "private"
)

type ThematicLibraryAccessLevel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicLibraryAccessLevels = []ThematicLibraryAccessLevel{
	{ID: ThematicLibraryAccessLevelPublic, Name: "公开", Description: "公开"},
	{ID: ThematicLibraryAccessLevelInternal, Name: "内部", Description: "内部"},
	{ID: ThematicLibraryAccessLevelPrivate, Name: "私有", Description: "私有"},
}

// 主题库状态常量
const (
	ThematicLibraryStatusDraft     = "draft"
	ThematicLibraryStatusPublished = "published"
	ThematicLibraryStatusArchived  = "archived"
)

type ThematicLibraryStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicLibraryStatuses = []ThematicLibraryStatus{
	{ID: ThematicLibraryStatusDraft, Name: "草稿", Description: "草稿状态，可以编辑修改"},
	{ID: ThematicLibraryStatusPublished, Name: "已发布", Description: "已发布状态，可以对外提供服务"},
	{ID: ThematicLibraryStatusArchived, Name: "已归档", Description: "已归档状态，不再提供服务"},
}

// 主题接口类型常量
const (
	ThematicInterfaceTypeRealtime = "realtime"
	ThematicInterfaceTypeBatch    = "batch"
)

type ThematicInterfaceType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicInterfaceTypes = []ThematicInterfaceType{
	{ID: ThematicInterfaceTypeRealtime, Name: "实时接口", Description: "提供实时数据访问服务"},
	{ID: ThematicInterfaceTypeBatch, Name: "批量接口", Description: "提供批量数据处理服务"},
}

// 主题接口状态常量
const (
	ThematicInterfaceStatusActive   = "active"
	ThematicInterfaceStatusInactive = "inactive"
)

type ThematicInterfaceStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ThematicInterfaceStatuses = []ThematicInterfaceStatus{
	{ID: ThematicInterfaceStatusActive, Name: "激活", Description: "接口正常运行，可以提供服务"},
	{ID: ThematicInterfaceStatusInactive, Name: "停用", Description: "接口已停用，暂不提供服务"},
}

// GetThematicLibraryCategories 获取主题库分类列表
func GetThematicLibraryCategories() []ThematicLibraryCategory {
	return ThematicLibraryCategories
}

// GetThematicLibraryDomains 获取主题库数据域列表
func GetThematicLibraryDomains() []ThematicLibraryDomain {
	return ThematicLibraryDomains
}

// GetThematicLibraryAccessLevels 获取主题库访问级别列表
func GetThematicLibraryAccessLevels() []ThematicLibraryAccessLevel {
	return ThematicLibraryAccessLevels
}

// GetThematicLibraryStatuses 获取主题库状态列表
func GetThematicLibraryStatuses() []ThematicLibraryStatus {
	return ThematicLibraryStatuses
}

// GetThematicInterfaceTypes 获取主题接口类型列表
func GetThematicInterfaceTypes() []ThematicInterfaceType {
	return ThematicInterfaceTypes
}

// GetThematicInterfaceStatuses 获取主题接口状态列表
func GetThematicInterfaceStatuses() []ThematicInterfaceStatus {
	return ThematicInterfaceStatuses
}

// IsValidThematicLibraryCategory 验证主题库分类是否有效
func IsValidThematicLibraryCategory(category string) bool {
	for _, cat := range ThematicLibraryCategories {
		if cat.ID == category {
			return true
		}
	}
	return false
}

// IsValidThematicLibraryDomain 验证主题库数据域是否有效
func IsValidThematicLibraryDomain(domain string) bool {
	for _, dom := range ThematicLibraryDomains {
		if dom.ID == domain {
			return true
		}
	}
	return false
}

// IsValidThematicLibraryStatus 验证主题库状态是否有效
func IsValidThematicLibraryStatus(status string) bool {
	for _, stat := range ThematicLibraryStatuses {
		if stat.ID == status {
			return true
		}
	}
	return false
}

// IsValidThematicInterfaceType 验证主题接口类型是否有效
func IsValidThematicInterfaceType(interfaceType string) bool {
	for _, typ := range ThematicInterfaceTypes {
		if typ.ID == interfaceType {
			return true
		}
	}
	return false
}

// IsValidThematicInterfaceStatus 验证主题接口状态是否有效
func IsValidThematicInterfaceStatus(status string) bool {
	for _, stat := range ThematicInterfaceStatuses {
		if stat.ID == status {
			return true
		}
	}
	return false
}
