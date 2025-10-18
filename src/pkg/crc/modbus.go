package crc

// CRC16 calculates the Modbus RTU CRC16 checksum
// This implements the standard Modbus RTU CRC-16 algorithm
func CRC16(data []byte) uint16 {
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

// VerifyCRC verifies the CRC16 checksum of a Modbus RTU response
// Returns true if the CRC is valid, false otherwise
func VerifyCRC(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Calculate CRC for data (excluding last 2 CRC bytes)
	dataWithoutCRC := data[:len(data)-2]
	calculatedCRC := CRC16(dataWithoutCRC)

	// Extract CRC from message (little-endian: CRC low byte first, then high byte)
	messageCRC := uint16(data[len(data)-2]) | (uint16(data[len(data)-1]) << 8)

	return calculatedCRC == messageCRC
}

// AppendCRC appends the CRC16 checksum to the data
// The CRC is appended in little-endian format (low byte first, high byte second)
func AppendCRC(data []byte) []byte {
	crc := CRC16(data)

	// Append CRC in little-endian format (Modbus standard)
	result := make([]byte, len(data)+2)
	copy(result, data)
	result[len(data)] = byte(crc & 0xFF)          // Low byte
	result[len(data)+1] = byte((crc >> 8) & 0xFF) // High byte

	return result
}
