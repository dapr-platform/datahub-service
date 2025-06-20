# 数据底座服务 (DataHub Service) 概览

## 项目简介

数据底座服务是智慧园区微服务架构中的核心数据管理服务，基于 Dapr 框架构建，使用 Docker Compose 进行部署。该服务提供统一的数据存储、管理、治理和共享能力，支持多种数据源的集成和多样化的数据服务。

## 核心功能

### 1. 数据存储管理

- **基础库管理**：存储未经加工的原始数据，保持数据完整性
- **主题库管理**：按业务主题组织的数据集合，支持复杂查询和分析
- **数据分类**：支持会员数据、资产数据、供应链数据、园区运行数据等多种数据分类

### 2. 数据治理

- **权限管理**：集成 PostgREST RBAC 系统，提供细粒度的权限控制
- **数据质量管理**：监控数据完整性、准确性、时效性等质量指标
- **数据脱敏**：保护敏感信息，支持多级脱敏策略
- **审计日志**：完整记录数据访问和操作历史

### 3. 数据共享服务

- **数据 API**：提供 RESTful API 接口，支持实时数据访问
- **数据订阅**：支持数据变更通知和自动分发
- **库表同步**：支持全量和增量数据同步
- **数据目录**：提供数据发现和元数据管理功能

### 4. 数据监控和分析

- **性能监控**：实时监控 API 调用性能和系统状态
- **使用统计**：分析数据访问模式和用户行为
- **存储管理**：监控存储空间使用和容量规划
- **质量报告**：生成数据质量分析报告

## 新增视图系统

### 📊 数据质量监控视图

- **v_data_completeness_stats**：数据完整性统计，监控各库的数据完整性情况
- **v_data_accuracy_monitor**：数据准确性监控，跟踪验证结果和准确性指标
- **v_data_timeliness_analysis**：数据时效性分析，确保数据按时更新

### 🗺️ 数据资产管理视图

- **v_data_asset_map**：数据资产地图，提供全景数据资产视图
- **v_data_usage_stats**：数据使用统计，支持热度分析和资源优化

### 📈 数据访问监控视图

- **v_api_call_stats**：API 调用统计，监控性能和成功率
- **v_user_behavior_analysis**：用户行为分析，优化用户体验

### 🔐 数据治理视图

- **v_permission_overview**：权限分配概览，支持权限管理和审计
- **v_data_masking_status**：数据脱敏状态，确保敏感数据保护

### ⚡ 运营监控视图

- **v_system_performance_monitor**：系统性能监控，确保 SLA 达标
- **v_storage_usage_monitor**：存储使用监控，支持容量规划

### 🔗 数据血缘视图

- **v_data_lineage**：数据血缘关系，追踪数据流转和影响分析

### 📋 综合仪表板视图

- **v_datahub_dashboard**：数据底座概览，提供关键指标快速查看

## 技术架构

### 后端技术栈

- **语言**：Go 1.21+
- **框架**：Gin Web Framework
- **数据库**：PostgreSQL 15+
- **ORM**：GORM v2
- **权限管理**：PostgREST RBAC
- **微服务框架**：Dapr
- **容器化**：Docker & Docker Compose

### 数据库设计

- **基础表**：basic_libraries, thematic_libraries, governance_policies, sharing_logs
- **权限表**：集成 PostgREST 的 users, roles, permissions 体系
- **监控视图**：13 个业务视图，覆盖质量、资产、访问、治理等场景

### API 设计

- **RESTful API**：标准的 REST 接口设计
- **Swagger 文档**：完整的 API 文档和测试界面
- **JWT 认证**：基于 JWT Token 的身份认证
- **权限控制**：细粒度的资源访问控制

## 性能指标

### 系统性能要求

- **API 响应时间**：99% 的请求在 200ms 内响应
- **查询响应时间**：复杂查询平均响应时间不超过 500ms
- **并发用户数**：支持至少 200 个并发用户
- **数据导入速度**：每秒至少 10,000 条记录

### 可用性要求

- **服务可用率**：99.95% 以上
- **数据存储周期**：至少 3 年
- **备份策略**：定时自动备份关键数据

## 部署和运维

### 部署方式

- **私有化部署**：支持园区数据中心独立部署
- **虚拟化支持**：兼容主流虚拟化平台
- **容器化部署**：基于 Docker Compose 的一键部署

### 监控和告警

- **自监控**：实时监控服务状态和性能指标
- **告警通知**：支持短信和飞书接口推送
- **日志管理**：完整的操作日志和审计记录

## 文件结构

### 核心目录

- **api/**：API 控制器和路由定义
- **service/**：业务逻辑和数据服务
- **service/models/**：数据模型定义
- **service/database/**：数据库连接和配置
- **scripts/**：数据库脚本和工具
- **ai_docs/**：项目文档和设计说明
- **memory-bank/**：项目记忆库和上下文

### 重要文件

- **recommended_views.md**：视图设计详细说明
- **create_views.sql**：视图创建 SQL 脚本
- **model.md**：数据模型设计文档
- **requirements.md**：详细需求规格说明
- **postgrest_rbac_guide.md**：权限管理集成指南

## 使用场景

### 智慧园区数据管理

- 会员数据管理和分析
- 资产数据监控和维护
- 供应链数据集成和分析
- 园区运行数据实时监控

### 数据治理和合规

- 数据质量持续监控
- 敏感数据脱敏保护
- 数据访问权限控制
- 合规性审计和报告

### 数据服务和共享

- 统一数据 API 服务
- 跨系统数据同步
- 数据订阅和通知
- 数据目录和发现

## 下一步发展

### 短期计划

- 视图系统性能优化
- API 文档完善
- 集成测试和验证

### 中期计划

- 数据导入导出功能
- 高级查询和分析
- 第三方系统集成

### 长期计划

- 智能数据推荐
- 自动化数据治理
- 机器学习集成
- 分布式架构演进

---

_该服务是智慧园区数据底座的核心组件，为园区的数字化运营提供强有力的数据支撑。_
