# 数据底座模型设计

## 更新记录

**最后更新时间**: 2024-01-15  
**更新内容**:

- 完善数据主题库模型，增加分类、域、权限等字段
- 更新访问令牌模型，与已实现的Token管理功能保持一致
- 新增数据脱敏规则模型，支持多种脱敏类型
- 更新流程图节点类型，与流程设计器实现保持一致

## 1. 数据基础库与主题库相关模型

### 1.1 数据基础库模型 (BasicLibrary)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 基础库唯一标识符

- **名称(中文)**: name_zh

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 基础库中文名称

- **名称(英文)**: name_en

  - **类型**: string
  - **约束**: 不可为空，唯一，符合数据库schema命名规范
  - **描述**: 基础库英文名称，用作数据库schema

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 基础库功能和用途描述

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 基础库创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 基础库更新时间

- **状态**: status
  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 基础库状态：active/inactive/archived

### 1.2 数据接口模型 (DataInterface)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 接口唯一标识符

- **基础库ID**: library_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的基础库ID

- **名称(中文)**: name_zh

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 接口中文名称

- **名称(英文)**: name_en

  - **类型**: string
  - **约束**: 不可为空，符合表名命名规范
  - **描述**: 接口英文名称，用作数据库表名

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 接口类型：realtime(实时)/batch(批量)

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 接口功能和用途描述

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 接口创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 接口更新时间

- **状态**: status
  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 接口状态：active/inactive/archived

### 1.3 数据源模型 (DataSource)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 数据源唯一标识符

- **接口ID**: interface_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的接口ID

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 数据源类型：kafka/redis/nats/http/db/hostpath等

- **连接配置**: connection_config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 根据类型不同的连接配置参数

- **参数配置**: params_config

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 接口参数变量配置

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 数据源创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 数据源更新时间

### 1.4 接口字段模型 (InterfaceField)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 字段唯一标识符

- **接口ID**: interface_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的接口ID

- **名称(中文)**: name_zh

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 字段中文名称

- **名称(英文)**: name_en

  - **类型**: string
  - **约束**: 不可为空，符合列名命名规范
  - **描述**: 字段英文名称，用作数据库列名

- **数据类型**: data_type

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 字段数据类型

- **是否主键**: is_primary_key

  - **类型**: boolean
  - **约束**: 不可为空，默认false
  - **描述**: 是否为主键

- **是否可为空**: is_nullable

  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 是否允许为空

- **默认值**: default_value

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 字段默认值

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 字段描述

- **排序号**: order_num
  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 字段在表中的顺序

### 1.5 数据清洗规则模型 (CleansingRule)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 规则唯一标识符

- **接口ID**: interface_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的接口ID

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 规则类型：missing_data/default_value/delete_row/delete_column/normalize_type/format_data/rename_column/delete_empty/delete_illegal/delete_duplicate

- **配置**: config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 规则的具体配置参数

- **排序**: order_num

  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 规则执行顺序

- **是否启用**: is_enabled
  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 规则是否启用

### 1.6 数据主题库模型 (ThematicLibrary)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 主题库唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 主题库名称

- **编码**: code

  - **类型**: string
  - **约束**: 不可为空，唯一，符合数据库schema命名规范
  - **描述**: 主题库编码，用作数据库schema

- **主题分类**: category

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 主题分类：business(业务主题)/technical(技术主题)/analysis(分析主题)/report(报表主题)

- **数据域**: domain

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 数据域：user(用户域)/order(订单域)/product(商品域)/finance(财务域)/marketing(营销域)

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 主题库功能和用途描述

- **标签**: tags

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 主题库标签数组

- **数据源库**: source_libraries

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 关联的基础库ID数组

- **主题表配置**: tables

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 主题表定义和配置

- **发布状态**: publish_status

  - **类型**: enum
  - **约束**: 不可为空，默认'draft'
  - **描述**: 发布状态：draft(草稿)/published(已发布)/archived(已归档)

- **版本号**: version

  - **类型**: string
  - **约束**: 不可为空，默认'1.0.0'
  - **描述**: 主题库版本号

- **访问权限**: access_level

  - **类型**: enum
  - **约束**: 不可为空，默认'internal'
  - **描述**: 访问权限：public(公开)/internal(内部)/private(私有)

- **授权用户**: authorized_users

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 授权用户ID数组

- **授权角色**: authorized_roles

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 授权角色ID数组

- **更新频率**: update_frequency

  - **类型**: enum
  - **约束**: 不可为空，默认'daily'
  - **描述**: 更新频率：realtime(实时)/hourly(每小时)/daily(每日)/weekly(每周)/monthly(每月)

- **数据保留期**: retention_period

  - **类型**: integer
  - **约束**: 不可为空，默认365
  - **描述**: 数据保留期（天）

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 主题库创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 主题库更新时间

- **状态**: status
  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 主题库状态：active/inactive/archived

### 1.7 主题库接口模型 (ThematicInterface)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 主题库接口唯一标识符

- **主题库ID**: library_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的主题库ID

- **名称(中文)**: name_zh

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 接口中文名称

- **名称(英文)**: name_en

  - **类型**: string
  - **约束**: 不可为空，符合表名命名规范
  - **描述**: 接口英文名称，用作数据库表名

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 接口类型：realtime(实时)/http(HTTP)

- **配置**: config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 根据类型不同的接口配置

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 接口功能和用途描述

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 接口创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 接口更新时间

- **状态**: status
  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 接口状态：active/inactive/archived

### 1.8 数据流程图模型 (DataFlowGraph)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 流程图唯一标识符

- **主题库接口ID**: thematic_interface_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的主题库接口ID

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 流程图名称

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 流程图描述

- **定义**: definition

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 有向无环图的节点和边定义

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 流程图创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 流程图更新时间

- **状态**: status
  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 流程图状态：draft/active/inactive

### 1.9 流程图节点模型 (FlowNode)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 节点唯一标识符

- **流程图ID**: flow_graph_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的流程图ID

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 节点类型：datasource(数据源)/api(API接口)/file(文件)/filter(数据过滤)/transform(数据转换)/aggregate(数据聚合)/output(数据输出)

- **配置**: config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 节点的配置参数

- **位置X**: position_x

  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 节点在图中的X坐标

- **位置Y**: position_y

  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 节点在图中的Y坐标

- **名称**: name
  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 节点名称

## 2. 数据治理相关模型

### 2.1 数据脱敏规则模型 (DataMaskingRule)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 脱敏规则唯一标识符

- **规则名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 脱敏规则名称

- **数据源**: data_source

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 数据源名称

- **数据表**: data_table

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 数据表名称

- **字段名**: field_name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 需要脱敏的字段名

- **字段类型**: field_type

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 字段数据类型

- **脱敏类型**: masking_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 脱敏类型：mask(掩码)/replace(替换)/encrypt(加密)/pseudonymize(假名化)

- **脱敏配置**: masking_config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 脱敏配置参数，根据脱敏类型不同而不同

- **是否启用**: is_enabled

  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 规则是否启用

- **创建者ID**: creator_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 规则创建者用户ID

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 规则创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 规则更新时间

### 2.2 元数据模型 (Metadata)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 元数据唯一标识符

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 元数据类型：technical/business/management

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 元数据名称

- **内容**: content

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 元数据详细内容

- **关联对象ID**: related_object_id

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 关联对象的ID

- **关联对象类型**: related_object_type

  - **类型**: enum
  - **约束**: 可为空
  - **描述**: 关联对象类型：basic_library/data_interface/thematic_library等

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 元数据创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 元数据更新时间

### 2.2 数据质量规则模型 (QualityRule)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 质量规则唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 规则名称

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 规则类型：completeness/standardization/consistency/accuracy/uniqueness/timeliness

- **配置**: config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 规则的配置参数

- **关联对象ID**: related_object_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 关联对象的ID

- **关联对象类型**: related_object_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 关联对象类型：interface/thematic_interface

- **是否启用**: is_enabled
  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 规则是否启用

### 2.3 数据脱敏规则模型 (DesensitizationRule)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 脱敏规则唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 规则名称

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 规则类型：mask/replace/encrypt/pseudonymize

- **配置**: config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 规则的配置参数

- **字段ID**: field_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的字段ID

- **是否启用**: is_enabled
  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 规则是否启用

## 3. 访问控制相关模型

### 3.1 用户模型 (User)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 用户唯一标识符

- **用户名**: username

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 用户登录名

- **密码哈希**: password_hash

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 加密后的密码

- **姓名**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 用户真实姓名

- **电子邮件**: email

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 用户电子邮件

- **手机号**: phone

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 用户手机号

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 用户状态：active/inactive/locked

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 用户创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 用户更新时间

- **最后登录时间**: last_login_at
  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 用户最后登录时间

### 3.2 角色模型 (Role)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 角色唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 角色名称

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 角色描述

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 角色创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 角色更新时间

### 3.3 权限模型 (Permission)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 权限唯一标识符

- **代码**: code

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 权限唯一代码

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 权限名称

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 权限描述

- **资源类型**: resource_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 资源类型：basic_library/thematic_library/interface/system等

- **操作类型**: action_type
  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 操作类型：read/write/execute/admin等

### 3.4 访问令牌模型 (AccessToken)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 令牌唯一标识符

- **令牌名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 令牌名称/用途

- **令牌值**: token

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: API访问令牌值

- **应用名称**: app_name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 关联的应用名称

- **过期时间**: expires_at

  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 令牌过期时间，null表示永不过期

- **权限配置**: permissions

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 令牌权限配置，包含主题库和接口权限

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 令牌状态：active(活跃)/revoked(已撤销)/expired(已过期)

- **最后使用时间**: last_used_at

  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 令牌最后使用时间

- **使用次数**: usage_count

  - **类型**: integer
  - **约束**: 不可为空，默认0
  - **描述**: 令牌使用次数

- **创建者ID**: creator_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 令牌创建者用户ID

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 令牌创建时间

- **更新时间**: updated_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 令牌更新时间

### 3.5 令牌权限模型 (TokenPermission)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 令牌权限唯一标识符

- **令牌ID**: token_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的令牌ID

- **资源ID**: resource_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 资源ID

- **资源类型**: resource_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 资源类型：thematic_library/interface等

- **操作类型**: action_type
  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 操作类型：read/write

## 4. 数据运维相关模型

### 4.1 系统日志模型 (SystemLog)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 日志唯一标识符

- **操作类型**: operation_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 操作类型：create/update/delete/query等

- **对象类型**: object_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 对象类型：basic_library/thematic_library/interface/user等

- **对象ID**: object_id

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 操作对象的ID

- **操作者ID**: operator_id

  - **类型**: string
  - **约束**: 外键，可为空
  - **描述**: 操作用户ID

- **操作者IP**: operator_ip

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 操作用户IP地址

- **操作内容**: operation_content

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 操作的详细内容

- **操作时间**: operation_time

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 操作时间

- **操作结果**: operation_result
  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 操作结果：success/failure

### 4.2 备份配置模型 (BackupConfig)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 备份配置唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 备份配置名称

- **类型**: type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 备份类型：full/incremental

- **对象类型**: object_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 备份对象类型：thematic_library/basic_library

- **对象ID**: object_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 备份对象ID

- **策略**: strategy

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 备份策略配置（周期、保留策略等）

- **存储位置**: storage_location

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 备份存储位置

- **是否启用**: is_enabled

  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 是否启用该备份配置

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 更新时间

### 4.3 备份记录模型 (BackupRecord)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 备份记录唯一标识符

- **备份配置ID**: backup_config_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的备份配置ID

- **备份开始时间**: start_time

  - **类型**: datetime
  - **约束**: 不可为空
  - **描述**: 备份开始时间

- **备份结束时间**: end_time

  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 备份结束时间

- **备份大小**: backup_size

  - **类型**: bigint
  - **约束**: 可为空
  - **描述**: 备份文件大小(字节)

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 备份状态：in_progress/success/failure

- **备份文件路径**: file_path

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 备份文件路径

- **错误信息**: error_message
  - **类型**: string
  - **约束**: 可为空
  - **描述**: 备份失败时的错误信息

## 5. 数据共享服务相关模型

### 5.1 API接入应用模型 (ApiApplication)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 应用唯一标识符

- **名称**: name

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 应用名称

- **应用密钥**: app_key

  - **类型**: string
  - **约束**: 不可为空，唯一
  - **描述**: 应用ID密钥

- **应用密钥哈希**: app_secret_hash

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 加密的应用密钥

- **描述**: description

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 应用描述

- **联系人**: contact_person

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 应用负责人

- **联系邮箱**: contact_email

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 联系邮箱

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 应用状态：active/inactive

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 更新时间

### 5.2 API调用限制模型 (ApiRateLimit)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 限制规则唯一标识符

- **应用ID**: application_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 关联的应用ID

- **接口路径**: api_path

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: API接口路径，支持通配符

- **时间窗口(秒)**: time_window

  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 限流时间窗口，单位秒

- **最大请求数**: max_requests

  - **类型**: integer
  - **约束**: 不可为空
  - **描述**: 时间窗口内最大请求数

- **是否启用**: is_enabled
  - **类型**: boolean
  - **约束**: 不可为空，默认true
  - **描述**: 是否启用该限制规则

### 5.3 数据订阅模型 (DataSubscription)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 订阅唯一标识符

- **订阅者ID**: subscriber_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 订阅者ID（用户ID或应用ID）

- **订阅者类型**: subscriber_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 订阅者类型：user/application

- **资源ID**: resource_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 订阅资源ID

- **资源类型**: resource_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 资源类型：thematic_interface/basic_interface

- **通知方式**: notification_method

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 通知方式：webhook/message_queue/email

- **通知配置**: notification_config

  - **类型**: jsonb
  - **约束**: 不可为空
  - **描述**: 通知配置（URL、队列名等）

- **过滤条件**: filter_condition

  - **类型**: jsonb
  - **约束**: 可为空
  - **描述**: 数据变更过滤条件

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空，默认'active'
  - **描述**: 订阅状态：active/paused/terminated

- **创建时间**: created_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 创建时间

- **更新时间**: updated_at
  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 更新时间

### 5.4 数据使用申请模型 (DataAccessRequest)

- **ID**: id

  - **类型**: string
  - **约束**: 主键，不可为空，唯一
  - **描述**: 申请唯一标识符

- **申请者ID**: requester_id

  - **类型**: string
  - **约束**: 外键，不可为空
  - **描述**: 申请用户ID

- **资源ID**: resource_id

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 申请访问的资源ID

- **资源类型**: resource_type

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 资源类型：thematic_library/basic_library/interface

- **申请原因**: request_reason

  - **类型**: string
  - **约束**: 不可为空
  - **描述**: 申请访问原因

- **访问权限**: access_permission

  - **类型**: enum
  - **约束**: 不可为空
  - **描述**: 申请的权限类型：read/write

- **有效期**: valid_until

  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 访问权限有效期

- **状态**: status

  - **类型**: enum
  - **约束**: 不可为空，默认'pending'
  - **描述**: 申请状态：pending/approved/rejected/expired

- **审批意见**: approval_comment

  - **类型**: string
  - **约束**: 可为空
  - **描述**: 审批意见

- **审批者ID**: approver_id

  - **类型**: string
  - **约束**: 外键，可为空
  - **描述**: 审批人ID

- **申请时间**: requested_at

  - **类型**: datetime
  - **约束**: 不可为空，默认当前时间
  - **描述**: 申请提交时间

- **审批时间**: approved_at
  - **类型**: datetime
  - **约束**: 可为空
  - **描述**: 审批时间
