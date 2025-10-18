package modbus

import (
	"encoding/binary"
	"fmt"
	"mqtt-modbus-bridge/pkg/crc"
)

// BuildModbusCommand builds a complete Modbus RTU command with CRC
// slaveID: Modbus slave/device ID
// functionCode: Modbus function code (0x03 = Read Holding Registers, etc.)
// startAddress: Starting register address
// count: Number of registers to read/write
func BuildModbusCommand(slaveID uint8, functionCode uint8, startAddress uint16, count uint16) []byte {
	// Build the command without CRC
	command := make([]byte, 6)
	command[0] = slaveID
	command[1] = functionCode
	binary.BigEndian.PutUint16(command[2:4], startAddress)
	binary.BigEndian.PutUint16(command[4:6], count)

	// Append CRC using crc package
	return crc.AppendCRC(command)
}

// BuildModbusCommandHex builds a complete Modbus RTU command and returns it as a hex string
func BuildModbusCommandHex(slaveID uint8, functionCode uint8, startAddress uint16, count uint16) string {
	command := BuildModbusCommand(slaveID, functionCode, startAddress, count)
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
		command[0], command[1], command[2], command[3],
		command[4], command[5], command[6], command[7])
}
