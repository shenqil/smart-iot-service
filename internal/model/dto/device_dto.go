package dto

import (
	"gorm.io/datatypes"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
)

// 设备相关常量
const (
	DefaultReadInterval = 60
	DefaultPageSize     = 10
	DefaultPageNumber   = 1

	// 预警类型常量
	AlertTypeGreater = "greater"
	AlertTypeLess    = "less"
	AlertTypeEqual   = "equal"
	AlertTypeRange   = "range"

	// 设备状态常量
	DeviceStatusActive   = "active"
	DeviceStatusInactive = "inactive"
	DeviceStatusFault    = "fault"

	// 坐标范围常量
	MaxLongitude = 180
	MinLongitude = -180
	MaxLatitude  = 90
	MinLatitude  = -90
)

// ============================================================================
// 基础配置结构体 - 包含设备的核心配置数据结构
// ============================================================================

// DeviceLocation 地理坐标结构
type DeviceLocation struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// Validate 坐标校验方法
func (l *DeviceLocation) Validate() error {
	if l.Longitude < MinLongitude || l.Longitude > MaxLongitude {
		return errx.New(constant.BadRequest, "经度范围无效", nil)
	}
	if l.Latitude < MinLatitude || l.Latitude > MaxLatitude {
		return errx.New(constant.BadRequest, "纬度范围无效", nil)
	}
	return nil
}

// ============================================================================
// 传感器配置相关结构体 - 处理设备传感器的数据配置和校验
// ============================================================================

// SensorCorrection 传感器数值纠正模型
type SensorCorrection struct {
	Mode             string  `json:"mode,omitempty"`             // 模式：coefficient（系数）、custom（自定义）
	CoefficientValue float64 `json:"coefficientValue,omitempty"` // 系数模式下的系数值
	ErrorValue       float64 `json:"errorValue,omitempty"`       // 误差模式下的误差值
	CustomValue      float64 `json:"customValue,omitempty"`      // 自定义模式下的自定义值
	FloatValue       float64 `json:"floatValue,omitempty"`       // 浮动值，用于所有模式的浮动调整
}

// SensorAlertConfig 传感器预警配置
type SensorAlertConfig struct {
	Enabled        bool    `json:"enabled"`                                                      // 是否启用预警
	AlertType      string  `json:"alertType" binding:"omitempty,oneof=greater less equal range"` // 预警类型：greater（大于）、less（小于）、equal（等于）、range（范围）
	Threshold      float64 `json:"threshold,omitempty"`                                          // 预警阈值（用于 greater、less、equal 类型）
	MinValue       float64 `json:"minValue,omitempty"`                                           // 范围预警的最小值（用于 range 类型）
	MaxValue       float64 `json:"maxValue,omitempty"`                                           // 范围预警的最大值（用于 range 类型）
	RepeatAlert    bool    `json:"repeatAlert"`                                                  // 是否启用重复预警（避免频繁报警）
	RepeatInterval int     `json:"repeatInterval,omitempty"`                                     // 重复预警间隔（秒）
}

// Validate 预警配置验证
func (a *SensorAlertConfig) Validate() error {
	if !a.Enabled {
		return nil
	}

	// 验证预警类型
	if a.AlertType == "" {
		return errx.New(constant.BadRequest, "预警类型不能为空", nil)
	}

	// 根据预警类型验证阈值
	switch a.AlertType {
	case AlertTypeGreater, AlertTypeLess, AlertTypeEqual:
		// 阈值可以为任意值，包括0
	case AlertTypeRange:
		if a.MinValue >= a.MaxValue {
			return errx.New(constant.BadRequest, "范围预警的最小值必须小于最大值", nil)
		}
	default:
		return errx.New(constant.BadRequest, "不支持的预警类型", nil)
	}

	// 验证重复预警间隔
	if a.RepeatAlert && a.RepeatInterval <= 0 {
		return errx.New(constant.BadRequest, "重复预警间隔必须大于0", nil)
	}

	return nil
}

// IsAlertTriggered 检查是否触发告警
func (a *SensorAlertConfig) IsAlertTriggered(value float64) bool {
	if !a.Enabled {
		return false
	}

	switch a.AlertType {
	case AlertTypeGreater:
		return value > a.Threshold
	case AlertTypeLess:
		return value < a.Threshold
	case AlertTypeEqual:
		return value == a.Threshold
	case AlertTypeRange:
		return value >= a.MinValue && value < a.MaxValue
	}
	return false
}

// SensorConfig 传感器配置
type SensorConfig struct {
	ChannelNum   uint                `json:"channelNum" binding:"required"`
	StandardID   *uint               `json:"standardId,omitempty"`
	StandardName string              `json:"standardName,omitempty"`
	DisplayName  string              `json:"displayName,omitempty"`
	MinValue     float64             `json:"minValue,omitempty"`
	MaxValue     float64             `json:"maxValue,omitempty"`
	Resolution   float64             `json:"resolution,omitempty"`
	Unit         string              `json:"unit,omitempty"`
	URL          string              `json:"url,omitempty"`
	Correction   *SensorCorrection   `json:"correction,omitempty"`   // 纠正配置，可选
	AlertConfigs []SensorAlertConfig `json:"alertConfigs,omitempty"` // 预警配置数组，可选
}

// Validate 传感器配置验证
func (s *SensorConfig) Validate() error {
	// 验证预警配置数组
	for _, alertConfig := range s.AlertConfigs {
		if err := alertConfig.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// 继电器配置相关结构体 - 处理设备继电器的数据配置
// ============================================================================

// RelayConfig 继电器配置
type RelayConfig struct {
	ChannelNum   uint   `json:"channelNum" binding:"required"`
	StandardID   *uint  `json:"standardId,omitempty"`
	StandardName string `json:"standardName,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	URL          string `json:"url,omitempty"`
}

// ============================================================================
// 摄像头配置相关结构体 - 处理设备摄像头的数据配置
// ============================================================================

// CameraConfig 摄像头配置
type CameraConfig struct {
	AppKey        string `json:"appKey,omitempty"`
	Secret        string `json:"secret,omitempty"`
	SerialNumber  string `json:"serialNumber,omitempty"`
	Type          string `json:"type,omitempty"`
	ChannelNumber uint   `json:"channelNumber,omitempty"`
}

// ============================================================================
// 请求结构体 - 处理设备相关的API请求数据结构
// ============================================================================

// DeviceCreateRequest 创建设备请求 - 用于创建设备时的完整数据
type DeviceCreateRequest struct {
	DeviceNum           uint                       `json:"deviceNum" binding:"required"`
	Name                *string                    `json:"name,omitempty" binding:"omitempty,max=100"`
	Description         string                     `json:"description,omitempty"`
	DeviceType          string                     `json:"deviceType" binding:"required"`
	Location            *DeviceLocation            `json:"location,omitempty"`
	ReadInterval        *int                       `json:"readInterval,omitempty" binding:"omitempty,min=1"`
	Address             int                        `json:"address,omitempty" binding:"omitempty,min=1"`
	MACAddress          *string                    `json:"macAddress,omitempty"`
	SIMCard             string                     `json:"simCard,omitempty"`
	IsActive            *bool                      `json:"isActive,omitempty"`
	Sensors             []SensorConfig             `json:"sensors" binding:"required,dive"`
	Relays              []RelayConfig              `json:"relays" binding:"required,dive"`
	CameraConfig        *CameraConfig              `json:"cameraConfig,omitempty"`
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
	CreatorID           uint                       `json:"-" form:"-"` // 创建者ID（从上下文获取，不暴露给用户）
	Status              string                     `json:"status" binding:"omitempty,oneof=active inactive fault"`
	CoverImage          string                     `json:"coverImage" binding:"omitempty,max=255"`
	CompanyID           uint                       `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// DeviceUpdateRequest 更新设备请求 - 用于更新设备时的完整数据
type DeviceUpdateRequest struct {
	Name                *string                    `json:"name,omitempty" binding:"omitempty,max=100"`
	Description         *string                    `json:"description,omitempty"`
	DeviceType          *string                    `json:"deviceType,omitempty" binding:"omitempty"`
	Location            *DeviceLocation            `json:"location,omitempty"`
	ReadInterval        *int                       `json:"readInterval,omitempty" binding:"omitempty,min=1"`
	MACAddress          *string                    `json:"macAddress,omitempty" binding:"omitempty,mac"`
	Address             *int                       `json:"address,omitempty" binding:"omitempty,min=1"`
	SIMCard             *string                    `json:"simCard,omitempty"`
	IsActive            *bool                      `json:"isActive,omitempty"`
	Sensors             *[]SensorConfig            `json:"sensors,omitempty" binding:"omitempty,dive"`
	Relays              *[]RelayConfig             `json:"relays,omitempty" binding:"omitempty,dive"`
	CameraConfig        *CameraConfig              `json:"cameraConfig,omitempty"`
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
	Status              *string                    `json:"status,omitempty" binding:"omitempty,oneof=active inactive fault"`
	CoverImage          *string                    `json:"coverImage,omitempty" binding:"omitempty,max=255"`
	CompanyID           uint                       `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// DeviceDeleteRequest 批量删除设备请求 - 用于批量删除设备时的数据
type DeviceDeleteRequest struct {
	IDs       []uint `json:"ids" binding:"required,min=1"`
	CompanyID uint   `json:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// DeviceFilter 设备查询过滤器 - 用于分页查询设备时的过滤条件
type DeviceFilter struct {
	DeviceNum         *uint   `form:"deviceNum,omitempty"`
	DeviceType        *string `form:"deviceType,omitempty"`
	MACAddress        *string `form:"macAddress,omitempty"`
	IsActive          *bool   `form:"isActive,omitempty"`
	Status            *string `form:"status,omitempty"`
	ReadInterval      *int    `form:"readInterval,omitempty"`
	SiteID            *uint   `form:"siteId,omitempty"`        // 园区ID过滤
	ExcludeSiteID     *uint   `form:"excludeSiteId,omitempty"` // 排除指定园区ID的设备
	CompanyID         uint    `form:"-" json:"-"`              // 公司ID（从上下文获取，不暴露给用户）
	Q                 *string `form:"q,omitempty"`             // 模糊搜索参数
	PaginationRequest `form:",inline"`
}

// DeviceCompany 设备公司信息 - 设备关联的公司基本信息
type DeviceCompany struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// ============================================================================
// 响应结构体 - 处理设备相关的API响应数据结构
// ============================================================================

// DeviceResponse 设备响应结构 - 返回设备完整信息时的数据结构
type DeviceResponse struct {
	ID                  uint                       `json:"id"`
	DeviceNum           uint                       `json:"deviceNum"`
	Name                *string                    `json:"name,omitempty"`
	Description         string                     `json:"description"`
	DeviceType          string                     `json:"deviceType"`
	Location            *DeviceLocation            `json:"location"`
	ReadInterval        *int                       `json:"readInterval,omitempty"`
	MACAddress          *string                    `json:"macAddress"`
	Address             int                        `json:"address"`
	SIMCard             string                     `json:"simCard"`
	IsActive            bool                       `json:"isActive"`
	Sensors             []SensorConfig             `json:"sensors"`
	Relays              []RelayConfig              `json:"relays"`
	CameraConfig        *CameraConfig              `json:"cameraConfig,omitempty"`
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
	CreatorID           uint                       `json:"creatorId"`
	Status              string                     `json:"status"`
	CoverImage          string                     `json:"coverImage"`
	Company             *DeviceCompany             `json:"company,omitempty"`
	CreatedAt           int64                      `json:"createdAt"`
	UpdatedAt           int64                      `json:"updatedAt"`
}

// ============================================================================
// 数据转换方法 - 处理DTO与实体之间的数据转换
// ============================================================================

// ConvertToEntity 将创建请求转换为实体 - 将DeviceCreateRequest转换为数据库实体
func (req *DeviceCreateRequest) ConvertToEntity() (*entity.Device, error) {
	// 坐标校验
	if err := req.Location.Validate(); err != nil {
		return nil, err
	}

	device := &entity.Device{
		DeviceNum:    req.DeviceNum,
		Name:         req.Name,
		Description:  req.Description,
		DeviceType:   req.DeviceType,
		ReadInterval: req.ReadInterval,
		Address:      req.Address,
		MACAddress:   req.MACAddress,
		SIMCard:      req.SIMCard,
		IsActive:     req.IsActive,
		CreatorID:    req.CreatorID,
		CoverImage:   req.CoverImage,
		CompanyID:    &req.CompanyID,
	}

	// 状态设置
	if req.Status != "" {
		device.Status = entity.DeviceStatus(req.Status)
	} else {
		device.Status = entity.DeviceStatusActive
	}

	// 地理坐标转换
	if req.Location != nil {
		locationJSON, err := convertToJSON(*req.Location, "经纬度")
		if err != nil {
			return nil, err
		}
		device.Location = locationJSON
	}

	// 验证并转换传感器配置
	if err := validateSensors(req.Sensors); err != nil {
		return nil, err
	}
	sensorsJSON, err := convertToJSON(req.Sensors, "传感器")
	if err != nil {
		return nil, err
	}
	device.Sensors = sensorsJSON

	// 转换继电器配置
	relaysJSON, err := convertToJSON(req.Relays, "继电器")
	if err != nil {
		return nil, err
	}
	device.Relays = relaysJSON

	// 摄像头配置
	if req.CameraConfig != nil {
		cameraJSON, err := convertToJSON(*req.CameraConfig, "摄像头")
		if err != nil {
			return nil, err
		}
		device.CameraConfig = cameraJSON
	}

	// 水肥信息配置
	if req.WaterFertilizerInfo != nil {
		waterFertilizerJSON, err := convertToJSON(*req.WaterFertilizerInfo, "水肥信息")
		if err != nil {
			return nil, err
		}
		device.WaterFertilizerInfo = waterFertilizerJSON
	}

	return device, nil
}

// ConvertToQuery 将过滤器转换为实体查询 - 将DeviceFilter转换为数据库查询条件
func (f *DeviceFilter) ConvertToQuery() entity.DeviceQuery {
	query := entity.DeviceQuery{
		DeviceNum:     f.DeviceNum,
		MACAddress:    f.MACAddress,
		IsActive:      f.IsActive,
		ReadInterval:  f.ReadInterval,
		SiteID:        f.SiteID,
		ExcludeSiteID: f.ExcludeSiteID,
		CompanyID:     &f.CompanyID,
		Q:             f.Q,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}

	// 设备类型转换
	if f.DeviceType != nil && *f.DeviceType != "" {
		query.DeviceTypes = []string{*f.DeviceType}
	}

	// 状态转换
	if f.Status != nil {
		status := entity.DeviceStatus(*f.Status)
		query.Status = &status
	}

	return query
}

// ConvertToDeviceResponse 将实体转换为响应 - 将数据库实体转换为DeviceResponse
func ConvertToDeviceResponse(device *entity.Device, getStatus func(deviceID uint) string) (*DeviceResponse, error) {

	status := getStatus(device.DeviceNum)

	resp := &DeviceResponse{
		ID:           device.ID,
		DeviceNum:    device.DeviceNum,
		Name:         device.Name,
		Description:  device.Description,
		DeviceType:   device.DeviceType,
		ReadInterval: device.ReadInterval,
		Address:      device.Address,
		MACAddress:   device.MACAddress,
		SIMCard:      device.SIMCard,
		CreatorID:    device.CreatorID,
		Status:       status,
		CoverImage:   device.CoverImage,
		CreatedAt:    device.CreatedAt.Unix(),
		UpdatedAt:    device.UpdatedAt.Unix(),
	}

	// 设置激活状态
	if device.IsActive != nil {
		resp.IsActive = *device.IsActive
	} else {
		resp.IsActive = true
	}

	// 地理坐标转换
	var location DeviceLocation
	if err := parseJSON(device.Location, &location); err == nil {
		resp.Location = &location
	}

	// 传感器配置转换
	var sensors []SensorConfig
	if err := parseJSON(device.Sensors, &sensors); err == nil {
		resp.Sensors = sensors
	}

	// 继电器配置转换
	var relays []RelayConfig
	if err := parseJSON(device.Relays, &relays); err == nil {
		resp.Relays = relays
	}

	// 摄像头配置
	if device.CameraConfig != nil {
		var cameraConfig CameraConfig
		if err := parseJSON(device.CameraConfig, &cameraConfig); err == nil {
			resp.CameraConfig = &cameraConfig
		}
	}

	// 水肥信息配置
	if device.WaterFertilizerInfo != nil {
		var waterFertilizerInfo DeviceWaterFertilizerInfo
		if err := parseJSON(device.WaterFertilizerInfo, &waterFertilizerInfo); err == nil {
			resp.WaterFertilizerInfo = &waterFertilizerInfo
		}
	}

	// 处理公司信息
	if device.Company != nil {
		resp.Company = &DeviceCompany{
			ID:   device.Company.ID,
			Name: device.Company.Name,
			Code: device.Company.Code,
		}
	}

	return resp, nil
}

// ============================================================================
// 分页辅助方法 - 处理分页相关的安全获取逻辑
// ============================================================================

// GetPageNumber 安全获取页码 - 确保页码不小于1
func (f *DeviceFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return DefaultPageNumber
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小 - 确保每页大小不小于1
func (f *DeviceFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return DefaultPageSize
	}
	return f.PageSize
}

// ============================================================================
// 辅助函数
// ============================================================================

// convertToJSON 安全地将对象转换为JSON
func convertToJSON(data interface{}, fieldName string) (datatypes.JSON, error) {
	jsonData, err := ToJSON(data)
	if err != nil {
		return nil, errx.New(constant.BadRequest, fieldName+"配置转换失败", err)
	}
	return jsonData, nil
}

// validateSensors 验证传感器配置数组
func validateSensors(sensors []SensorConfig) error {
	for _, sensor := range sensors {
		if err := sensor.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// parseJSON 安全地从JSON解析数据
func parseJSON(jsonData datatypes.JSON, target interface{}) error {
	if jsonData == nil {
		return nil
	}
	return FromJSON(jsonData, target)
}

// ============================================================================
// 独立更新请求结构体 - 专门用于单独更新设备某个配置项的请求
// ============================================================================

// UpdateCameraConfigRequest 更新摄像头配置请求 - 单独更新设备的摄像头配置
type UpdateCameraConfigRequest struct {
	CameraConfig *CameraConfig `json:"cameraConfig,omitempty"`
}

// UpdateSensorsRequest 更新传感器配置请求 - 单独更新设备的传感器配置
type UpdateSensorsRequest struct {
	Sensors []SensorConfig `json:"sensors" binding:"required,dive"`
}

// UpdateRelaysRequest 更新继电器配置请求 - 单独更新设备的继电器配置
type UpdateRelaysRequest struct {
	Relays []RelayConfig `json:"relays" binding:"required,dive"`
}

// UpdateLocationRequest 更新地理坐标请求 - 单独更新设备的地理位置坐标
type UpdateLocationRequest struct {
	Location *DeviceLocation `json:"location,omitempty"`
}

// UpdateBasicInfoRequest 更新设备基础信息请求 - 更新设备的基本信息，不包含复杂的配置字段
type UpdateBasicInfoRequest struct {
	Name         *string         `json:"name,omitempty" binding:"omitempty,max=100"`
	Description  *string         `json:"description,omitempty"`
	DeviceType   *string         `json:"deviceType,omitempty"`
	Location     *DeviceLocation `json:"location,omitempty"`
	ReadInterval *int            `json:"readInterval,omitempty" binding:"omitempty,min=1"`
	Address      *int            `json:"address,omitempty" binding:"omitempty,min=1"`
	MACAddress   *string         `json:"macAddress,omitempty" binding:"omitempty,mac"`
	SIMCard      *string         `json:"simCard,omitempty"`
	IsActive     *bool           `json:"isActive,omitempty"`
	Status       *string         `json:"status,omitempty" binding:"omitempty,oneof=active inactive fault"`
	CoverImage   *string         `json:"coverImage,omitempty" binding:"omitempty,max=255"`
}

// ============================================================================
// 水肥信息相关结构体 - 处理设备水肥信息的配置和管理
// ============================================================================

// DeviceWaterFertilizerInfo 设备水肥信息 - 存储设备的完整水肥配置信息
type DeviceWaterFertilizerInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Info string `json:"info"`
}

// UpdateWaterFertilizerInfoRequest 更新水肥信息请求 - 单独更新设备的水肥信息
type UpdateWaterFertilizerInfoRequest struct {
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
}

// GetWaterFertilizerInfoResponse 查询水肥信息响应 - 返回设备水肥信息的查询结果
type GetWaterFertilizerInfoResponse struct {
	WaterFertilizerInfo *DeviceWaterFertilizerInfo `json:"waterFertilizerInfo,omitempty"`
}
