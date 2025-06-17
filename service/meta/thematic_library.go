package meta

const (
	ThematicLibraryCategoryBusiness = "business"
	ThematicLibraryCategoryTechnical = "technical"
	ThematicLibraryCategoryAnalysis = "analysis"
	ThematicLibraryCategoryReport = "report"

	
)
type ThematicLibraryCategory struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
}
var ThematicLibraryCategories = []ThematicLibraryCategory{
	{ID: ThematicLibraryCategoryBusiness, Name: "业务", Description: "业务"},
	{ID: ThematicLibraryCategoryTechnical, Name: "技术", Description: "技术"},
	{ID: ThematicLibraryCategoryAnalysis, Name: "分析", Description: "分析"},
	{ID: ThematicLibraryCategoryReport, Name: "报告", Description: "报告"},
}	


const (
	ThematicLibraryDomainUser = "user"
	ThematicLibraryDomainOrder = "order"
	ThematicLibraryDomainProduct = "product"
	ThematicLibraryDomainFinance = "finance"
	ThematicLibraryDomainMarketing = "marketing"
	ThematicLibraryDomainAsset = "asset"
	ThematicLibraryDomainSupplyChain = "supply_chain"
	ThematicLibraryDomainParkOperation = "park_operation"
	ThematicLibraryDomainParkManagement = "park_management"
	ThematicLibraryDomainEmergencySafety = "emergency_safety"
	ThematicLibraryDomainEnergy = "energy"
)
type ThematicLibraryDomain struct {
	ID string `json:"id"`
	Name string `json:"name"`
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
	ThematicLibraryAccessLevelPublic = "public"
	ThematicLibraryAccessLevelInternal = "internal"
	ThematicLibraryAccessLevelPrivate = "private"
)
type ThematicLibraryAccessLevel struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
}
var ThematicLibraryAccessLevels = []ThematicLibraryAccessLevel{
	{ID: ThematicLibraryAccessLevelPublic, Name: "公开", Description: "公开"},
	{ID: ThematicLibraryAccessLevelInternal, Name: "内部", Description: "内部"},
	{ID: ThematicLibraryAccessLevelPrivate, Name: "私有", Description: "私有"},
}