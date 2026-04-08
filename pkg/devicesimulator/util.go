package devicesimulator

import "fmt"

func FindPacket(buffer []byte) (packet []byte, remain []byte) {
	if len(buffer) < 7 {
		return nil, buffer
	}
	// 查找 STX
	i := 0
	for i < len(buffer) && buffer[i] != STX {
		i++
	}
	if i > 0 {
		buffer = buffer[i:]
	}
	if len(buffer) < 7 {
		return nil, buffer
	}

	// 长度字段
	length := int(buffer[2])<<8 | int(buffer[3])
	total := 7 + length

	if len(buffer) < total {
		return nil, buffer
	}

	if buffer[total-1] != ETX {
		return nil, buffer[1:]
	}

	return buffer[:total], buffer[total:]
}

func fmtHex(b []byte) string {
	s := ""
	for _, v := range b {
		s += fmt.Sprintf("%02X ", v)
	}
	return s
}
