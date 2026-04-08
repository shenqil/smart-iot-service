package devicesimulator

func CRC16Modbus(data []byte) uint16 {
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
	return crc
}
