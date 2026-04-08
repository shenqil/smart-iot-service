package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"rbac-admin/pkg/collector/protocols"
	"rbac-admin/pkg/collector/singletons"
	"rbac-admin/pkg/logger"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2"
)

// IoTConnContext 连接上下文
type IoTConnContext struct {
	buffer         *bytes.Buffer // 数据缓冲区
	deviceID       uint32        // 设备ID
	idReceived     bool          // 是否已收到设备ID
	readIDTimer    *time.Timer   // 读取ID的超时定时器（10秒内未收到响应则断开连接）
	connectionID   string        // 连接ID
	connectionTime int64         // 连接时间戳
}

// IoTService IoT协议服务
type IoTService struct {
	gnet.BuiltinEventEngine
	addr   string
	ctx    context.Context
	cancel context.CancelFunc
	name   string       // 服务名称
	logger *slog.Logger // 服务专用日志器

	deviceConns     sync.Map                      // map[uint32]gnet.Conn - 设备ID到连接的映射
	protocolHandler *protocols.IoTProtocolHandler // 协议处理器
}

// NewIoTService 创建新的IoT协议服务
func NewIoTService(addr string) *IoTService {
	ctx, cancel := context.WithCancel(context.Background())
	return &IoTService{
		addr:            addr,
		ctx:             ctx,
		cancel:          cancel,
		name:            "IoT协议服务",
		logger:          logger.NewServiceLogger("iot_service"),
		protocolHandler: protocols.NewIoTProtocolHandler(),
	}
}

// GetName 获取服务名称
func (s *IoTService) GetName() string {
	return s.name
}

// Start 启动服务
func (s *IoTService) Start() error {
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
func (s *IoTService) Stop() error {
	s.logger.Info(fmt.Sprintf("停止 %s", s.name))

	// 取消上下文
	s.cancel()

	// 关闭所有设备连接并清理映射
	var deviceIDs []uint32
	s.deviceConns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(gnet.Conn); ok {
			conn.Close()
			deviceIDs = append(deviceIDs, key.(uint32))
		}
		return true
	})

	// 从映射中删除所有设备
	for _, deviceID := range deviceIDs {
		s.deviceConns.Delete(deviceID)
	}

	s.logger.Info(fmt.Sprintf("%s 已停止，清理了 %d 个设备连接", s.name, len(deviceIDs)))
	return nil
}

// getDeviceConn 获取设备连接
func (s *IoTService) getDeviceConn(deviceID uint32) (gnet.Conn, bool) {
	connInterface, exists := s.deviceConns.Load(deviceID)
	if !exists {
		return nil, false
	}
	conn, ok := connInterface.(gnet.Conn)
	return conn, ok
}

// DisconnectedDevice 断开某个设备连接
func (s *IoTService) DisconnectedDevice(deviceID uint32) bool {
	conn, ok := s.getDeviceConn(deviceID)
	if !ok {
		return false
	}
	conn.Close()
	s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
	return true
}

// RelaySwitch 控制继电器开关状态
func (s *IoTService) RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) (error, bool) {
	if relayIndex < 0 || relayIndex > 31 {
		return fmt.Errorf("无效的继电器索引: %d，必须在0-31范围内", relayIndex), true
	}

	conn, ok := s.getDeviceConn(deviceID)
	if !ok {
		return nil, false
	}

	cmd := s.protocolHandler.BuildRelayOperationCommand(relayIndex, isOpen)
	action := "关闭"
	if isOpen {
		action = "开启"
	}

	err := conn.AsyncWrite(cmd, func(c gnet.Conn, err error) error {
		if err != nil {
			s.logger.Error(fmt.Sprintf("%s 发送继电器操作命令失败: %v", s.name, err))
			return err
		}
		s.logger.Info(fmt.Sprintf("%s 发送继电器操作命令: 设备%d, 继电器%d %s", s.name, deviceID, relayIndex, action))
		return nil
	})

	if err != nil {
		return fmt.Errorf("发送继电器操作命令失败: %v", err), true
	}
	return nil, true
}

// SetDeviceReadInterval 设置特定设备的数据读取间隔（分钟）
func (s *IoTService) SetDeviceReadInterval(deviceID uint32, intervalMinutes int) (error, bool) {
	if intervalMinutes <= 0 {
		return fmt.Errorf("间隔分钟数必须 > 0"), true
	}

	conn, ok := s.getDeviceConn(deviceID)
	if !ok {
		return nil, false
	}

	cmd := s.protocolHandler.BuildSetReportIntervalCommand(uint16(intervalMinutes))
	err := conn.AsyncWrite(cmd, func(c gnet.Conn, err error) error {
		if err != nil {
			s.logger.Error(fmt.Sprintf("%s 发送设置上报间隔命令失败: %v", s.name, err))
			return err
		}
		s.logger.Info(fmt.Sprintf("%s 设置设备 %d 上报间隔为 %d 分钟", s.name, deviceID, intervalMinutes))
		return nil
	})

	if err != nil {
		return fmt.Errorf("发送设置上报间隔命令失败: %v", err), true
	}
	return nil, true
}

// OnBoot gnet事件处理 - 服务启动
func (s *IoTService) OnBoot(eng gnet.Engine) gnet.Action {
	s.logger.Info(fmt.Sprintf("%s 服务启动，监听地址: %s", s.name, s.addr))
	return gnet.None
}

// OnOpen gnet事件处理 - 连接建立
func (s *IoTService) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	addr := c.RemoteAddr().String()
	ctx := &IoTConnContext{
		buffer:         bytes.NewBuffer(nil),
		deviceID:       0,
		idReceived:     false,
		readIDTimer:    nil,
		connectionID:   addr,
		connectionTime: time.Now().Unix(),
	}
	c.SetContext(ctx)

	s.logger.Info(fmt.Sprintf("%s 客户端连接: %s", s.name, addr))

	// 发送读取ID指令并启动超时定时器（10秒内未收到响应则断开连接）
	readIDCmd := s.protocolHandler.BuildReadIDCommand()
	ctx.readIDTimer = time.AfterFunc(10*time.Second, func() {
		s.handleReadIDTimeout(c)
	})

	s.logger.Info(fmt.Sprintf("%s 发送读取ID指令到: %s, 数据: %x", s.name, addr, readIDCmd))

	// 直接返回数据，让 gnet 自动发送
	return readIDCmd, gnet.None
}

// OnClose gnet事件处理 - 连接关闭
func (s *IoTService) OnClose(c gnet.Conn, err error) gnet.Action {
	addr := c.RemoteAddr().String()
	ctx, ok := c.Context().(*IoTConnContext)
	if !ok || ctx == nil {
		s.logger.Info(fmt.Sprintf("%s 客户端断开: %s, 错误: %v", s.name, addr, err))
		return gnet.None
	}

	// 清理定时器
	if ctx.readIDTimer != nil {
		ctx.readIDTimer.Stop()
	}
	// 清理缓冲区
	if ctx.buffer != nil {
		ctx.buffer.Reset()
	}

	// 如果设备ID有效，检查并清理设备映射
	if ctx.deviceID != 0 {
		if existingConn, exists := s.getDeviceConn(ctx.deviceID); exists && existingConn == c {
			s.deviceConns.Delete(ctx.deviceID)
			s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
				ID:           ctx.deviceID,
				Online:       0,
				Time:         time.Now(),
				ConnectionID: addr,
			})
			s.logger.Info(fmt.Sprintf("%s 清理设备映射: %d", s.name, ctx.deviceID))
		} else if exists {
			s.logger.Info(fmt.Sprintf("%s 设备 %d 的连接已被替换，跳过清理", s.name, ctx.deviceID))
		}
	}

	s.logger.Info(fmt.Sprintf("%s 客户端断开: %s, 错误: %v", s.name, addr, err))
	return gnet.None
}

// OnTraffic gnet事件处理 - 数据接收
func (s *IoTService) OnTraffic(c gnet.Conn) gnet.Action {
	addr := c.RemoteAddr().String()
	s.logger.Debug(fmt.Sprintf("%s OnTraffic 被触发 from %s", s.name, addr))

	ctx, ok := c.Context().(*IoTConnContext)
	if !ok || ctx == nil {
		s.logger.Error(fmt.Sprintf("%s OnTraffic: 上下文无效 from %s", s.name, addr))
		return gnet.Close
	}

	// 读取所有可用数据
	data, err := c.Next(-1)
	if err != nil {
		s.logger.Error(fmt.Sprintf("%s 读取数据错误: %v from %s", s.name, err, addr))
		return gnet.Close
	}
	if len(data) == 0 {
		s.logger.Debug(fmt.Sprintf("%s OnTraffic: 读取到空数据 from %s", s.name, addr))
		return gnet.None
	}

	s.logger.Info(fmt.Sprintf("%s 接收到原始数据包 (设备ID: %d, 长度: %d字节) from %s: %x",
		s.name, ctx.deviceID, len(data), addr, data))

	// 检查缓冲区大小，防止内存耗尽（10KB限制）
	if ctx.buffer.Len()+len(data) > 10*1024 {
		s.logger.Warn(fmt.Sprintf("%s 缓冲区溢出，关闭连接: %s", s.name, addr))
		return gnet.Close
	}

	ctx.buffer.Write(data)

	// 处理缓冲区中的完整数据包
	processed := 0
	for {
		packet, isValid, shouldContinue := s.protocolHandler.ExtractCompletePacket(ctx.buffer.Bytes())
		if !shouldContinue {
			break
		}

		ctx.buffer.Next(len(packet))
		if !isValid {
			s.logger.Debug(fmt.Sprintf("%s 过滤无效数据包，长度: %d from %s", s.name, len(packet), addr))
			continue
		}

		cmd, data, err := s.protocolHandler.ParsePacket(packet)
		if err != nil {
			s.logger.Error(fmt.Sprintf("%s 解析数据包失败: %v", s.name, err))
			continue
		}

		// 处理不同类型的命令
		switch cmd {
		case protocols.CMD_READ_ID:
			s.handleReadIDResponse(c, ctx, data)
		case protocols.CMD_SET_REPORT_INTERVAL:
			s.handleSetReportIntervalResponse(ctx, data)
		case protocols.CMD_CONTROL_RELAY:
			s.handleControlRelay(ctx, data)
		case protocols.CMD_DATA_REPORT:
			s.handleDataReport(c, ctx, data)
		default:
			s.logger.Warn(fmt.Sprintf("%s 未知命令: 0x%02X from %s", s.name, cmd, addr))
		}

		processed++
		if processed > 1000 {
			s.logger.Warn(fmt.Sprintf("%s 单次事件处理数据包过多: %s", s.name, addr))
			break
		}
	}

	// 如果缓冲区仍然很大，记录警告（8KB警告阈值）
	if ctx.buffer.Len() > 8*1024 {
		s.logger.Warn(fmt.Sprintf("%s 缓冲区仍然较大: %d字节，可能存在未处理的数据 from %s",
			s.name, ctx.buffer.Len(), addr))
	}

	return gnet.None
}

// handleReadIDResponse 处理读取设备ID响应
func (s *IoTService) handleReadIDResponse(c gnet.Conn, ctx *IoTConnContext, data []byte) {
	if ctx.idReceived {
		s.logger.Debug(fmt.Sprintf("%s 设备ID已收到，丢弃后续响应 from %s", s.name, c.RemoteAddr().String()))
		return
	}

	deviceID, ok := s.protocolHandler.ParseDeviceIDResponse(data)
	if !ok {
		s.logger.Warn(fmt.Sprintf("%s 设备ID数据解析失败", s.name))
		return
	}

	s.logger.Info(fmt.Sprintf("%s 收到设备ID响应: %d from %s", s.name, deviceID, c.RemoteAddr().String()))

	ctx.idReceived = true
	ctx.deviceID = deviceID

	// 停止超时定时器
	if ctx.readIDTimer != nil {
		ctx.readIDTimer.Stop()
		ctx.readIDTimer = nil
	}

	// 如果已有相同设备ID的连接，断开旧连接
	if existingConn, exists := s.getDeviceConn(deviceID); exists {
		s.logger.Warn(fmt.Sprintf("%s 检测到设备 %d 已存在连接，断开旧连接", s.name, deviceID))
		existingConn.Close()
	}

	// 建立设备ID到连接的映射
	s.deviceConns.Store(deviceID, c)

	// 发布设备在线状态事件
	s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
		ID:           deviceID,
		Online:       1,
		Time:         time.Now(),
		ConnectionID: c.RemoteAddr().String(),
	})
}

// handleDataReport 处理设备主动上报数据
func (s *IoTService) handleDataReport(c gnet.Conn, ctx *IoTConnContext, data []byte) {
	if ctx.deviceID == 0 {
		s.logger.Warn(fmt.Sprintf("%s 收到数据上报但设备ID无效，丢弃数据 from %s", s.name, c.RemoteAddr().String()))
		return
	}

	if !s.protocolHandler.ParseDataReport(data) {
		s.logger.Warn(fmt.Sprintf("%s 设备数据上报解析失败", s.name))
		return
	}

	s.logger.Info(fmt.Sprintf("%s 收到设备数据上报: 设备ID=%d, 数据长度=%d字节", s.name, ctx.deviceID, len(data)))

	s.publishEvent(singletons.EventData, &singletons.DeviceData{
		ID:       ctx.deviceID,
		Type:     1, // IoT协议设备类型
		LastSeen: time.Now(),
		Data:     data,
	})
}

// handleControlRelay 处理控制继电器响应
func (s *IoTService) handleControlRelay(ctx *IoTConnContext, data []byte) {
	if ctx.deviceID == 0 {
		s.logger.Warn(fmt.Sprintf("%s 收到继电器控制响应但设备ID无效，丢弃响应", s.name))
		return
	}

	relayIndex, relayState := s.protocolHandler.ParseRelayOperationResponse(data)
	status := byte(0)
	if relayState == protocols.RELAY_ON {
		status = 1
	}

	s.publishEvent(singletons.EventRelayStatus, &singletons.RelayStatus{
		ID:       ctx.deviceID,
		RelayNum: uint32(relayIndex),
		State:    status,
	})

	s.logger.Info(fmt.Sprintf("%s 收到继电器控制响应 - 设备ID: %d, 继电器: %d, 状态: %d",
		s.name, ctx.deviceID, relayIndex, status))
}

// handleSetReportIntervalResponse 处理设置上报间隔响应
func (s *IoTService) handleSetReportIntervalResponse(ctx *IoTConnContext, data []byte) {
	if ctx.deviceID == 0 {
		s.logger.Warn(fmt.Sprintf("%s 收到设置上报间隔响应但设备ID无效，丢弃响应", s.name))
		return
	}

	intervalMinutes, ok := s.protocolHandler.ParseSetReportIntervalResponse(data)
	if !ok {
		s.logger.Warn(fmt.Sprintf("%s 设置上报间隔响应解析失败", s.name))
		return
	}

	s.logger.Info(fmt.Sprintf("%s 收到设置上报间隔响应: %d分钟 from %d", s.name, intervalMinutes, ctx.deviceID))
}

// handleReadIDTimeout 处理读取ID超时，断开连接
func (s *IoTService) handleReadIDTimeout(c gnet.Conn) {
	ctx, ok := c.Context().(*IoTConnContext)
	if !ok || ctx == nil {
		return
	}

	// 如果已经收到设备ID，不需要断开连接
	if ctx.idReceived {
		return
	}

	s.logger.Warn(fmt.Sprintf("%s 读取设备ID超时（10秒内未收到响应），断开连接: %s", s.name, c.RemoteAddr().String()))
	c.Close()
}

// publishEvent 发布事件
func (s *IoTService) publishEvent(eventType singletons.EventType, data interface{}) {
	// 使用单例事件发布器
	publisher := singletons.GetEventPublisher()
	event := singletons.Event{Type: eventType, Data: data}
	publisher.Publish(event)
}

// DisconnectOtherConnections 断开同一设备ID的其他连接
func (s *IoTService) DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64) {
	conn, ok := s.getDeviceConn(deviceID)
	if !ok {
		return
	}

	ctxInterface := conn.Context()
	if ctxInterface == nil {
		return
	}

	ctx, ok := ctxInterface.(*IoTConnContext)
	if !ok {
		return
	}

	// 如果是同一个连接，跳过
	if ctx.connectionID == currentConnectionID {
		s.logger.Info(fmt.Sprintf("%s 跳过断开设备 %d 的连接 %s（当前连接）", s.name, deviceID, currentConnectionID))
		return
	}

	// 如果当前连接时间更新，则断开旧连接
	if currentTime > ctx.connectionTime {
		s.logger.Warn(fmt.Sprintf("%s 检测到设备 %d 存在更新的连接，断开旧连接: %s", s.name, deviceID, ctx.connectionID))
		conn.Close()
		s.logger.Info(fmt.Sprintf("%s 主动断开设备连接: %d", s.name, deviceID))
		return
	}

	s.logger.Info(fmt.Sprintf("%s 跳过断开设备 %d 的连接 %s（旧连接时间更新）", s.name, deviceID, ctx.connectionID))
}
