# Golang 分层架构代码生成器

## 概述

基于给定 Entity，自动生成符合 DDD 与 Clean Architecture 的四层结构：Model（Entity）、Repository、Service、Handler。内置统一的错误结构、日志、响应规范。

## 架构设计原则

### 1. 分层职责

| 层级              | 作用说明                                     |
| ----------------- | -------------------------------------------- |
| **Entity 层**     | 领域模型定义，仅包含业务实体、查询结构       |
| **Repository 层** | 数据访问层，仅操作 Entity，严禁操作 DTO      |
| **Service 层**    | 业务逻辑层，负责 DTO ↔ Entity 转换与业务逻辑 |
| **Handler 层**    | 接口层，负责参数绑定、响应输出，仅操作 DTO   |

### 2. 数据流向

```
Handler (dto) → Service (dto→entity) → Repository (entity) → Database
```

### 3. 核心规范

- 模板规范：
  - 严格按生成模板输出，禁止手工破坏结构

---

## 代码生成要求

### 请生成符合以下规范的 entity 层代码

- 使用 GORM 标签定义 MySQL 字段属性
- 特殊字段需注明：
  - 唯一索引字段添加`uniqueIndex`
  - 枚举类型使用自定义类型
  - 地理坐标使用 s2.Point
  - JSON 字段使用 datatypes.JSON
  - 必须包含对应的 Query 结构体

---

### 请生成符合以下规范的 dto 层代码

- 包含 Create/Delete/Update/Filter/Response 四种结构体
- 字段要求：
  - 时间字段使用 int64 秒级时间戳
  - 地理坐标转换为经纬度结构体
  - JSON 字段定义具体结构
  - `Create`：必填校验
  - `Update`：指针类型，支持部分更新
  - `Filter`：支持模糊、精确匹配，含分页
  - JSON 字段 camelCase 风格
- 必须实现以下方法：
  - `ConvertToQuery`
  - `ConvertToEntity`
  - `ConvertToResponse`

---

### 请生成符合以下规范的 repository 层代码

- 基础方法：
  - GetByField(field, value)
  - ExistsByField(field, value)
  - Create/Update/Delete/List
- 特殊处理：
  - 地理搜索实现 ST_Distance_Sphere
  - JSON 字段使用->>操作符查询
- 错误处理：
  - 记录完整错误日志
  - 返回 errx.Error 类型`

---

### 请生成符合以下规范的 Service 层代码

- 必须包含：
  - 唯一性校验
  - 状态机校验
  - DTO/Entity 转换
- 特殊字段处理：
  - 可空唯一字段的空值处理
  - 地理坐标转换
    -JSON 字段验证

---

### 请生成符合以下规范 Handler 层代码

- 框架：
  - 必须使用 Gin
  - 生成 Swagger 注释
- 参数：
  - 查询参数用 DTO Filter
  - 请求体用 DTO Request
- 错误响应：
  - 全部使用 `response.Error(c, err)`
- 字段命名：
  - 查询参数小驼峰
  - 路径参数小写
  - 请求体 JSON 小驼峰

---

## 完整代码模板

### 使用示例 Entity

```go
type Demo struct {
 gorm.Model

 // 基础信息
 Name        string  `gorm:"type:varchar(100);not null;comment:名称"`
 Description *string `gorm:"type:text;default:null;comment:描述"`

 // 数值信息
 SortOrder int     `gorm:"type:int;default:0;comment:排序权重"`
 Price     float64 `gorm:"type:decimal(10,2);default:0.00;comment:价格"`
 Rating    float32 `gorm:"type:float;default:0.0;comment:评分"`

 // 状态信息
 IsActive bool `gorm:"type:tinyint(1);default:1;comment:是否激活"`

 // 枚举字段 - 状态类型
 Status DemoStatusType `gorm:"type:enum('pending','approved','rejected');default:'pending';comment:状态类型"`

 // 时间信息
 ExpireTime *time.Time `gorm:"type:timestamp;default:null;comment:过期时间"`

 // 联系信息
 Phone *string `gorm:"type:varchar(20);uniqueIndex;default:null;comment:手机号"`

 // 地理信息 (WGS84坐标系)
 Location *s2.Point `gorm:"type:point;comment:地理坐标;srid:4326"`

 // 媒体信息
 AvatarURL string `gorm:"type:varchar(255);default:'';comment:头像URL"`

 // 分类信息
 Tags datatypes.JSON `gorm:"type:json;comment:标签"`

 // 扩展信息
 Attachments datatypes.JSON `gorm:"type:json;comment:附件URL列表"`
 Metadata    datatypes.JSON `gorm:"type:json;comment:元数据"`
}
```

---

### 依赖的公共文件

#### pkg/constant/constant.go (公共文件)

```go
package constant

const RequestIDKey = "X-Request-ID"

const (
 Success             = 200
 BadRequest          = 400
 Unauthorized        = 401
 Forbidden           = 403
 NotFound            = 404
 Conflict            = 409
 InternalServerError = 500
)

const (
 MsgSuccess           = "请求成功"
 MsgInvalidInput      = "请求参数不合法"
 MsgUnauthorized      = "未授权访问"
 MsgForbidden         = "没有权限"
 MsgNotFound          = "资源不存在"
 MsgConflict          = "资源冲突"
 MsgInternalServerErr = "服务器内部错误"
)
```

#### pkg/errx/errx.go (公共文件)

```go
package errx

import (
 "fmt"
)

// Error 结构化错误类型
type Error struct {
 Code    int    `json:"code"`
 Message string `json:"message"`
 Err     error  `json:"-"`
}

func (e *Error) Error() string {
 if e.Err != nil {
  return fmt.Sprintf("%s: %v", e.Message, e.Err)
 }
 return e.Message
}

func New(code int, message string, err error) *Error {
 return &Error{
  Code:    code,
  Message: message,
  Err:     err,
 }
}

```

---

#### internal/model/entity/common.go (公共文件)

```go
package entity

// Pagination 分页参数（通用）
type Pagination struct {
    PageNumber int
    PageSize   int
}
```

#### internal/model/dto/common.go (公共文件)

```go
package dto

import (
 "encoding/json"

 "gorm.io/datatypes"
)

// PaginationRequest 分页请求参数（通用）
type PaginationRequest struct {
 PageNumber int `form:"pageNumber" binding:"omitempty,min=1" default:"1"`        // 当前页码
 PageSize   int `form:"pageSize" binding:"omitempty,min=1,max=100" default:"10"` // 每页大小
}

// PaginationResponse 分页响应结构（通用）
type PaginationResponse struct {
 Records    interface{} `json:"records"`    // 数据列表
 TotalCount int64       `json:"totalCount"` // 总记录数
}

// Response 统一响应结构（通用）
type Response struct {
 Code    int         `json:"code"`
 Message string      `json:"message"`
 Data    interface{} `json:"data,omitempty"`
}

// ToJSON 将结构体转换为datatypes.JSON
func ToJSON(v interface{}) (datatypes.JSON, error) {
 data, err := json.Marshal(v)
 if err != nil {
  return nil, err
 }
 return datatypes.JSON(data), nil
}

// FromJSON 从datatypes.JSON解析到结构体
func FromJSON(data datatypes.JSON, v interface{}) error {
 if len(data) == 0 {
  return nil
 }
 if err := json.Unmarshal(data, v); err != nil {
  return err
 }
 return nil
}
```

### 需要生成 6 个文件

#### 1. internal/model/entity/demo.go

```go
package entity

import (
 "time"

 "github.com/golang/geo/s2"
 "github.com/lib/pq"
 "gorm.io/datatypes"
 "gorm.io/gorm"
)

// DemoStatusType 状态类型枚举
type DemoStatusType string

const (
 DemoStatusPending  DemoStatusType = "pending"
 DemoStatusApproved DemoStatusType = "approved"
 DemoStatusRejected DemoStatusType = "rejected"
)

// Demo 示例模型
type Demo struct {
 gorm.Model

 // 基础信息
 Name        string  `gorm:"type:varchar(100);not null;comment:名称"`
 Description *string `gorm:"type:text;default:null;comment:描述"`

 // 数值信息
 SortOrder int     `gorm:"type:int;default:0;comment:排序权重"`
 Price     float64 `gorm:"type:decimal(10,2);default:0.00;comment:价格"`
 Rating    float32 `gorm:"type:float;default:0.0;comment:评分"`

 // 状态信息
 IsActive bool `gorm:"type:tinyint(1);default:1;comment:是否激活"`

 // 枚举字段 - 状态类型
 Status DemoStatusType `gorm:"type:enum('pending','approved','rejected');default:'pending';comment:状态类型"`

 // 时间信息
 ExpireTime *time.Time `gorm:"type:timestamp;default:null;comment:过期时间"`

 // 联系信息
 Phone *string `gorm:"type:varchar(20);uniqueIndex;default:null;comment:手机号"`

 // 地理信息 (WGS84坐标系)
 Location *s2.Point `gorm:"type:point;comment:地理坐标;srid:4326"`

 // 媒体信息
 AvatarURL string `gorm:"type:varchar(255);default:'';comment:头像URL"`

 // 分类信息
 Tags datatypes.JSON `gorm:"type:json;comment:标签"`

 // 扩展信息
 Attachments datatypes.JSON `gorm:"type:json;comment:附件URL列表"`
 Metadata    datatypes.JSON `gorm:"type:json;comment:元数据"`
}

// DemoQuery 查询条件结构体
type DemoQuery struct {
 Name       string          // 名称模糊匹配
 Phone      *string         // 手机号精确匹配
 Status     *DemoStatusType // 状态精确匹配
 IsActive   *bool           // 激活状态
 MinPrice   *float64        // 最低价格
 MaxPrice   *float64        // 最高价格
 MinRating  *float32        // 最低评分
 BeforeTime *time.Time      // 过期时间之前
 AfterTime  *time.Time      // 过期时间之后
 Tags       pq.StringArray  // 标签包含
 Location   *s2.Point       // 地理坐标附近
 Distance   float64         // 距离范围(米)
 Pagination                 // 分页参数
}

```

---

#### 2. internal/model/dto/demo_dto.go

```go
package dto

import (
 "time"

 "rbac-admin/internal/model/entity"
 "rbac-admin/pkg/constant"
 "rbac-admin/pkg/errx"

 "github.com/golang/geo/s2"
 "github.com/lib/pq"
 "gorm.io/datatypes"
)

// DemoLocation 地理坐标结构
type DemoLocation struct {
 Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
 Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
}

// Validate 坐标校验方法
func (l *DemoLocation) Validate() error {
 if l.Longitude < -180 || l.Longitude > 180 {
  return errx.New(constant.BadRequest, "经度范围无效", nil)
 }
 if l.Latitude < -90 || l.Latitude > 90 {
  return errx.New(constant.BadRequest, "纬度范围无效", nil)
 }
 return nil
}

// DemoTag 标签结构
type DemoTag struct {
 Key   string `json:"key" binding:"required,max=50"`
 Value string `json:"value" binding:"required,max=100"`
}

// DemoCreateRequest 创建请求
type DemoCreateRequest struct {
 Name        string         `json:"name" binding:"required,min=1,max=100"`
 Description *string        `json:"description,omitempty" binding:"omitempty,max=500"`
 SortOrder   int            `json:"sortOrder" binding:"min=0"`
 Price       float64        `json:"price" binding:"min=0"`
 Rating      float32        `json:"rating" binding:"min=0,max=5"`
 IsActive    bool           `json:"isActive"`
 Status      string         `json:"status" binding:"required,oneof=pending approved rejected"`
 ExpireTime  *int64         `json:"expireTime,omitempty"`
 Phone       *string        `json:"phone,omitempty" binding:"omitempty,e164"`
 Location    *DemoLocation  `json:"location,omitempty"`
 AvatarURL   string         `json:"avatarUrl" binding:"omitempty,url,max=255"`
 Tags        []DemoTag      `json:"tags" binding:"dive"`
 Attachments []string       `json:"attachments" binding:"dive,url"`
 Metadata    datatypes.JSON `json:"metadata" binding:"required"`
}

// DemoUpdateRequest 更新请求
type DemoUpdateRequest struct {
 Name        *string         `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
 Description *string         `json:"description,omitempty" binding:"omitempty,max=500"`
 SortOrder   *int            `json:"sortOrder,omitempty" binding:"omitempty,min=0"`
 Price       *float64        `json:"price,omitempty" binding:"omitempty,min=0"`
 Rating      *float32        `json:"rating,omitempty" binding:"omitempty,min=0,max=5"`
 IsActive    *bool           `json:"isActive,omitempty"`
 Status      *string         `json:"status,omitempty" binding:"omitempty,oneof=pending approved rejected"`
 ExpireTime  *int64          `json:"expireTime,omitempty"`
 Phone       *string         `json:"phone,omitempty" binding:"omitempty,e164"`
 Location    *DemoLocation   `json:"location,omitempty"`
 AvatarURL   *string         `json:"avatarUrl,omitempty" binding:"omitempty,url,max=255"`
 Tags        *[]DemoTag      `json:"tags,omitempty" binding:"omitempty,dive"`
 Attachments *[]string       `json:"attachments,omitempty" binding:"omitempty,dive,url"`
 Metadata    *datatypes.JSON `json:"metadata,omitempty"`
}

// DemoDeleteRequest 批量删除请求
type DemoDeleteRequest struct {
 IDs []uint `json:"ids" binding:"required,min=1"`
}

// DemoFilter 查询过滤器
type DemoFilter struct {
 Name              string           `form:"name,omitempty"`       // 名称模糊匹配
 Phone             *string          `form:"phone,omitempty"`      // 手机号精确匹配
 Status            *string          `form:"status,omitempty"`     // 状态精确匹配
 IsActive          *bool            `form:"isActive,omitempty"`   // 激活状态
 MinPrice          *float64         `form:"minPrice,omitempty"`   // 最低价格
 MaxPrice          *float64         `form:"maxPrice,omitempty"`   // 最高价格
 MinRating         *float32         `form:"minRating,omitempty"`  // 最低评分
 BeforeTime        *int64           `form:"beforeTime,omitempty"` // 过期时间之前(秒级时间戳)
 AfterTime         *int64           `form:"afterTime,omitempty"`  // 过期时间之后(秒级时间戳)
 Longitude         *float64         `form:"lng,omitempty"`        // 中心经度
 Latitude          *float64         `form:"lat,omitempty"`        // 中心纬度
 Radius            *float64         `form:"radius,omitempty"`     // 搜索半径(米)
 TagKey            *string          `form:"tagKey,omitempty"`     // 标签键
 TagValue          *string          `form:"tagValue,omitempty"`   // 标签值
 PaginationRequest `form:",inline"` // 分页参数
}

// DemoResponse 响应结构
type DemoResponse struct {
 ID          uint           `json:"id"`
 Name        string         `json:"name"`
 Description *string        `json:"description,omitempty"`
 SortOrder   int            `json:"sortOrder"`
 Price       float64        `json:"price"`
 Rating      float32        `json:"rating"`
 IsActive    bool           `json:"isActive"`
 Status      string         `json:"status"`
 ExpireTime  *int64         `json:"expireTime,omitempty"` // 秒级时间戳
 Phone       *string        `json:"phone,omitempty"`
 Location    *DemoLocation  `json:"location,omitempty"`
 AvatarURL   string         `json:"avatarUrl"`
 Tags        []DemoTag      `json:"tags"`
 Attachments []string       `json:"attachments"`
 Metadata    datatypes.JSON `json:"metadata"`
 CreatedAt   int64          `json:"createdAt"` // 秒级时间戳
 UpdatedAt   int64          `json:"updatedAt"` // 秒级时间戳
}

// ConvertToEntity 将创建请求转换为实体
func (req *DemoCreateRequest) ConvertToEntity() (*entity.Demo, error) {
 // 坐标校验
 if req.Location != nil {
  if err := req.Location.Validate(); err != nil {
   return nil, err
  }
 }

 demo := &entity.Demo{
  Name:        req.Name,
  Description: req.Description,
  SortOrder:   req.SortOrder,
  Price:       req.Price,
  Rating:      req.Rating,
  IsActive:    req.IsActive,
  Status:      entity.DemoStatusType(req.Status),
  Phone:       req.Phone,
  AvatarURL:   req.AvatarURL,
  Metadata:    req.Metadata,
 }

 // 时间字段转换
 if req.ExpireTime != nil && *req.ExpireTime > 0 {
  t := time.Unix(*req.ExpireTime, 0)
  demo.ExpireTime = &t
 }

 // 地理坐标转换
 if req.Location != nil {
  latLng := s2.LatLngFromDegrees(req.Location.Latitude, req.Location.Longitude)
  point := s2.PointFromLatLng(latLng)
  demo.Location = &point
 }

 // Tags转换
 tagsJSON, err := ToJSON(req.Tags)
 if err != nil {
  return nil, errx.New(constant.BadRequest, "标签转换失败", err)
 }
 demo.Tags = tagsJSON

 // Attachments转换
 attachmentsJSON, err := ToJSON(req.Attachments)
 if err != nil {
  return nil, errx.New(constant.BadRequest, "附件转换失败", err)
 }
 demo.Attachments = attachmentsJSON

 return demo, nil
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *DemoFilter) ConvertToQuery() entity.DemoQuery {
 query := entity.DemoQuery{
  Name:      f.Name,
  Phone:     f.Phone,
  IsActive:  f.IsActive,
  MinPrice:  f.MinPrice,
  MaxPrice:  f.MaxPrice,
  MinRating: f.MinRating,
  Pagination: entity.Pagination{
   PageNumber: f.GetPageNumber(),
   PageSize:   f.GetPageSize(),
  },
 }

 // 状态转换
 if f.Status != nil {
  status := entity.DemoStatusType(*f.Status)
  query.Status = &status
 }

 // 时间字段转换
 if f.BeforeTime != nil && *f.BeforeTime > 0 {
  t := time.Unix(*f.BeforeTime, 0)
  query.BeforeTime = &t
 }
 if f.AfterTime != nil && *f.AfterTime > 0 {
  t := time.Unix(*f.AfterTime, 0)
  query.AfterTime = &t
 }

 // 地理坐标转换
 if f.Longitude != nil && f.Latitude != nil && f.Radius != nil {
  latLng := s2.LatLngFromDegrees(*f.Latitude, *f.Longitude)
  point := s2.PointFromLatLng(latLng)
  query.Location = &point
  query.Distance = *f.Radius
 }

 // 标签查询
 if f.TagKey != nil && f.TagValue != nil {
  query.Tags = pq.StringArray{*f.TagKey + ":" + *f.TagValue}
 }

 return query
}

// ConvertToDemoResponse 将实体转换为响应
func ConvertToDemoResponse(demo *entity.Demo) (*DemoResponse, error) {
 resp := &DemoResponse{
  ID:          demo.ID,
  Name:        demo.Name,
  Description: demo.Description,
  SortOrder:   demo.SortOrder,
  Price:       demo.Price,
  Rating:      demo.Rating,
  IsActive:    demo.IsActive,
  Status:      string(demo.Status),
  Phone:       demo.Phone,
  AvatarURL:   demo.AvatarURL,
  Metadata:    demo.Metadata,
  CreatedAt:   demo.CreatedAt.Unix(),
  UpdatedAt:   demo.UpdatedAt.Unix(),
 }

 // 时间字段转换
 if demo.ExpireTime != nil {
  expireUnix := demo.ExpireTime.Unix()
  if expireUnix > 0 {
   resp.ExpireTime = &expireUnix
  }
 }

 // 地理坐标转换
 if demo.Location != nil {
  latLng := s2.LatLngFromPoint(*demo.Location)
  resp.Location = &DemoLocation{
   Longitude: latLng.Lng.Degrees(),
   Latitude:  latLng.Lat.Degrees(),
  }
 }

 // Tags转换
 var tags []DemoTag
 if err := FromJSON(demo.Tags, &tags); err == nil {
  resp.Tags = tags
 }

 // Attachments转换
 var attachments []string
 if err := FromJSON(demo.Attachments, &attachments); err == nil {
  resp.Attachments = attachments
 }

 return resp, nil
}

// GetPageNumber 安全获取页码
func (f *DemoFilter) GetPageNumber() int {
 if f.PageNumber <= 0 {
  return 1
 }
 return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *DemoFilter) GetPageSize() int {
 if f.PageSize <= 0 {
  return 10
 }
 return f.PageSize
}

```

---

#### 3. internal/repository/demo_repository.go

```go
package repository

import (
 "context"
 "fmt"
 "strings"

 "rbac-admin/internal/model/entity"
 "rbac-admin/pkg/constant"
 "rbac-admin/pkg/contextx"
 "rbac-admin/pkg/errx"

 "github.com/golang/geo/s2"
 "gorm.io/gorm"
)

type DemoRepository struct {
 Db *gorm.DB
}

// GetByField 通用字段查询方法
func (r *DemoRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.Demo, error) {
 logger := contextx.GetLoggerFromContext(ctx)
 var demo entity.Demo

 err := r.Db.WithContext(ctx).
  Where(fmt.Sprintf("%s = ?", field), value).
  First(&demo).Error

 switch {
 case err == gorm.ErrRecordNotFound:
  return nil, errx.New(constant.NotFound, "记录不存在", nil)
 case err != nil:
  logger.Errorf("查询失败[%s]: %v", field, err)
  return nil, errx.New(constant.InternalServerError, "查询失败", err)
 }

 return &demo, nil
}

// ExistsByField 通用存在性检查
func (r *DemoRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
 logger := contextx.GetLoggerFromContext(ctx)
 var count int64

 err := r.Db.WithContext(ctx).
  Model(&entity.Demo{}).
  Where(fmt.Sprintf("%s = ?", field), value).
  Count(&count).Error

 if err != nil {
  logger.Errorf("存在性检查失败[%s]: %v", field, err)
  return false, errx.New(constant.InternalServerError, "检查失败", err)
 }

 return count > 0, nil
}

// Create 创建记录
func (r *DemoRepository) Create(ctx context.Context, demo *entity.Demo) error {
 logger := contextx.GetLoggerFromContext(ctx)
 if demo == nil {
  return errx.New(constant.BadRequest, "参数不能为空", nil)
 }

 err := r.Db.WithContext(ctx).Create(demo).Error
 if err != nil {
  logger.Errorf("创建失败: %v", err)
  return errx.New(constant.InternalServerError, "创建失败", err)
 }

 return nil
}

// Update 更新记录
func (r *DemoRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
 logger := contextx.GetLoggerFromContext(ctx)
 result := r.Db.WithContext(ctx).
  Model(&entity.Demo{}).
  Where("id = ?", id).
  Updates(updates)

 if result.Error != nil {
  logger.Errorf("更新失败: %v", result.Error)
  return errx.New(constant.InternalServerError, "更新失败", result.Error)
 }
 if result.RowsAffected == 0 {
  return errx.New(constant.NotFound, "记录不存在", nil)
 }

 return nil
}

// Delete 删除记录
func (r *DemoRepository) Delete(ctx context.Context, id uint) error {
 logger := contextx.GetLoggerFromContext(ctx)
 result := r.Db.WithContext(ctx).Delete(&entity.Demo{}, id)

 if result.Error != nil {
  logger.Errorf("删除失败: %v", result.Error)
  return errx.New(constant.InternalServerError, "删除失败", result.Error)
 }
 if result.RowsAffected == 0 {
  return errx.New(constant.NotFound, "记录不存在", nil)
 }

 return nil
}

// BatchDelete 批量删除
func (r *DemoRepository) BatchDelete(ctx context.Context, ids []uint) error {
 logger := contextx.GetLoggerFromContext(ctx)
 result := r.Db.WithContext(ctx).
  Where("id IN (?)", ids).
  Delete(&entity.Demo{})

 if result.Error != nil {
  logger.Errorf("批量删除失败: %v", result.Error)
  return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
 }
 if result.RowsAffected == 0 {
  return errx.New(constant.NotFound, "无有效记录", nil)
 }

 return nil
}

// List 分页查询（支持地理空间搜索）
func (r *DemoRepository) List(ctx context.Context, query entity.DemoQuery) ([]*entity.Demo, int64, error) {
 logger := contextx.GetLoggerFromContext(ctx)
 var demos []*entity.Demo
 var total int64

 db := r.Db.WithContext(ctx).Model(&entity.Demo{})

 // 基础条件
 if query.Name != "" {
  db = db.Where("name LIKE ?", "%"+query.Name+"%")
 }
 if query.Phone != nil {
  db = db.Where("phone = ?", *query.Phone)
 }
 if query.Status != nil {
  db = db.Where("status = ?", *query.Status)
 }
 if query.IsActive != nil {
  db = db.Where("is_active = ?", *query.IsActive)
 }

 // 数值范围
 if query.MinPrice != nil {
  db = db.Where("price >= ?", *query.MinPrice)
 }
 if query.MaxPrice != nil {
  db = db.Where("price <= ?", *query.MaxPrice)
 }
 if query.MinRating != nil {
  db = db.Where("rating >= ?", *query.MinRating)
 }

 // 时间范围
 if query.BeforeTime != nil {
  db = db.Where("expire_time <= ?", *query.BeforeTime)
 }
 if query.AfterTime != nil {
  db = db.Where("expire_time >= ?", *query.AfterTime)
 }

 // 地理空间查询
 if query.Location != nil && query.Distance > 0 {
  db = db.Where(`
   ST_Distance_Sphere(
    location,
    ST_GeomFromText(?)
   ) <= ?`,
   fmt.Sprintf("POINT(%f %f)",
    s2.LatLngFromPoint(*query.Location).Lng.Degrees(),
    s2.LatLngFromPoint(*query.Location).Lat.Degrees(),
   ),
   query.Distance,
  )
 }

 // JSON 数组查询
 if len(query.Tags) > 0 {
  for _, tag := range query.Tags {
   parts := strings.Split(tag, ":")
   if len(parts) == 2 {
    db = db.Where("tags->>'$.key' = ? AND tags->>'$.value' = ?", parts[0], parts[1])
   }
  }
 }

 // 获取总数
 if err := db.Count(&total).Error; err != nil {
  logger.Errorf("统计总数失败: %v", err)
  return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
 }

 // 分页
 if query.PageSize > 0 {
  offset := (query.PageNumber - 1) * query.PageSize
  db = db.Offset(offset).Limit(query.PageSize)
 }

 // 排序
 db = db.Order("sort_order DESC, created_at DESC")

 // 执行查询
 if err := db.Find(&demos).Error; err != nil {
  logger.Errorf("查询列表失败: %v", err)
  return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
 }

 return demos, total, nil
}

// GetByID 通过ID查询（包装通用方法）
func (r *DemoRepository) GetByID(ctx context.Context, id uint) (*entity.Demo, error) {
 return r.GetByField(ctx, "id", id)
}

// ExistsByID 检查ID是否存在（包装通用方法）
func (r *DemoRepository) ExistsByID(ctx context.Context, id uint) (bool, error) {
 return r.ExistsByField(ctx, "id", id)
}

```

---

#### 4. internal/service/demo_service.go

```go
package service

import (
 "context"
 "time"

 "rbac-admin/internal/model/dto"
 "rbac-admin/internal/repository"
 "rbac-admin/pkg/constant"
 "rbac-admin/pkg/errx"

 "github.com/golang/geo/s2"
)

type DemoService struct {
 DemoRepo *repository.DemoRepository
}

// Create 创建记录
func (s *DemoService) Create(ctx context.Context, req dto.DemoCreateRequest) (*dto.DemoResponse, error) {
 // 唯一字段检查
 if exists, err := s.DemoRepo.ExistsByField(ctx, "name", req.Name); err != nil {
  return nil, err
 } else if exists {
  return nil, errx.New(constant.Conflict, "名称已存在", nil)
 }

 if req.Phone != nil {
  if exists, err := s.DemoRepo.ExistsByField(ctx, "phone", *req.Phone); err != nil {
   return nil, err
  } else if exists {
   return nil, errx.New(constant.Conflict, "手机号已存在", nil)
  }
 }

 // DTO转换
 entity, err := req.ConvertToEntity()
 if err != nil {
  return nil, err
 }

 // 创建记录
 if err := s.DemoRepo.Create(ctx, entity); err != nil {
  return nil, err
 }

 return dto.ConvertToDemoResponse(entity)
}

// Get 获取单条记录
func (s *DemoService) Get(ctx context.Context, id uint) (*dto.DemoResponse, error) {
 entity, err := s.DemoRepo.GetByID(ctx, id)
 if err != nil {
  return nil, err
 }
 return dto.ConvertToDemoResponse(entity)
}

// Update 完整更新方法
func (s *DemoService) Update(ctx context.Context, id uint, req dto.DemoUpdateRequest) error {
 updates := make(map[string]interface{})
 current, err := s.DemoRepo.GetByID(ctx, id)
 if err != nil {
  return err
 }

 // 1. 处理唯一字段（名称）
 if req.Name != nil && *req.Name != current.Name {
  if exists, err := s.DemoRepo.ExistsByField(ctx, "name", *req.Name); err != nil {
   return err
  } else if exists {
   return errx.New(constant.Conflict, "名称已存在", nil)
  }
  updates["name"] = *req.Name
 }

 // 2. 处理可空唯一字段（手机号）
 if req.Phone != nil {
  switch {
  case *req.Phone == "": // 清空字段
   updates["phone"] = nil
  case *req.Phone != *current.Phone: // 新值且需要校验
   if exists, err := s.DemoRepo.ExistsByField(ctx, "phone", *req.Phone); err != nil {
    return err
   } else if exists {
    return errx.New(constant.Conflict, "手机号已存在", nil)
   }
   updates["phone"] = *req.Phone
  }
 }

 // 3. 处理地理坐标
 if req.Location != nil {
  if err := req.Location.Validate(); err != nil {
   return err
  }
  latLng := s2.LatLngFromDegrees(req.Location.Latitude, req.Location.Longitude)
  point := s2.PointFromLatLng(latLng)
  updates["location"] = &point
 }

 // 4. 处理JSON字段（标签）
 if req.Tags != nil {
  if len(*req.Tags) == 0 {
   updates["tags"] = nil // 清空标签
  } else {
   tagsJSON, err := dto.ToJSON(*req.Tags)
   if err != nil {
    return errx.New(constant.BadRequest, "标签格式错误", err)
   }
   updates["tags"] = tagsJSON
  }
 }

 // 5. 处理JSON字段（附件）
 if req.Attachments != nil {
  if len(*req.Attachments) == 0 {
   updates["attachments"] = nil
  } else {
   attachmentsJSON, err := dto.ToJSON(*req.Attachments)
   if err != nil {
    return errx.New(constant.BadRequest, "附件格式错误", err)
   }
   updates["attachments"] = attachmentsJSON
  }
 }

 // 6. 处理JSON元数据
 if req.Metadata != nil {
  // 可添加元数据校验逻辑
  updates["metadata"] = *req.Metadata
 }

 // 7. 处理枚举状态
 if req.Status != nil {
  updates["status"] = *req.Status
 }

 // 8. 处理时间字段
 if req.ExpireTime != nil {
  if *req.ExpireTime == 0 {
   updates["expire_time"] = nil // 清空时间
  } else {
   updates["expire_time"] = time.Unix(*req.ExpireTime, 0)
  }
 }

 // 9. 处理普通字段
 if req.Description != nil {
  updates["description"] = *req.Description
 }
 if req.SortOrder != nil {
  updates["sort_order"] = *req.SortOrder
 }
 if req.Price != nil {
  updates["price"] = *req.Price
 }
 if req.Rating != nil {
  updates["rating"] = *req.Rating
 }
 if req.IsActive != nil {
  updates["is_active"] = *req.IsActive
 }
 if req.AvatarURL != nil {
  updates["avatar_url"] = *req.AvatarURL
 }

 // 无有效更新时返回
 if len(updates) == 0 {
  return errx.New(constant.BadRequest, "未提供有效更新字段", nil)
 }

 return s.DemoRepo.Update(ctx, id, updates)
}

// Delete 删除记录
func (s *DemoService) Delete(ctx context.Context, id uint) error {
 return s.DemoRepo.Delete(ctx, id)
}

// List 分页列表
func (s *DemoService) List(ctx context.Context, filter dto.DemoFilter) ([]*dto.DemoResponse, int64, error) {
 query := filter.ConvertToQuery()
 entities, total, err := s.DemoRepo.List(ctx, query)
 if err != nil {
  return nil, 0, err
 }

 responses := make([]*dto.DemoResponse, 0, len(entities))
 for _, e := range entities {
  resp, err := dto.ConvertToDemoResponse(e)
  if err != nil {
   return nil, 0, err
  }
  responses = append(responses, resp)
 }

 return responses, total, nil
}

// BatchDelete 批量删除
func (s *DemoService) BatchDelete(ctx context.Context, req dto.DemoDeleteRequest) error {
 if len(req.IDs) == 0 {
  return errx.New(constant.BadRequest, "ID列表不能为空", nil)
 }
 return s.DemoRepo.BatchDelete(ctx, req.IDs)
}
```

---

#### 5. internal/handler/demo_handler.go

```go
package handler

import (
 "strconv"

 "rbac-admin/internal/handler/response"
 "rbac-admin/internal/model/dto"
 "rbac-admin/internal/service"
 "rbac-admin/pkg/constant"
 "rbac-admin/pkg/contextx"
 "rbac-admin/pkg/errx"
 "rbac-admin/pkg/utils"

 "github.com/gin-gonic/gin"
 "go.uber.org/zap"
)

type DemoHandler struct {
 DemoSer *service.DemoService
}

// Create 创建记录
// @Summary 创建示例
// @Tags 示例管理
// @Accept json
// @Produce json
// @Param request body dto.DemoCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.DemoResponse}
// @Router /api/v1/auth/demos [post]
func (h *DemoHandler) Create(c *gin.Context) {
 var req dto.DemoCreateRequest
 if err := c.ShouldBindJSON(&req); err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.String("req_body", utils.LogStruct(req)))

 resp, err := h.DemoSer.Create(ctx, req)
 if err != nil {
  response.Error(c, err)
  return
 }

 response.Success(c, resp)
}

// Get 获取详情
// @Summary 获取示例详情
// @Tags 示例管理
// @Produce json
// @Param id path int true "示例ID"
// @Success 200 {object} dto.Response{data=dto.DemoResponse}
// @Router /api/v1/auth/demos/{id} [get]
func (h *DemoHandler) Get(c *gin.Context) {
 id, err := strconv.Atoi(c.Param("id"))
 if err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.Int("req_Id", id))

 resp, err := h.DemoSer.Get(ctx, uint(id))
 if err != nil {
  response.Error(c, err)
  return
 }

 response.Success(c, resp)
}

// Update 更新记录
// @Summary 更新示例
// @Tags 示例管理
// @Accept json
// @Produce json
// @Param id path int true "示例ID"
// @Param request body dto.DemoUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/demos/{id} [put]
func (h *DemoHandler) Update(c *gin.Context) {
 id, err := strconv.Atoi(c.Param("id"))
 if err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
  return
 }

 var req dto.DemoUpdateRequest
 if err := c.ShouldBindJSON(&req); err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.Int("req_Id", id), zap.String("req_body", utils.LogStruct(req)))

 if err := h.DemoSer.Update(ctx, uint(id), req); err != nil {
  response.Error(c, err)
  return
 }

 response.Success(c, nil)
}

// Delete 删除记录
// @Summary 删除示例
// @Tags 示例管理
// @Produce json
// @Param id path int true "示例ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/demos/{id} [delete]
func (h *DemoHandler) Delete(c *gin.Context) {
 id, err := strconv.Atoi(c.Param("id"))
 if err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.Int("req_Id", id))

 if err := h.DemoSer.Delete(ctx, uint(id)); err != nil {
  response.Error(c, err)
  return
 }

 response.Success(c, nil)
}

// List 分页列表
// @Summary 获取示例列表
// @Tags 示例管理
// @Produce json
// @Param name query string false "名称"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DemoResponse}}
// @Router /api/v1/auth/demos [get]
func (h *DemoHandler) List(c *gin.Context) {
 var filter dto.DemoFilter
 if err := c.ShouldBindQuery(&filter); err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.String("req_filter", utils.LogStruct(filter)))

 list, total, err := h.DemoSer.List(ctx, filter)
 if err != nil {
  response.Error(c, err)
  return
 }

 response.Paginate(c, list, total)
}

// BatchDelete 批量删除
// @Summary 批量删除示例
// @Tags 示例管理
// @Accept json
// @Produce json
// @Param request body dto.DemoDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/demos/batch [delete]
func (h *DemoHandler) BatchDelete(c *gin.Context) {
 var req dto.DemoDeleteRequest
 if err := c.ShouldBindJSON(&req); err != nil {
  contextx.GetLoggerFromContext(c.Request.Context()).Errorln(err)
  response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
  return
 }

 ctx := contextx.SetLoggerToContext(c.Request.Context(), zap.String("req_body", utils.LogStruct(req)))

 if err := h.DemoSer.BatchDelete(ctx, req); err != nil {
  response.Error(c, err)
  return
 }

 response.Success(c, nil)
}

```
