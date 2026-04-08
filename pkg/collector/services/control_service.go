package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"rbac-admin/pkg/collector/singletons"
	"rbac-admin/pkg/logger"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2"
)

// ControlConnContext 连接上下文
type ControlConnContext struct {
	deviceID       uint32        // 设备ID
	timeoutTimer   *time.Timer   // 超时定时器
	buffer         *bytes.Buffer // 数据缓冲区，用于处理粘包和分包
	packetTimer    *time.Timer   // 数据包超时定时器
	connectionID   string        // 连接ID
	connectionTime int64         // 连接时间戳
}

// ControlService 控制服务
type ControlService struct {
	gnet.BuiltinEventEngine
	eng         gnet.Engine
	addr        string
	ctx         context.Context
	cancel      context.CancelFunc
	name        string
	logger      *slog.Logger // 服务专用日志器
	deviceConns sync.Map     // map[uint32]gnet.Conn - 设备ID到连接的映射
	scheduler   *singletons.Scheduler
}

// NewControlService 创建新的控制服务
func NewControlService(addr string) *ControlService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ControlService{
		addr:      addr,
		ctx:       ctx,
		cancel:    cancel,
		name:      "控制服务",
		logger:    logger.NewServiceLogger("control_service"),
		scheduler: singletons.GetScheduler(),
	}
}

// GetName 获取服务名称
func (s *ControlService) GetName() string {
	return s.name
}

// Start 启动服务
func (s *ControlService) Start() error {
	s.logger.Info(fmt.Sprintf("启动 %s，监听地址: %s", s.name, s.addr))

	// 启动gnet服务器
	if err := gnet.Run(s, s.addr, gnet.WithMulticore(true)); err != nil {
		s.logger.Error(fmt.Sprintf("启动 %s 失败: %v", s.name, err))
		return err
	}

	s.logger.Info(fmt.Sprintf("%s 启动成功", s.name))
	return nil
}

// Stop 停止服务
func (s *ControlService) Stop() error {
	s.logger.Info(fmt.Sprintf("停止 %s", s.name))

	// 取消上下文
	s.cancel()

	// 停止所有调度任务
	if s.scheduler != nil {
		s.scheduler.StopAllTasks()
	}

	// 关闭所有设备连接
	s.deviceConns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(gnet.Conn); ok {
			conn.Close()
		}
		return true
	})

	// 清空连接映射
	s.deviceConns = sync.Map{}

	s.logger.Info(fmt.Sprintf("%s 已停止", s.name))
	return nil
}

// DisconnectedDevice 断开某个设备连接
func (s *ControlService) DisconnectedDevice(deviceID uint32) bool {
	// 查找对应的连接
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return false // 服务存在但设备未连接，返回不存在
	}

	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return false // 服务存在但连接无效，返回不存在
	}

	// 停止调度器任务
	s.scheduler.StopTask(deviceID)

	// 关闭连接
	conn.Close()
	s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
	return true // 服务存在并已处理
}

// gnet事件处理
func (s *ControlService) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	s.logger.Info(fmt.Sprintf("%s 服务启动，监听地址: %s", s.name, s.addr))
	return gnet.None
}

func (s *ControlService) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	ctx := &ControlConnContext{
		deviceID:       0, // 初始设备ID为0，等待设备响应
		timeoutTimer:   nil,
		buffer:         bytes.NewBuffer(nil),    // 初始化数据缓冲区
		packetTimer:    nil,                     // 数据包定时器
		connectionID:   c.RemoteAddr().String(), // 使用远程地址作为连接ID
		connectionTime: time.Now().Unix(),       // 当前时间戳
	}

	c.SetContext(ctx)

	s.logger.Info(fmt.Sprintf("%s 客户端连接: %s", s.name, c.RemoteAddr().String()))

	// 启动10秒超时定时器，如果15秒内没有收到设备ID，则关闭连接
	ctx.timeoutTimer = time.AfterFunc(10*time.Second, func() {
		if ctx.deviceID == 0 {
			s.logger.Warn(fmt.Sprintf("%s 连接 %s 在15秒内未收到设备ID，主动关闭连接", s.name, c.RemoteAddr().String()))
			c.Close()
		}
	})

	// 构建读取ID指令并直接返回发送
	idRequestCmd := s.buildReadIDCommand()
	s.logger.Info(fmt.Sprintf("%s 发送读取ID指令", s.name))

	return idRequestCmd, gnet.None
}

func (s *ControlService) OnClose(c gnet.Conn, err error) gnet.Action {
	addr := c.RemoteAddr().String()

	// 清理连接上下文和定时器，获取设备ID并清理
	if ctxInterface := c.Context(); ctxInterface != nil {
		if ctx, ok := ctxInterface.(*ControlConnContext); ok {
			// 停止超时定时器
			if ctx.timeoutTimer != nil {
				ctx.timeoutTimer.Stop()
			}
			// 停止数据包定时器
			if ctx.packetTimer != nil {
				ctx.packetTimer.Stop()
			}

			// 如果有设备ID，先取出连接，判断是否是同一个连接才发布设备离线事件
			if ctx.deviceID != 0 {
				deviceID := ctx.deviceID
				// 取出存储的连接
				if existingConnInterface, exists := s.deviceConns.Load(deviceID); exists {
					if existingConn, ok := existingConnInterface.(gnet.Conn); ok {
						// 判断是否是同一个连接
						if existingConn == c {
							// 删除设备映射
							s.deviceConns.Delete(deviceID)
							// 停止定时任务
							s.scheduler.StopTask(deviceID)

							// 发布设备离线状态
							s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
								ID:           deviceID,
								Online:       0,
								Time:         time.Now(),
								ConnectionID: addr,
							})
							s.logger.Info(fmt.Sprintf("%s 清理设备映射: %d", s.name, deviceID))
						} else {
							s.logger.Info(fmt.Sprintf("%s 设备 %d 的连接已被替换，跳过清理", s.name, deviceID))
						}
					}
				}
			}
		}
	}

	s.logger.Info(fmt.Sprintf("%s 客户端断开: %s, 错误: %v", s.name, addr, err))
	return gnet.None
}

func (s *ControlService) OnTraffic(c gnet.Conn) gnet.Action {
	ctx, ok := c.Context().(*ControlConnContext)
	if !ok || ctx == nil {
		return gnet.Close
	}

	// 读取所有可用数据
	data, err := c.Next(-1)
	if err != nil {
		s.logger.Error(fmt.Sprintf("%s 读取数据错误: %v", s.name, err))
		return gnet.Close
	}
	if len(data) == 0 {
		return gnet.None
	}

	// 按16进制打印接收到的原始数据
	s.logger.Info(fmt.Sprintf("%s 接收到原始数据包 (设备ID: %d, 长度: %d字节) from %s: %x", s.name, ctx.deviceID, len(data), c.RemoteAddr().String(), data))

	// 将新数据写入缓冲区
	ctx.buffer.Write(data)

	// 获取设备ID
	deviceID := ctx.deviceID

	// 如果上下文中还没有设备ID，尝试解析
	if deviceID == 0 {
		// 尝试解析设备ID
		if parsedDeviceID, success := s.parseDeviceID(ctx.buffer.Bytes()); success {
			deviceID = parsedDeviceID
			// 更新连接上下文
			ctx.deviceID = deviceID
			// 停止超时定时器
			if ctx.timeoutTimer != nil {
				ctx.timeoutTimer.Stop()
				ctx.timeoutTimer = nil
			}

			// 检查是否已有相同设备ID的连接，如果有则先断开
			if existingConnInterface, exists := s.deviceConns.Load(deviceID); exists {
				if existingConn, ok := existingConnInterface.(gnet.Conn); ok {
					s.logger.Warn(fmt.Sprintf("%s 检测到设备 %d 已存在连接，断开旧连接", s.name, deviceID))
					// 停止调度器任务
					s.scheduler.StopTask(deviceID)
					// 关闭旧连接
					existingConn.Close()
				}
			}

			// 存储到设备映射中
			s.deviceConns.Store(deviceID, c)

			// 发布设备上线状态
			s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
				ID:           deviceID,
				Online:       1,
				Time:         time.Now(),
				ConnectionID: c.RemoteAddr().String(),
			})

			// 清空缓冲区，因为设备ID已经解析完成
			ctx.buffer.Reset()
		}
	}

	// 如果设备ID为0，说明还没有解析到设备ID，不执行后续逻辑
	if deviceID == 0 {
		return gnet.None
	}

	// 停止之前的定时器（如果存在）
	if ctx.packetTimer != nil {
		ctx.packetTimer.Stop()
	}

	// 启动新的200ms定时器
	ctx.packetTimer = time.AfterFunc(200*time.Millisecond, func() {
		// 处理完整的数据包
		s.processCompleteData(ctx)
	})

	return gnet.None
}

// publishEvent 发布事件
func (s *ControlService) publishEvent(eventType singletons.EventType, data interface{}) {
	// 使用单例事件发布器
	publisher := singletons.GetEventPublisher()
	event := singletons.Event{Type: eventType, Data: data}
	publisher.Publish(event)
}

// ================= 协议处理函数 =================

// 请求命令全局变量
var (
	// ID请求命令
	idRequest = []byte{0x00, 0x03, 0x00, 0x60, 0x00, 0x04, 0x45, 0xc6}
	// 数据请求命令
	dataRequest = []byte{0x00, 0x03, 0x00, 0x00, 0xf0, 0x24}
)

// calculateCRC 计算CRC16校验码
func (s *ControlService) calculateCRC(data []byte) (byte, byte) {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return byte(crc & 0xFF), byte(crc >> 8)
}

// buildReadIDCommand 构建读取ID命令
func (s *ControlService) buildReadIDCommand() []byte {
	return idRequest
}

// buildDataRequestCommand 构建数据请求命令
func (s *ControlService) buildDataRequestCommand() []byte {
	return dataRequest
}

// buildRelayControlCommand 构建继电器控制命令
func (s *ControlService) buildRelayControlCommand(relayIndex int, isOpen bool) []byte {
	cmd := make([]byte, 8)
	cmd[0] = 0x00             // 设备地址
	cmd[1] = 0x10             // 功能码
	cmd[2] = 0x00             // 起始位置高字节
	cmd[3] = 0x7A             // 起始位置低字节
	cmd[4] = byte(relayIndex) // 继电器索引
	cmd[5] = 0x01             // 继电器数量
	cmd[6] = 0x00             // 数据长度高字节
	cmd[7] = 0x02             // 数据长度低字节

	// 添加继电器状态数据
	status := byte(0x00)
	if isOpen {
		status = 0x01
	}
	cmd = append(cmd, status)

	// 计算CRC
	crc1, crc2 := s.calculateCRC(cmd)
	cmd = append(cmd, crc1, crc2)

	return cmd
}

// parseDeviceID 解析设备ID
func (s *ControlService) parseDeviceID(frame []byte) (uint32, bool) {
	if len(frame) != 12 || frame[1] != 0x03 || frame[2] != 0x00 || frame[3] != 0x60 {
		return 0, false
	}

	result, err := strconv.Atoi(fmt.Sprintf("%02d%02d%02d%02d", frame[6], frame[7], frame[8], frame[9]))
	if err != nil {
		s.logger.Error(fmt.Sprintf("解析设备ID失败: %v", err))
		return 0, false
	}

	return uint32(result), true
}

// parseSensorData 解析传感器数据
func (s *ControlService) parseSensorData(frame []byte) ([]byte, bool) {
	if len(frame) != 70 || frame[1] != 0x03 || frame[2] != 0x00 || frame[3] != 0x40 {
		return nil, false
	}

	return frame[4:68], true
}

// parseRelayControl 解析继电器控制
func (s *ControlService) parseRelayControl(frame []byte) (byte, byte, bool) {
	if len(frame) != 8 || frame[1] != 0x10 || frame[2] != 0x00 || frame[3] != 0x7a {
		return 0, 0, false
	}

	relayIndex := frame[4]
	relayStatus := frame[5]

	return relayIndex, relayStatus, true
}

// ================= 任务管理函数 =================

// startDataReadingTask 启动数据读取任务
func (s *ControlService) startDataReadingTask(deviceID uint32, conn gnet.Conn, intervalMinutes int) {
	// 使用调度器启动定时任务
	task := func() {
		// 发送数据请求命令
		dataCmd := s.buildDataRequestCommand()
		if _, err := conn.Write(dataCmd); err != nil {
			s.logger.Error(fmt.Sprintf("%s 发送数据请求命令失败 (设备ID: %d): %v", s.name, deviceID, err))
		} else {
			s.logger.Info(fmt.Sprintf("%s 已发送数据请求命令 (设备ID: %d)", s.name, deviceID))
		}
	}

	// 启动定时任务
	s.scheduler.StartTask(deviceID, intervalMinutes, task)
	s.logger.Info(fmt.Sprintf("%s 已启动数据读取任务 (设备ID: %d, 间隔: %d分钟)", s.name, deviceID, intervalMinutes))
}

// ================= 控制接口实现 =================

// RelaySwitch 控制继电器开关状态
func (s *ControlService) RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) (error, bool) {
	// 查找设备连接
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return fmt.Errorf("设备 %d 未连接", deviceID), false
	}

	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return fmt.Errorf("设备 %d 连接无效", deviceID), false
	}

	// 构建继电器控制命令
	relayCmd := s.buildRelayControlCommand(relayIndex, isOpen)

	// 发送控制命令
	if _, err := conn.Write(relayCmd); err != nil {
		s.logger.Error(fmt.Sprintf("%s 发送继电器控制命令失败 (设备ID: %d, 继电器: %d, 状态: %v): %v",
			s.name, deviceID, relayIndex, isOpen, err))
		return err, true
	}

	s.logger.Info(fmt.Sprintf("%s 已发送继电器控制命令 (设备ID: %d, 继电器: %d, 状态: %v)",
		s.name, deviceID, relayIndex, isOpen))
	return nil, true
}

// SetDeviceReadInterval 设置特定设备的数据读取间隔（分钟）
func (s *ControlService) SetDeviceReadInterval(deviceID uint32, intervalMinutes int) (error, bool) {
	// 查找设备连接
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return fmt.Errorf("设备 %d 未连接", deviceID), false
	}

	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return fmt.Errorf("设备 %d 连接无效", deviceID), false
	}

	// 停止现有任务
	s.scheduler.StopTask(deviceID)

	// 启动新的定时任务
	s.startDataReadingTask(deviceID, conn, intervalMinutes)

	s.logger.Info(fmt.Sprintf("%s 已更新设备读取间隔 (设备ID: %d, 间隔: %d分钟)", s.name, deviceID, intervalMinutes))
	return nil, true
}

// DisconnectOtherConnections 断开同一设备ID的其他连接
func (s *ControlService) DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64) {
	// 查找对应的连接
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return // 服务存在但设备未连接，直接返回
	}

	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return // 服务存在但连接无效，直接返回
	}

	// 获取连接上下文
	ctxInterface := conn.Context()
	if ctxInterface == nil {
		return // 连接上下文无效，直接返回
	}

	ctx, ok := ctxInterface.(*ControlConnContext)
	if !ok {
		return // 连接上下文类型错误，直接返回
	}

	// 如果是同一个连接，跳过
	if ctx.connectionID == currentConnectionID {
		s.logger.Info(fmt.Sprintf("%s 跳过断开设备 %d 的连接 %s（当前连接）", s.name, deviceID, currentConnectionID))
		return
	}

	// 检查时间，如果当前连接时间更新，则断开旧连接
	if currentTime > ctx.connectionTime {
		s.logger.Warn(fmt.Sprintf("%s 检测到设备 %d 存在更新的连接，断开旧连接: %s", s.name, deviceID, ctx.connectionID))

		// 停止调度器任务
		s.scheduler.StopTask(deviceID)

		// 关闭连接
		conn.Close()
		s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
		return
	}

	// 如果旧连接时间更新，则跳过断开
	s.logger.Info(fmt.Sprintf("%s 跳过断开设备 %d 的连接 %s（旧连接时间更新）", s.name, deviceID, ctx.connectionID))
}

// processCompleteData 处理完整的数据包
func (s *ControlService) processCompleteData(ctx *ControlConnContext) {
	// 获取设备ID
	deviceID := ctx.deviceID

	// 如果设备ID为0，说明还没有解析到设备ID，不执行后续逻辑
	if deviceID == 0 {
		return
	}

	// 尝试解析传感器数据
	if sensorData, success := s.parseSensorData(ctx.buffer.Bytes()); success {
		// 发布传感器数据事件
		s.publishEvent(singletons.EventData, &singletons.DeviceData{
			ID:       deviceID,
			Type:     1, // 控制服务设备类型
			LastSeen: time.Now(),
			Data:     sensorData,
		})
	} else {
		// 尝试解析继电器控制响应
		if relayIndex, relayStatus, success := s.parseRelayControl(ctx.buffer.Bytes()); success {
			// 计算操作成功类型
			status := byte(0)
			if relayStatus == 1 {
				status = 1
			}

			// 发布继电器状态事件（操作成功）
			s.publishEvent(singletons.EventRelayStatus, &singletons.RelayStatus{
				ID:       deviceID,
				RelayNum: uint32(relayIndex), // relayIndex是0-31
				State:    status,
			})
			s.logger.Info(fmt.Sprintf("%s 收到继电器控制响应 - 设备ID: %d, 继电器: %d, 状态: %d",
				s.name, deviceID, relayIndex, status))
		} else {
			// 打印未知数据
			s.logger.Warn(fmt.Sprintf("%s 收到未知数据 (设备ID: %d): %x", s.name, deviceID, ctx.buffer.Bytes()))
		}
	}

	// 清空数据缓冲区
	ctx.buffer.Reset()
}
