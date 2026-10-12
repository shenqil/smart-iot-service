package activereport

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

// ActiveReportConnContext 连接上下文
type ActiveReportConnContext struct {
	deviceID       uint32        // 设备ID
	timeoutTimer   *time.Timer   // 超时定时器
	buffer         *bytes.Buffer // 数据缓冲区，用于处理粘包和分包
	connectionID   string        // 连接ID
	connectionTime int64         // 连接时间戳
}

// ActiveReportService 主动上报服务
type ActiveReportService struct {
	gnet.BuiltinEventEngine
	eng         gnet.Engine
	addr        string
	ctx         context.Context
	cancel      context.CancelFunc
	name        string
	logger      *slog.Logger // 服务专用日志器
	deviceConns sync.Map     // map[uint32]gnet.Conn - 设备ID到连接的映射
}

// NewActiveReportService 创建新的主动上报服务
func NewActiveReportService(addr string) *ActiveReportService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ActiveReportService{
		addr:   addr,
		ctx:    ctx,
		cancel: cancel,
		name:   "主动上报服务",
		logger: logger.NewServiceLogger("active_report_service"),
	}
}

// GetName 获取服务名称
func (s *ActiveReportService) GetName() string {
	return s.name
}

// Start 启动服务
func (s *ActiveReportService) Start() error {
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
func (s *ActiveReportService) Stop() error {
	s.logger.Info(fmt.Sprintf("停止 %s", s.name))

	// 取消上下文
	s.cancel()

	// 清空连接映射
	s.deviceConns = sync.Map{}

	s.logger.Info(fmt.Sprintf("%s 已停止", s.name))
	return nil
}

// DisconnectedDevice 断开某个设备连接
func (s *ActiveReportService) DisconnectedDevice(deviceID uint32) bool {
	// 查找对应的连接
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return false // 服务存在但设备未连接，返回不存在
	}

	conn, ok := connInterface.(gnet.Conn)
	if !ok {
		return false // 服务存在但连接无效，返回不存在
	}

	// 关闭连接
	conn.Close()
	s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
	return true // 服务存在并已处理
}

// RelaySwitch 控制继电器开关状态
func (s *ActiveReportService) RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) (error, bool) {
	// 检查设备是否连接到此服务
	_, exists := s.deviceConns.Load(deviceID)
	if exists {
		// 设备连接到此服务，但不处理继电器控制
		s.logger.Info(fmt.Sprintf("主动上报服务: 设备 %d 连接到此服务，但不处理继电器控制", deviceID))
		return nil, true // 服务存在但不处理此功能
	}
	return nil, false // 设备未连接到此服务
}

// SetDeviceReadInterval 设置特定设备的数据读取间隔（分钟）
func (s *ActiveReportService) SetDeviceReadInterval(deviceID uint32, intervalMinutes int) (error, bool) {
	// 检查设备是否连接到此服务
	_, exists := s.deviceConns.Load(deviceID)
	if exists {
		// 设备连接到此服务，但不处理间隔设置
		s.logger.Info(fmt.Sprintf("主动上报服务: 设备 %d 连接到此服务，但不处理间隔设置", deviceID))
		return nil, true // 服务存在但不处理此功能
	}
	return nil, false // 设备未连接到此服务
}

// gnet事件处理
func (s *ActiveReportService) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	s.logger.Info(fmt.Sprintf("%s 服务启动，监听地址: %s", s.name, s.addr))
	return gnet.None
}

func (s *ActiveReportService) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	ctx := &ActiveReportConnContext{
		deviceID:       0, // 初始设备ID为0，等待设备响应
		timeoutTimer:   nil,
		buffer:         bytes.NewBuffer(make([]byte, 0, 200)), // 初始化缓冲区，容量为200字节
		connectionID:   c.RemoteAddr().String(),               // 使用远程地址作为连接ID
		connectionTime: time.Now().Unix(),                     // 当前时间戳
	}

	c.SetContext(ctx)

	s.logger.Info(fmt.Sprintf("%s 客户端连接: %s", s.name, c.RemoteAddr().String()))

	// 启动1小时超时定时器，如果1小时内没有收到设备ID，则关闭连接
	ctx.timeoutTimer = time.AfterFunc(1*time.Hour, func() {
		if ctx.deviceID == 0 {
			s.logger.Warn(fmt.Sprintf("%s 连接 %s 在1小时内未收到设备ID，主动关闭连接", s.name, c.RemoteAddr().String()))
			c.Close()
		}
	})

	return nil, gnet.None
}

func (s *ActiveReportService) OnClose(c gnet.Conn, err error) gnet.Action {
	addr := c.RemoteAddr().String()

	// 清理连接上下文和定时器，获取设备ID并清理
	if ctxInterface := c.Context(); ctxInterface != nil {
		if ctx, ok := ctxInterface.(*ActiveReportConnContext); ok {
			// 停止超时定时器
			if ctx.timeoutTimer != nil {
				ctx.timeoutTimer.Stop()
			}

			// 清理缓冲区
			ctx.buffer.Reset()

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

func (s *ActiveReportService) OnTraffic(c gnet.Conn) gnet.Action {
	ctx, ok := c.Context().(*ActiveReportConnContext)
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

	// 处理缓冲区中的数据，处理粘包和分包
	if ctx.buffer.Len() >= 72 { // 传感器数据包长度为72字节
		var packetData []byte
		var dataProcessed int

		if ctx.deviceID != 0 {
			// 当设备ID已知时，查找数据包开头并匹配deviceID
			packetData, dataProcessed = s.findPacketWithDeviceID(ctx.buffer, ctx.deviceID)
		} else {
			// 当设备ID未知时，直接读取前72字节
			packetData = ctx.buffer.Bytes()[:72]
			dataProcessed = 72
		}

		if packetData != nil {
			// 从缓冲区中移除已处理的数据
			ctx.buffer.Next(dataProcessed)
			deviceID, sensorData, _, success := s.parseReportedSensorData(packetData)

			if success {
				// 如果上下文中还没有设备ID，执行设备上线逻辑
				if ctx.deviceID == 0 {
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
				}

				// 发布传感器数据事件
				s.publishEvent(singletons.EventData, &singletons.DeviceData{
					ID:       deviceID,
					Type:     1, // 主动上报设备类型
					LastSeen: time.Now(),
					Data:     sensorData,
				})
			} else {
				// 数据校验失败，直接丢弃数据包
				s.logger.Warn(fmt.Sprintf("%s 数据校验失败，丢弃数据包: %x", s.name, packetData))
			}
		}
	}

	// 如果缓冲区过大，截断到合理大小（保留最后72字节）
	if ctx.buffer.Len() > 72 {
		// 保留最后72字节
		remainingData := ctx.buffer.Bytes()[ctx.buffer.Len()-72:]
		ctx.buffer.Reset()
		ctx.buffer.Write(remainingData)
	}

	return gnet.None
}

// publishEvent 发布事件
func (s *ActiveReportService) publishEvent(eventType singletons.EventType, data interface{}) {
	// 使用单例事件发布器
	publisher := singletons.GetEventPublisher()
	event := singletons.Event{Type: eventType, Data: data}
	publisher.Publish(event)
}

// calculateCRC 计算CRC16校验码
func (s *ActiveReportService) calculateCRC(data []byte) (byte, byte) {
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

// verifyCRC 验证CRC16校验码
func (s *ActiveReportService) verifyCRC(data []byte, crcLow byte, crcHigh byte) bool {
	calcLow, calcHigh := s.calculateCRC(data)
	return calcLow == crcLow && calcHigh == crcHigh
}

// findPacketWithDeviceID 在缓冲区中查找匹配指定deviceID的数据包开头
func (s *ActiveReportService) findPacketWithDeviceID(buffer *bytes.Buffer, targetDeviceID uint32) ([]byte, int) {
	bufferData := buffer.Bytes()
	bufferLen := len(bufferData)

	// 如果缓冲区长度不足72字节，直接返回
	if bufferLen < 72 {
		return nil, 0
	}

	// 将目标deviceID转换为字节数组，用于快速比较
	targetBytes := []byte{
		byte(targetDeviceID >> 24),
		byte(targetDeviceID >> 16),
		byte(targetDeviceID >> 8),
		byte(targetDeviceID),
	}

	// 使用优化的模式匹配算法
	return s.findPatternInBuffer(bufferData, targetBytes, 72)
}

// findPatternInBuffer 在缓冲区中查找指定模式，返回匹配的数据包
func (s *ActiveReportService) findPatternInBuffer(bufferData []byte, pattern []byte, packetSize int) ([]byte, int) {
	bufferLen := len(bufferData)

	// 如果缓冲区长度不足数据包大小，直接返回
	if bufferLen < packetSize {
		return nil, 0
	}

	// 使用Go标准库的bytes.Index进行高效搜索
	searchEnd := bufferLen - packetSize + 1
	if searchEnd <= 0 {
		return nil, 0
	}

	// 在可搜索范围内查找模式
	searchData := bufferData[:searchEnd]
	index := bytes.Index(searchData, pattern)

	if index != -1 {
		// 找到匹配的模式，返回完整数据包
		return bufferData[index : index+packetSize], index + packetSize
	}

	return nil, 0
}

// parseReportedSensorData 解析传感器上报数据
func (s *ActiveReportService) parseReportedSensorData(frame []byte) (uint32, []byte, []byte, bool) {
	// 检查数据包长度（72字节：4字节设备ID + 2字节命令 + 64字节数据 + 2字节CRC）
	if len(frame) != 72 {
		s.logger.Error(fmt.Sprintf("%s 数据包长度不正确，期望72字节，实际%d字节", s.name, len(frame)))
		return 0, nil, nil, false
	}

	// 检查命令码
	if frame[4] != 0xa2 || frame[5] != 0x40 {
		s.logger.Error(fmt.Sprintf("%s 命令码不正确，期望0xa2 0x40，实际0x%02x 0x%02x", s.name, frame[4], frame[5]))
		return 0, nil, nil, false
	}

	// 验证CRC校验码（对前70字节进行校验：4字节设备ID + 2字节命令 + 64字节数据）
	crcData := frame[:70]
	receivedCRCLow := frame[70]
	receivedCRCHigh := frame[71]

	if !s.verifyCRC(crcData, receivedCRCLow, receivedCRCHigh) {
		s.logger.Error(fmt.Sprintf("%s CRC校验失败，数据包被丢弃", s.name))
		return 0, nil, nil, false
	}

	// 解析设备ID (前4个字节，使用字符串拼接方式)
	result, err := strconv.Atoi(fmt.Sprintf("%02d%02d%02d%02d", frame[0], frame[1], frame[2], frame[3]))
	if err != nil {
		s.logger.Error(fmt.Sprintf("%s 设备ID解析失败: %v", s.name, err))
		return 0, nil, nil, false
	}
	deviceID := uint32(result)

	// 返回设备ID、传感器数据(第6-66字节)、空响应命令
	return deviceID, frame[6:66], nil, true
}

// DisconnectOtherConnections 断开同一设备ID的其他连接
func (s *ActiveReportService) DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64) {
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

	ctx, ok := ctxInterface.(*ActiveReportConnContext)
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

		// 关闭连接
		conn.Close()
		s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
		return
	}

	// 如果旧连接时间更新，则跳过断开
	s.logger.Info(fmt.Sprintf("%s 跳过断开设备 %d 的连接 %s（旧连接时间更新）", s.name, deviceID, ctx.connectionID))
}
