package protocols

import (
	"encoding/binary"
	"fmt"
)

// 协议常量定义
const (
	STX = 0xAA
	ETX = 0x55
)

// 命令类型定义
const (
	CMD_READ_ID             = 0x01 // 读取设备ID
	CMD_CONTROL_RELAY       = 0x02 // 控制继电器
	CMD_SET_REPORT_INTERVAL = 0x03 // 设置数据上报间隔
	CMD_DATA_REPORT         = 0x04 // 设备主动上报数据
)

// 继电器操作定义
const (
	RELAY_OFF = 0x00
	RELAY_ON  = 0x01
)

// IoTProtocolHandler IoT协议处理器
type IoTProtocolHandler struct{}

// NewIoTProtocolHandler 创建新的IoT协议处理器
func NewIoTProtocolHandler() *IoTProtocolHandler {
	return &IoTProtocolHandler{}
}

// calculateModbusCRC 计算Modbus CRC16校验码
func (h *IoTProtocolHandler) calculateModbusCRC(data []byte) (byte, byte) {
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
func (h *IoTProtocolHandler) verifyCRC(data []byte, crcLow byte, crcHigh byte) bool {
	calcLow, calcHigh := h.calculateModbusCRC(data)
	return calcLow == crcLow && calcHigh == crcHigh
}

// ExtractCompletePacket 从缓冲区提取完整数据包
// 返回 (packet, isValid, shouldContinue)
// isValid: 数据包是否有效
// shouldContinue: 是否应该继续处理下一个包
func (h *IoTProtocolHandler) ExtractCompletePacket(buffer []byte) ([]byte, bool, bool) {
	if len(buffer) == 0 {
		return nil, false, false
	}

	// 查找起始符
	startIndex := -1
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == STX {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		// 没有找到起始符，清空整个缓冲区
		return buffer, false, false
	}

	// 移除起始符之前的所有无效数据
	if startIndex > 0 {
		// 返回起始符之前的所有无效数据，让上层过滤掉
		invalidData := make([]byte, startIndex)
		copy(invalidData, buffer[:startIndex])
		return invalidData, false, true
	}

	// 检查最小包长度 (STX+CMD+LEN(2)+CRC_L+CRC_H+ETX=7字节)
	if len(buffer) < 7 {
		return nil, false, false
	}

	// 获取数据长度（2字节，大端序）
	dataLen := int(binary.BigEndian.Uint16(buffer[2:4]))

	// 检查数据长度是否合理（防止异常大的数据包）
	if dataLen > 1024 { // 限制数据长度最大1KB
		// 数据长度异常，返回这个无效包让上层过滤
		packet := make([]byte, 1) // 只返回起始符
		copy(packet, buffer[:1])
		return packet, false, true
	}

	totalLen := 7 + dataLen // STX(1) + CMD(1) + LEN(2) + DATA(n) + CRC_L(1) + CRC_H(1) + ETX(1)

	// 检查是否收到完整数据包
	if len(buffer) < totalLen {
		return nil, false, false
	}

	// 检查结束符
	if buffer[totalLen-1] != ETX {
		// 结束符不匹配，返回这个无效包让上层过滤
		packet := make([]byte, totalLen)
		copy(packet, buffer[:totalLen])
		return packet, false, true
	}

	// 提取完整数据包
	packet := make([]byte, totalLen)
	copy(packet, buffer[:totalLen])

	// 验证CRC
	crcData := packet[1 : totalLen-3] // 从CMD到DATA结束
	crcLow := packet[totalLen-3]
	crcHigh := packet[totalLen-2]
	if !h.verifyCRC(crcData, crcLow, crcHigh) {
		// CRC校验失败，返回这个无效包让上层过滤
		return packet, false, true
	}

	return packet, true, true
}

// ParsePacket 解析数据包
func (h *IoTProtocolHandler) ParsePacket(packet []byte) (cmd byte, data []byte, err error) {
	if len(packet) < 7 {
		return 0, nil, fmt.Errorf("数据包长度不足")
	}

	cmd = packet[1] // 命令码（1字节）

	// 数据长度（2字节，大端序）
	dataLen := int(binary.BigEndian.Uint16(packet[2:4]))
	if dataLen > 0 {
		data = packet[4 : 4+dataLen]
	}

	return cmd, data, nil
}

// buildCommand 构建命令包(用于主动发送)
func (h *IoTProtocolHandler) buildCommand(cmd byte, data []byte) []byte {
	packet := make([]byte, 0, len(data)+7)
	packet = append(packet, STX)
	packet = append(packet, cmd) // 命令码（1字节）

	// 数据长度（2字节，大端序）
	dataLen := uint16(len(data))
	packet = append(packet, byte(dataLen>>8), byte(dataLen&0xFF))
	packet = append(packet, data...)

	// 计算CRC16校验码
	crcData := packet[1:]
	crcLow, crcHigh := h.calculateModbusCRC(crcData)
	packet = append(packet, crcLow, crcHigh)
	packet = append(packet, ETX)

	return packet
}

// BuildReadIDCommand 构建读取ID命令
func (h *IoTProtocolHandler) BuildReadIDCommand() []byte {
	return h.buildCommand(CMD_READ_ID, []byte{})
}

// ParseDeviceIDResponse 解析设备ID响应
func (h *IoTProtocolHandler) ParseDeviceIDResponse(data []byte) (uint32, bool) {
	if len(data) < 4 {
		return 0, false
	}

	// 解析4字节设备ID (大端序)
	deviceID := binary.BigEndian.Uint32(data[0:4])
	return deviceID, true
}

// BuildSetReportIntervalCommand 构建设置数据上报间隔命令
func (h *IoTProtocolHandler) BuildSetReportIntervalCommand(intervalMinutes uint16) []byte {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, intervalMinutes)
	return h.buildCommand(CMD_SET_REPORT_INTERVAL, data)
}

// ParseSetReportIntervalResponse 解析设置上报间隔响应
func (h *IoTProtocolHandler) ParseSetReportIntervalResponse(data []byte) (uint16, bool) {
	if len(data) < 2 {
		return 0, false
	}

	// 解析设置的间隔时间（2字节，大端序）
	intervalMinutes := binary.BigEndian.Uint16(data[0:2])
	return intervalMinutes, true
}

// BuildRelayOperationCommand 构建继电器操作命令
func (h *IoTProtocolHandler) BuildRelayOperationCommand(relayIndex int, isOpen bool) []byte {
	action := RELAY_OFF
	if isOpen {
		action = RELAY_ON
	}

	// 构建继电器操作数据：继电器索引(1字节) + 操作(1字节)
	data := make([]byte, 2)
	data[0] = byte(relayIndex)
	data[1] = byte(action)

	return h.buildCommand(CMD_CONTROL_RELAY, data)
}

// ParseRelayOperationResponse 解析继电器操作响应
func (h *IoTProtocolHandler) ParseRelayOperationResponse(data []byte) (int, byte) {
	if len(data) < 2 {
		return 0, 0
	}

	relayIndex := int(data[0])
	relayState := data[1] // 继电器状态 0或1

	return relayIndex, relayState
}

// ParseDataReport 解析设备主动上报数据
func (h *IoTProtocolHandler) ParseDataReport(data []byte) bool {
	if len(data) < 64 {
		return false
	}

	// 数据长度正确，返回true
	return true
}
