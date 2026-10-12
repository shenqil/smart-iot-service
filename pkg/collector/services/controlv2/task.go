package controlv2

import (
	"fmt"
	"github.com/panjf2000/gnet/v2"
)

func (s *ControlServiceV2) DisconnectedDevice(deviceID uint32) bool {
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return false
	}
	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return false
	}

	s.scheduler.StopTask(deviceID)
	conn.Close()
	s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
	return true
}

// BuildReadIDCommand 构建读取设备ID请求报文 (下行固定地址0: 00 03 00 60 00 04 45 C6)
func (s *ControlServiceV2) BuildReadIDCommand() []byte {
	return []byte{0x00, 0x03, 0x00, 0x60, 0x00, 0x04, 0x45, 0xc6}
}

// BuildDataRequestCommand 构建数据轮询请求报文 (下行固定地址0: 00 03 00 00 F0 24)
func (s *ControlServiceV2) BuildDataRequestCommand() []byte {
	return []byte{0x00, 0x03, 0x00, 0x00, 0xf0, 0x24}
}

func (s *ControlServiceV2) startDataReadingTask(deviceID uint32, intervalMinutes int) {
	task := func() {
		connInterface, exists := s.deviceConns.Load(deviceID)
		if !exists {
			s.logger.Warn(fmt.Sprintf("%s 任务执行时设备 %d 未连接，终止任务", s.name, deviceID))
			s.scheduler.StopTask(deviceID)
			return
		}
		conn, ok := connInterface.(gnet.Conn)
		if !ok {
			return
		}

		cmd := s.BuildDataRequestCommand()
		if _, err := conn.Write(cmd); err != nil {
			s.logger.Error(fmt.Sprintf("%s 轮询请求失败 (设备ID: %d): %v", s.name, deviceID, err))
		}
	}
	s.scheduler.StartTask(deviceID, intervalMinutes, task)
	s.logger.Info(fmt.Sprintf("%s 启动轮询任务 (设备ID: %d, 间隔: %d分钟)", s.name, deviceID, intervalMinutes))
}

// BuildRelayControlCommand 构建继电器控制请求报文 (下行固定地址0)
func (s *ControlServiceV2) BuildRelayControlCommand(relayIndex int, isOpen bool) []byte {
	cmd := make([]byte, 8)
	cmd[0] = 0x00             // 设备地址固定为0
	cmd[1] = 0x10             // 功能码
	cmd[2] = 0x00             // 起始位置高字节
	cmd[3] = 0x7A             // 起始位置低字节
	cmd[4] = byte(relayIndex) // 继电器索引
	cmd[5] = 0x01             // 继电器数量
	cmd[6] = 0x00             // 数据长度高字节
	cmd[7] = 0x02             // 数据长度低字节

	status := byte(0x00)
	if isOpen {
		status = 0x01
	}
	cmd = append(cmd, status)

	crc1, crc2 := s.calculateCRC(cmd)
	return append(cmd, crc1, crc2)
}

func (s *ControlServiceV2) RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) (error, bool) {
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return fmt.Errorf("设备 %d 未连接", deviceID), false
	}
	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return fmt.Errorf("设备 %d 连接无效", deviceID), false
	}

	cmd := s.BuildRelayControlCommand(relayIndex, isOpen)

	if _, err := conn.Write(cmd); err != nil {
		s.logger.Error(fmt.Sprintf("%s 继电器控制下发失败 (ID: %d): %v", s.name, deviceID, err))
		return err, true
	}
	return nil, true
}

func (s *ControlServiceV2) SetDeviceReadInterval(deviceID uint32, intervalMinutes int) (error, bool) {
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return fmt.Errorf("设备 %d 未连接", deviceID), false
	}
	if _, ok := connInterface.(gnet.Conn); !ok {
		return fmt.Errorf("设备 %d 连接无效", deviceID), false
	}

	s.scheduler.StopTask(deviceID)
	if intervalMinutes > 0 {
		s.startDataReadingTask(deviceID, intervalMinutes)
	}

	return nil, true
}

func (s *ControlServiceV2) DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64) {
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return
	}
	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return
	}

	ctx, ok := conn.Context().(*ControlConnContextV2)
	if !ok {
		return
	}

	ctx.mu.Lock()
	connID := ctx.connectionID
	connTime := ctx.connectionTime
	ctx.mu.Unlock()

	if connID == currentConnectionID {
		return
	}

	if currentTime > connTime {
		s.logger.Warn(fmt.Sprintf("%s 挤占旧连接 (设备: %d)", s.name, deviceID))
		s.scheduler.StopTask(deviceID)
		conn.Close()
	}
}
