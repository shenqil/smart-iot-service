package core

// ServiceInterface 采集服务接口
type ServiceInterface interface {
	// 基础方法
	Start() error
	Stop() error
	GetName() string

	// 设备控制方法
	DisconnectedDevice(deviceID uint32) (exists bool)
	DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64)
	RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) (error, bool)
	SetDeviceReadInterval(deviceID uint32, intervalMinutes int) (error, bool)
}
