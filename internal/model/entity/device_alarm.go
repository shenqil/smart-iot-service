package entity

import (
	"time"
)

// 枚举类型（使用 MySQL 8.0+，字段采用更明确的枚举/数值类型）
type AlarmLevel uint8
type ProcessStatus uint8

const (
	AlarmLevelCritical AlarmLevel = 1 // 紧急
	AlarmLevelError    AlarmLevel = 2 // 重要
	AlarmLevelWarning  AlarmLevel = 3 // 警告
	AlarmLevelInfo     AlarmLevel = 4 // 信息
)

// DeviceAlarmQuery 告警查询条件
type DeviceAlarmQuery struct {
	Q            *string        // 模糊搜索：设备名称、设备ID
	EnterpriseId *string        // 企业ID过滤
	DeviceId     *string        // 设备ID过滤
	SiteCode     *string        // 园区编码过滤
	DeviceIds    []string       // 设备ID列表（通过园区编码查询得到）
	AlarmType    *string        // 告警类型过滤
	AlarmLevel   *AlarmLevel    // 告警级别过滤
	IsProcessed  *ProcessStatus // 是否已处理过滤(0/1)
	ProcessedBy  *uint          // 处理人ID过滤
	StartTime    *time.Time     // 开始时间（按创建时间）
	EndTime      *time.Time     // 结束时间（按创建时间）
	Pagination                  // 分页参数（外部已有定义）
}

// GetPageNumber 安全获取页码
func (q *DeviceAlarmQuery) GetPageNumber() int {
	if q.PageNumber <= 0 {
		return 1
	}
	return q.PageNumber
}

// GetPageSize 安全获取每页大小
func (q *DeviceAlarmQuery) GetPageSize() int {
	if q.PageSize <= 0 {
		return 10
	}
	return q.PageSize
}

const (
	ProcessStatusUnprocessed ProcessStatus = 0 // 未处理
	ProcessStatusProcessed   ProcessStatus = 1 // 已处理
)

// 统一设备告警实体（通用告警）
type DeviceAlarm struct {
	ID           uint       `gorm:"primaryKey;autoIncrement;comment:业务ID"`
	AlarmLevel   AlarmLevel `gorm:"type:TINYINT UNSIGNED;index:idx_ent_processed_created,priority:2;index:idx_ent_type_created,priority:3;comment:告警级别"` // 1:Critical, 2:Error, 3:Warning, 4:Info
	AlarmType    string     `gorm:"size:50;index:idx_ent_type_created,priority:2;comment:告警类型(业务侧校验)"`
	AlarmSubType string     `gorm:"size:20;comment:告警子类型(例如 greater/less/equal/range)"`
	EnterpriseId string     `gorm:"size:64;index:idx_ent_created,priority:1;index:idx_ent_processed_created,priority:1;index:idx_ent_type_created,priority:1;comment:企业ID"`
	DeviceId     string     `gorm:"size:128;index;comment:设备ID"`
	DeviceName   string     `gorm:"size:200;comment:设备名称"`
	Description  string     `gorm:"type:text;comment:告警描述"`
	Value        string     `gorm:"size:100;comment:告警数值"`
	Threshold    string     `gorm:"size:100;comment:告警阈值"`

	// 告警处理相关
	IsProcessed   ProcessStatus `gorm:"type:TINYINT UNSIGNED;default:0;comment:是否已处理(0:未处理,1:已处理)"`
	ProcessedBy   *uint         `gorm:"comment:处理人ID"`
	ProcessorName string        `gorm:"size:50;comment:处理人姓名"`
	ProcessRemark string        `gorm:"type:text;comment:处理备注"`
	ProcessedAt   *time.Time    `gorm:"comment:处理时间"`

	CreatedAt time.Time `gorm:"index:idx_ent_created,priority:2;index:idx_ent_processed_created,priority:3;index:idx_ent_type_created,priority:4;comment:创建时间"`
}

// 告警类型常量（字符串枚举）
const (
	// 设备状态
	AlarmTypeDeviceOffline   = "device_offline"   // 设备离线
	AlarmTypeDeviceOnline    = "device_online"    // 设备上线
	AlarmTypeDeviceRestart   = "device_restart"   // 设备重启
	AlarmTypeLowBattery      = "low_battery"      // 低电量
	AlarmTypePowerOff        = "power_off"        // 电源断开
	AlarmTypeStorageFull     = "storage_full"     // 存储空间满
	AlarmTypeSIMInvalid      = "sim_invalid"      // SIM卡失效
	AlarmTypeHardwareFault   = "hardware_fault"   // 硬件故障
	AlarmTypeHighTemperature = "high_temperature" // 设备高温

	// 网络通信
	AlarmTypeWeakSignal        = "weak_signal"        // 信号弱
	AlarmTypeHighLatency       = "high_latency"       // 高延迟
	AlarmTypePacketLoss        = "packet_loss"        // 数据包丢失
	AlarmTypeConnectionLimit   = "connection_limit"   // 连接数超限
	AlarmTypeNetworkDisconnect = "network_disconnect" // 网络断开

	// 数据与业务
	AlarmTypeDataExceedLimit   = "data_exceed_limit"  // 数据超限
	AlarmTypeDataAbnormal      = "data_abnormal"      // 数据异常
	AlarmTypeDataStuck         = "data_stuck"         // 数据卡死
	AlarmTypeBusinessViolation = "business_violation" // 业务违规
	AlarmTypeThresholdBreach   = "threshold_breach"   // 阈值突破

	// 安全与入侵
	AlarmTypeUnauthorizedAccess = "unauthorized_access" // 未授权访问
	AlarmTypeFirmwareTamper     = "firmware_tamper"     // 固件篡改
	AlarmTypeAbnormalTraffic    = "abnormal_traffic"    // 异常流量
	AlarmTypePortScan           = "port_scan"           // 端口扫描
	AlarmTypeGeoFenceBreach     = "geo_fence_breach"    // 地理围栏突破
	AlarmTypeBruteForce         = "brute_force"         // 暴力破解

	// 系统服务
	AlarmTypeServiceDown   = "service_down"   // 服务下线
	AlarmTypeHighCPU       = "high_cpu"       // CPU使用率高
	AlarmTypeHighMemory    = "high_memory"    // 内存使用率高
	AlarmTypeDiskFull      = "disk_full"      // 磁盘空间满
	AlarmTypeDatabaseError = "database_error" // 数据库错误
)
