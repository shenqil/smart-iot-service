package controlv2

import (
	"fmt"
	"time"

	"rbac-admin/pkg/collector/singletons"

	"github.com/panjf2000/gnet/v2"
)

func (s *ControlServiceV2) publishEvent(eventType singletons.EventType, data interface{}) {
	publisher := singletons.GetEventPublisher()
	event := singletons.Event{Type: eventType, Data: data}
	publisher.Publish(event)
}

func (s *ControlServiceV2) handleDeviceIDResponse(frame []byte, ctx *ControlConnContextV2, c gnet.Conn) {
	// O(1) 纯数学计算解析设备ID
	deviceID := uint32(frame[6])*1000000 + uint32(frame[7])*10000 + uint32(frame[8])*100 + uint32(frame[9])
	s.registerDevice(deviceID, ctx, c)
}

func (s *ControlServiceV2) handlePollingData(frame []byte, totalLen int, indicator byte, ctx *ControlConnContextV2) {
	if ctx.deviceID == 0 {
		return
	}
	rawPayload := frame[4 : totalLen-2] // 写死边界：前4字节为头，后2字节为CRC

	var standardData []byte
	switch indicator {
	case 0x40:
		// 16 通道：原始数据本身即为标准的 64 字节 (32B通道 + 32B继电器)
		standardData = rawPayload

	case 0xA1:
		// A1 (64通道，每通道2字节，共160字节: 128B通道 + 32B继电器)
		// 只取前16通道数据(前32字节)与32个继电器状态(后32字节)
		standardData = make([]byte, 64)
		if len(rawPayload) >= 32 {
			copy(standardData[0:32], rawPayload[0:32])
		}
		if len(rawPayload) >= 128+32 {
			copy(standardData[32:64], rawPayload[128:160])
		}

	case 0xA0:
		// A0 (32通道，每通道4字节，共160字节: 128B通道 + 32B继电器)
		// 只取前16通道数据，每通道取低16位(2字节)有效数据
		standardData = make([]byte, 64)
		for ch := 0; ch < 16; ch++ {
			src := ch * 4
			dst := ch * 2
			if src+4 <= len(rawPayload) {
				standardData[dst] = rawPayload[src+2]
				standardData[dst+1] = rawPayload[src+3]
			}
		}
		if len(rawPayload) >= 128+32 {
			copy(standardData[32:64], rawPayload[128:160])
		}

	default:
		standardData = rawPayload
	}

	s.publishEvent(singletons.EventData, &singletons.DeviceData{
		ID:       ctx.deviceID,
		Type:     1,
		LastSeen: time.Now(),
		Data:     standardData,
	})
	s.logger.Info(fmt.Sprintf("%s 收到轮询数据 (设备ID: %d, 指令: 0x%02X)", s.name, ctx.deviceID, indicator))
}

func (s *ControlServiceV2) handleRelayResponse(frame []byte, ctx *ControlConnContextV2) {
	if ctx.deviceID == 0 {
		return
	}
	relayIndex := frame[4]
	relayStatus := frame[5]
	status := byte(0)
	if relayStatus == 1 {
		status = 1
	}
	s.publishEvent(singletons.EventRelayStatus, &singletons.RelayStatus{
		ID:       ctx.deviceID,
		RelayNum: uint32(relayIndex),
		State:    status,
	})
	s.logger.Info(fmt.Sprintf("%s 继电器响应 (设备ID: %d, 继电器: %d, 状态: %d)", s.name, ctx.deviceID, relayIndex, status))
}

func (s *ControlServiceV2) registerDevice(deviceID uint32, ctx *ControlConnContextV2, c gnet.Conn) {
	if ctx.deviceID == deviceID {
		return 
	}
	ctx.deviceID = deviceID
	if ctx.timeoutTimer != nil {
		ctx.timeoutTimer.Stop()
		ctx.timeoutTimer = nil
	}

	if existingConnInterface, exists := s.deviceConns.Load(deviceID); exists {
		if existingConn, ok := existingConnInterface.(gnet.Conn); ok && existingConn != c {
			s.logger.Warn(fmt.Sprintf("%s 设备 %d 已存在，踢出旧连接", s.name, deviceID))
			s.scheduler.StopTask(deviceID)
			existingConn.Close()
		}
	}

	s.deviceConns.Store(deviceID, c)
	s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
		ID:           deviceID,
		Online:       1,
		Time:         time.Now(),
		ConnectionID: c.RemoteAddr().String(),
	})
}
