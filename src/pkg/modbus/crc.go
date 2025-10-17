package modbus

import (
	"encoding/binary"
	"fmt"
)

// CalculateCRC16 calculates the Modbus CRC-16 checksum for the given data
// This implements the standard Modbus RTU CRC-16 algorithm
func CalculateCRC16(data []byte) uint16 {
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

// AppendCRC appends the CRC-16 checksum to the data
// The CRC is appended in little-endian format (low byte first, high byte second)
func AppendCRC(data []byte) []byte {
	crc := CalculateCRC16(data)

	// Append CRC in little-endian format (Modbus standard)
	result := make([]byte, len(data)+2)
	copy(result, data)
	result[len(data)] = byte(crc & 0xFF)          // Low byte
	result[len(data)+1] = byte((crc >> 8) & 0xFF) // High byte

	return result
}

// VerifyCRC verifies that the CRC in the data is correct
// Returns true if the CRC is valid, false otherwise
func VerifyCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// Extract the message (without CRC)
	message := data[:len(data)-2]

	// Extract the received CRC
	receivedCRC := uint16(data[len(data)-2]) | (uint16(data[len(data)-1]) << 8)

	// Calculate expected CRC
	calculatedCRC := CalculateCRC16(message)

	return receivedCRC == calculatedCRC
}

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

	// Append CRC
	return AppendCRC(command)
}

// BuildModbusCommandHex builds a complete Modbus RTU command and returns it as a hex string
func BuildModbusCommandHex(slaveID uint8, functionCode uint8, startAddress uint16, count uint16) string {
	commandBytes := BuildModbusCommand(slaveID, functionCode, startAddress, count)
	return fmt.Sprintf("%X", commandBytes)
}

// ParseModbusCommand parses a hex string into command components
// Returns: slaveID, functionCode, startAddress, count, valid
func ParseModbusCommand(hexString string) (uint8, uint8, uint16, uint16, bool) {
	// Remove any spaces or separators
	hexString = removeSpaces(hexString)

	// Convert hex string to bytes
	data, err := hexToBytes(hexString)
	if err != nil || len(data) < 8 {
		return 0, 0, 0, 0, false
	}

	// Verify CRC
	if !VerifyCRC(data) {
		return 0, 0, 0, 0, false
	}

	// Parse command components
	slaveID := data[0]
	functionCode := data[1]
	startAddress := binary.BigEndian.Uint16(data[2:4])
	count := binary.BigEndian.Uint16(data[4:6])

	return slaveID, functionCode, startAddress, count, true
}

// Helper function to remove spaces from hex string
func removeSpaces(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '-' && s[i] != ':' {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// Helper function to convert hex string to bytes
func hexToBytes(hexString string) ([]byte, error) {
	if len(hexString)%2 != 0 {
		return nil, fmt.Errorf("hex string must have even length")
	}

	result := make([]byte, len(hexString)/2)
	for i := 0; i < len(result); i++ {
		var b byte
		_, err := fmt.Sscanf(hexString[i*2:i*2+2], "%02X", &b)
		if err != nil {
			return nil, err
		}
		result[i] = b
	}

	return result, nil
}
