package dto

import (
	"rbac-admin/internal/model/entity"

	"gorm.io/datatypes"
)

// MySiteSimpleResponse 我的园区简单响应结构（不包含设备列表）
type MySiteSimpleResponse struct {
	SiteID      uint           `json:"siteId"`
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Address     string         `json:"address"`
	Province    string         `json:"province"`
	City        string         `json:"city"`
	District    string         `json:"district"`
	PostalCode  string         `json:"postalCode"`
	Longitude   float64        `json:"longitude"`
	Latitude    float64        `json:"latitude"`
	Type        string         `json:"type"`
	CoverImage  string         `json:"coverImage"`
	Images      []string       `json:"images"`
	Properties  datatypes.JSON `json:"properties"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

// MyDeviceResponse 我的设备响应结构
type MyDeviceResponse struct {
	DeviceNum    uint            `json:"deviceNum"`
	Name         *string         `json:"name,omitempty"`
	Description  string          `json:"description"`
	DeviceType   string          `json:"deviceType"`
	Location     *DeviceLocation `json:"location"`
	ReadInterval *int            `json:"readInterval,omitempty"`
	MACAddress   *string         `json:"macAddress"`
	Address      int             `json:"address"`
	SIMCard      string          `json:"simCard"`
	Status       string          `json:"status"`
	Sensors      []SensorConfig  `json:"sensors"`
	Relays       []RelayConfig   `json:"relays"`
	CameraConfig *CameraConfig   `json:"cameraConfig,omitempty"`
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
	CoverImage   string          `json:"coverImage"`
	CreatedAt    int64           `json:"createdAt"`
	UpdatedAt    int64           `json:"updatedAt"`
}

// RelaySwitchRequest 继电器控制请求
type RelaySwitchRequest struct {
	DeviceSN       string `json:"deviceSN"`   // 设备序列号
	RelayIndex     int    `json:"relayIndex"` // 继电器索引(0-31)
	IsOpen         bool   `json:"isOpen"`     // true:开启, false:关闭
	Operator       string `json:"operator"`   // 操作者
	OperatorUserID uint   `json:"-"`          // 操作者用户ID（从上下文获取，不暴露给用户）
}

// DeviceRelayRecordFilter 设备继电器记录过滤条件
type DeviceRelayRecordFilter struct {
	DeviceSN          *uint32          `form:"deviceSN,omitempty"`  // 设备号精确匹配
	Channel           *int             `form:"channel,omitempty"`   // 通道号精确匹配(0-31)
	Operator          *string          `form:"operator,omitempty"`  // 操作者模糊匹配
	Status            *string          `form:"status,omitempty"`    // 状态精确匹配
	StartTime         *int64           `form:"startTime,omitempty"` // 开始时间(秒级时间戳)
	EndTime           *int64           `form:"endTime,omitempty"`   // 结束时间(秒级时间戳)
	PaginationRequest `form:",inline"` // 分页参数
}

// DeviceRelayRecordResponse 设备继电器记录响应结构
type DeviceRelayRecordResponse struct {
	ID        uint   `json:"id"`        // 记录ID
	DeviceSN  uint32 `json:"deviceSN"`  // 设备号
	Channel   int    `json:"channel"`   // 通道号(0-31)
	Operator  string `json:"operator"`  // 操作者
	Status    string `json:"status"`    // 操作状态
	Duration  *int   `json:"duration"`  // 耗时(毫秒)
	Remark    string `json:"remark"`    // 备注
	CreatedAt int64  `json:"createdAt"` // 创建时间(秒级时间戳)
	UpdatedAt int64  `json:"updatedAt"` // 更新时间(秒级时间戳)
}

// MySiteFilter 我的园区过滤条件
type MySiteFilter struct {
	SiteName          *string          `form:"siteName"`   // 园区名称模糊搜索
	SiteCode          *string          `form:"siteCode"`   // 园区code模糊搜索
	CompanyID         uint             `form:"-" json:"-"` // 公司ID（从上下文获取，不暴露给用户）
	UserID            uint             `form:"-" json:"-"` // 用户ID（从上下文获取，不暴露给用户）
	PaginationRequest `form:",inline"` // 分页参数
}

// MyDeviceFilter 我的设备过滤条件
type MyDeviceFilter struct {
	SiteCodes         []string         `form:"siteCode"`   // 园区code数组，支持多选
	DeviceTypes       []string         `form:"deviceType"` // 设备类型数组，支持多选
	DeviceName        *string          `form:"deviceName"` // 设备名称模糊搜索
	Q                 *string          `form:"q"`          // 综合查询条件，用于DeviceNum、Name、SIMCard
	CompanyID         uint             `form:"-" json:"-"` // 公司ID（从上下文获取，不暴露给用户）
	PaginationRequest `form:",inline"` // 分页参数
}

// MyDeviceTypeStatsFilter 我的设备类型统计过滤条件
type MyDeviceTypeStatsFilter struct {
	SiteCodes []string `form:"siteCode"`   // 园区code数组，支持多选，为空则查询所有园区
	CompanyID uint     `form:"-" json:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// MyDeviceTypeStatsResponse 我的设备类型统计响应结构
type MyDeviceTypeStatsResponse struct {
	DeviceType string `json:"deviceType"` // 设备类型
	Count      int64  `json:"count"`      // 设备数量
}

// ConvertToMySiteSimpleResponse 将实体转换为我的园区简单响应（不包含设备）
func ConvertToMySiteSimpleResponse(site *entity.Site) (*MySiteSimpleResponse, error) {
	resp := &MySiteSimpleResponse{
		SiteID:      site.ID,
		Name:        site.Name,
		Title:       site.Title,
		Code:        site.Code,
		Description: site.Description,
		Address:     site.Address,
		Province:    site.Province,
		City:        site.City,
		District:    site.District,
		PostalCode:  site.PostalCode,
		Longitude:   site.Longitude,
		Latitude:    site.Latitude,
		Type:        site.Type,
		CoverImage:  site.CoverImage,
		Properties:  site.Properties,
		CreatedAt:   site.CreatedAt.Unix(),
		UpdatedAt:   site.UpdatedAt.Unix(),
	}

	// 处理图片
	var images []string
	if err := FromJSON(site.Images, &images); err == nil {
		resp.Images = images
	}

	return resp, nil
}

// ConvertToMyDeviceResponse 将实体转换为我的设备响应
func ConvertToMyDeviceResponse(device *entity.Device, getStatus func(deviceID uint) string) (*MyDeviceResponse, error) {
	status := getStatus(device.DeviceNum)

	resp := &MyDeviceResponse{
		DeviceNum:    device.DeviceNum,
		Name:         device.Name,
		Description:  device.Description,
		DeviceType:   device.DeviceType,
		ReadInterval: device.ReadInterval,
		Address:      device.Address,
		MACAddress:   device.MACAddress,
		SIMCard:      device.SIMCard,
		CoverImage:   device.CoverImage,
		Status:       string(status),
		CreatedAt:    device.CreatedAt.Unix(),
		UpdatedAt:    device.UpdatedAt.Unix(),
	}

	// 地理坐标转换
	var location DeviceLocation
	if err := FromJSON(device.Location, &location); err == nil {
		resp.Location = &location
	}

	// 传感器配置转换
	var sensors []SensorConfig
	if err := FromJSON(device.Sensors, &sensors); err == nil {
		resp.Sensors = sensors
	}

	// 继电器配置转换
	var relays []RelayConfig
	if err := FromJSON(device.Relays, &relays); err == nil {
		resp.Relays = relays
	}

	// 摄像头配置
	var cameraConfig CameraConfig
	if err := FromJSON(device.CameraConfig, &cameraConfig); err == nil {
		resp.CameraConfig = &cameraConfig
	}

	// 水肥信息配置
	if device.WaterFertilizerInfo != nil {
		var waterFertilizerInfo DeviceWaterFertilizerInfo
		if err := FromJSON(device.WaterFertilizerInfo, &waterFertilizerInfo); err == nil {
			resp.WaterFertilizerInfo = &waterFertilizerInfo
		}
	}

	return resp, nil
}

// ToDeviceQuery 将 MyDeviceFilter 转换为 entity.DeviceQuery
func (f *MyDeviceFilter) ToDeviceQuery() *entity.DeviceQuery {
	if f == nil {
		return nil
	}

	query := &entity.DeviceQuery{
		CompanyID: &f.CompanyID,
		Pagination: entity.Pagination{
			PageNumber: f.PageNumber,
			PageSize:   f.PageSize,
		},
	}

	if len(f.DeviceTypes) > 0 {
		query.DeviceTypes = f.DeviceTypes // 支持多设备类型查询
	}

	if f.DeviceName != nil && *f.DeviceName != "" {
		query.DeviceName = f.DeviceName // 使用指定字段进行模糊搜索，避免性能问题
	}

	if f.Q != nil && *f.Q != "" {
		query.Q = f.Q // 综合查询条件，用于DeviceNum、Name、SIMCard
	}

	if len(f.SiteCodes) > 0 {
		query.SiteCodes = f.SiteCodes // 园区编码数组，支持多端查询
	}

	return query
}

// ToSiteQuery 将 MySiteFilter 转换为 entity.SiteQuery
func (f *MySiteFilter) ToSiteQuery() *entity.SiteQuery {
	if f == nil {
		return nil
	}

	query := &entity.SiteQuery{
		CompanyID: &f.CompanyID,
		Pagination: entity.Pagination{
			PageNumber: f.PageNumber,
			PageSize:   f.PageSize,
		},
	}

	if f.SiteName != nil && *f.SiteName != "" {
		query.Q = f.SiteName // 使用 Q 字段进行模糊搜索
	}

	if f.SiteCode != nil && *f.SiteCode != "" {
		query.Code = f.SiteCode // 园区编码精确匹配
	}

	return query
}

// ToDeviceQuery 将 MyDeviceTypeStatsFilter 转换为 entity.DeviceQuery
func (f *MyDeviceTypeStatsFilter) ToDeviceQuery() *entity.DeviceQuery {
	if f == nil {
		return nil
	}

	query := &entity.DeviceQuery{
		CompanyID: &f.CompanyID,
	}

	if len(f.SiteCodes) > 0 {
		query.SiteCodes = f.SiteCodes // 园区编码数组，支持多选
	}

	return query
}
