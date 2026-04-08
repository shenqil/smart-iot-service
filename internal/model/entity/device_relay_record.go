package entity

import (
	"time"

	"gorm.io/gorm"
)

// OperationStatus 操作状态枚举
type OperationStatus string

const (
	OperationStatusOpening      OperationStatus = "opening"       // 开启中
	OperationStatusClosing      OperationStatus = "closing"       // 关闭中
	OperationStatusOpenSuccess  OperationStatus = "open_success"  // 开启成功
	OperationStatusOpenFailed   OperationStatus = "open_failed"   // 开启失败
	OperationStatusCloseSuccess OperationStatus = "close_success" // 关闭成功
	OperationStatusCloseFailed  OperationStatus = "close_failed"  // 关闭失败
)

// DeviceRelayRecord 设备继电器记录模型
type DeviceRelayRecord struct {
	gorm.Model
	DeviceSN       uint32          `gorm:"type:int unsigned;not null;index:idx_device_sn_created,priority:1;comment:设备号"` // 设备号
	Channel        int             `gorm:"not null;comment:通道号"`                                                          // 通道号
	Operator       string          `gorm:"size:50;not null;index;comment:操作者"`                                            // 操作者
	OperatorUserID *uint           `gorm:"index;comment:操作者用户ID"`                                                         // 操作者用户ID（可选）
	Status         OperationStatus `gorm:"size:20;not null;index;comment:操作状态"`                                           // 状态
	Duration       *int            `gorm:"comment:耗时(毫秒)"`                                                                // 耗时（毫秒）
	Remark         string          `gorm:"type:text;comment:备注"`                                                          // 备注
}

// DeviceRelayRecordQuery 设备继电器记录查询条件
type DeviceRelayRecordQuery struct {
	DeviceSN   *uint32          // 设备号精确匹配
	Channel    *int             // 通道号精确匹配
	Operator   *string          // 操作者模糊匹配
	Status     *OperationStatus // 状态精确匹配
	StartTime  *time.Time       // 开始时间（按创建时间）
	EndTime    *time.Time       // 结束时间（按创建时间）
	Pagination                  // 分页参数
}

// GetPageNumber 安全获取页码
func (q *DeviceRelayRecordQuery) GetPageNumber() int {
	if q.PageNumber <= 0 {
		return 1
	}
	return q.PageNumber
}

// GetPageSize 安全获取每页大小
func (q *DeviceRelayRecordQuery) GetPageSize() int {
	if q.PageSize <= 0 {
		return 10
	}
	return q.PageSize
}
