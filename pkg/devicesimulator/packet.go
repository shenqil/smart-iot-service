package devicesimulator

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	STX = 0xAA
	ETX = 0x55
)

type Packet struct {
	CMD  byte
	Data []byte
}

func BuildPacket(cmd byte, data []byte) []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(STX)
	buf.WriteByte(cmd)

	// LEN: 2 bytes big endian
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(data)))
	buf.Write(lenBytes)

	buf.Write(data)

	// CRC 计算：CMD + LEN + DATA
	crcData := append([]byte{cmd}, lenBytes...)
	crcData = append(crcData, data...)
	crc := CRC16Modbus(crcData)
	buf.WriteByte(byte(crc & 0xFF))
	buf.WriteByte(byte((crc >> 8) & 0xFF))

	buf.WriteByte(ETX)

	return buf.Bytes()
}

func ParsePacket(raw []byte) (*Packet, error) {
	if len(raw) < 7 {
		return nil, fmt.Errorf("packet too short")
	}
	if raw[0] != STX || raw[len(raw)-1] != ETX {
		return nil, fmt.Errorf("STX/ETX mismatch")
	}

	cmd := raw[1]
	length := binary.BigEndian.Uint16(raw[2:4])

	if int(length)+7 != len(raw) {
		return nil, fmt.Errorf("length mismatch")
	}

	data := raw[4 : 4+length]

	// CRC check: CMD + LEN + DATA
	crcData := append([]byte{cmd}, raw[2:4]...) // CMD + LEN
	crcData = append(crcData, data...)          // + DATA
	crcCalc := CRC16Modbus(crcData)
	crcRecv := uint16(raw[len(raw)-3]) | uint16(raw[len(raw)-2])<<8

	if crcCalc != crcRecv {
		return nil, fmt.Errorf("CRC mismatch")
	}

	return &Packet{CMD: cmd, Data: data}, nil
}
