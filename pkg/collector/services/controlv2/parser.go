package controlv2

import (
	"github.com/panjf2000/gnet/v2"
)

func (s *ControlServiceV2) tryParsePacket(frame []byte, ctx *ControlConnContextV2, c gnet.Conn) int {
	// 基础帧头需要至少 4 字节 (addr + cmd + 2B) 才能判断指令类型
	// 数据不足 4 字节时等待更多数据，避免误判导致错误丢弃字节
	if len(frame) < 4 {
		return 0
	}

	// 1. 严格校验 轮询/ID读取返回 (03)
	// 协议文档 2.2 和 2.3 案例限定:
	if frame[1] == 0x03 && frame[2] == 0x00 {
		indicator := frame[3]

		switch indicator {
		case 0x60: // 2.2 读取设备ID响应 (固定 12 字节)
			expectedLen := 12 // addr(1) + 03(1) + 00 60(2) + 00 04(2) + ID(4) + CRC(2) = 12
			if len(frame) < expectedLen {
				return 0
			}
			if frame[4] == 0x00 && frame[5] == 0x04 {
				if s.verifyCRC(frame, expectedLen) {
					s.handleDeviceIDResponse(frame, ctx, c)
					return expectedLen
				}
			}
		case 0x40: // 2.3.1 16通道数据读取 (固定 0x40 对应 64字节数据)
			expectedLen := 70 // addr(1) + 03(1) + 00 40(2) + 64(data) + CRC(2) = 70
			if len(frame) < expectedLen {
				return 0
			}
			if s.verifyCRC(frame, expectedLen) {
				s.handlePollingData(frame, expectedLen, indicator, ctx)
				return expectedLen
			}
		case 0xA0: // 2.3.2 32通道数据读取 (固定 0xA0 对应 160字节数据)
			expectedLen := 166 // addr(1) + 03(1) + 00 A0(2) + 160(data) + CRC(2) = 166
			if len(frame) < expectedLen {
				return 0
			}
			if s.verifyCRC(frame, expectedLen) {
				s.handlePollingData(frame, expectedLen, indicator, ctx)
				return expectedLen
			}
		case 0xA1: // 2.3.3 64通道数据读取 (固定 0xA1 对应 160字节数据)
			expectedLen := 166 // addr(1) + 03(1) + 00 A1(2) + 160(data) + CRC(2) = 166
			if len(frame) < expectedLen {
				return 0
			}
			if s.verifyCRC(frame, expectedLen) {
				s.handlePollingData(frame, expectedLen, indicator, ctx)
				return expectedLen
			}
		}
	}

	// 3. 严格校验 继电器控制响应 (10)
	// 协议文档 2.4 格式: addr 10 00 7A num state CRC_L CRC_H
	if len(frame) >= 4 && frame[1] == 0x10 && frame[2] == 0x00 && frame[3] == 0x7A {
		expectedLen := 8 // addr(1) + 10(1) + 00 7A(2) + num(1) + state(1) + CRC(2)
		if len(frame) < expectedLen {
			return 0
		}
		// 严密校验业务参数的合理范围：继电器编号(0-31), 状态(0/1)
		num := frame[4]
		state := frame[5]
		if num <= 31 && (state == 0x00 || state == 0x01) {
			if s.verifyCRC(frame, expectedLen) {
				s.handleRelayResponse(frame, ctx)
				return expectedLen
			}
		}
	}

	// 所有已知协议规则都不匹配，或者 CRC 校验失败
	// 滑动窗口：返回 -1，使 Buffer 丢弃 1 个字节，继续向后寻找合法的包头
	return -1
}

func (s *ControlServiceV2) verifyCRC(frame []byte, totalLen int) bool {
	crc1, crc2 := s.calculateCRC(frame[:totalLen-2])
	return frame[totalLen-2] == crc1 && frame[totalLen-1] == crc2
}

func (s *ControlServiceV2) calculateCRC(data []byte) (byte, byte) {
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
