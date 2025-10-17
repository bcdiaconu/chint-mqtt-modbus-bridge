package unit

import (
	"mqtt-modbus-bridge/pkg/modbus"
	"testing"
)

func TestCRC16Calculation(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint16
	}{
		{
			name:     "instant command",
			data:     []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22},
			expected: 0xB9CE, // CRC = 0xB9CE (stored as CE B9 in little-endian)
		},
		{
			name:     "energy command",
			data:     []byte{0x0B, 0x03, 0x40, 0x00, 0x00, 0x16},
			expected: 0x6ED1, // CRC = 0x6ED1 (stored as D1 6E in little-endian)
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0xFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modbus.CalculateCRC16(tt.data)
			if result != tt.expected {
				t.Errorf("CalculateCRC16() = 0x%04X, expected 0x%04X", result, tt.expected)
			}
		})
	}
}

func TestAppendCRC(t *testing.T) {
	data := []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x12}
	result := modbus.AppendCRC(data)

	// Should append 2 bytes for CRC
	if len(result) != len(data)+2 {
		t.Errorf("AppendCRC() length = %d, expected %d", len(result), len(data)+2)
	}

	// Original data should be intact
	for i := 0; i < len(data); i++ {
		if result[i] != data[i] {
			t.Errorf("AppendCRC() modified original data at index %d", i)
		}
	}

	// Verify the appended CRC is correct
	if !modbus.VerifyCRC(result) {
		t.Error("AppendCRC() produced invalid CRC")
	}
}

func TestVerifyCRC(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid instant group command",
			data:     []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22, 0xCE, 0xB9},
			expected: true,
		},
		{
			name:     "valid energy command",
			data:     []byte{0x0B, 0x03, 0x40, 0x00, 0x00, 0x16, 0xD1, 0x6E},
			expected: true,
		},
		{
			name:     "invalid CRC",
			data:     []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22, 0xFF, 0xFF},
			expected: false,
		},
		{
			name:     "too short",
			data:     []byte{0x0B},
			expected: false,
		},
		{
			name:     "empty",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modbus.VerifyCRC(tt.data)
			if result != tt.expected {
				t.Errorf("VerifyCRC() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBuildModbusCommand(t *testing.T) {
	tests := []struct {
		name         string
		slaveID      uint8
		functionCode uint8
		startAddress uint16
		count        uint16
		expected     string
	}{
		{
			name:         "instant group read",
			slaveID:      0x0B,
			functionCode: 0x03,
			startAddress: 0x2000,
			count:        0x0022, // Updated to 34 registers after our fix
			expected:     "0B0320000022",
		},
		{
			name:         "energy group read",
			slaveID:      0x0B,
			functionCode: 0x03,
			startAddress: 0x4000,
			count:        0x0016, // Updated to 22 registers after our fix
			expected:     "0B0340000016",
		},
		{
			name:         "single register read",
			slaveID:      0x01,
			functionCode: 0x03,
			startAddress: 0x0000,
			count:        0x0001,
			expected:     "010300000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modbus.BuildModbusCommand(tt.slaveID, tt.functionCode, tt.startAddress, tt.count)

			// Verify command structure (without CRC comparison since we want to test structure)
			if result[0] != tt.slaveID {
				t.Errorf("BuildModbusCommand() slaveID = 0x%02X, expected 0x%02X", result[0], tt.slaveID)
			}
			if result[1] != tt.functionCode {
				t.Errorf("BuildModbusCommand() functionCode = 0x%02X, expected 0x%02X", result[1], tt.functionCode)
			}

			// Verify CRC is valid
			if !modbus.VerifyCRC(result) {
				t.Error("BuildModbusCommand() produced invalid CRC")
			}

			// Verify total length (6 bytes command + 2 bytes CRC)
			if len(result) != 8 {
				t.Errorf("BuildModbusCommand() length = %d, expected 8", len(result))
			}
		})
	}
}

func TestBuildModbusCommandHex(t *testing.T) {
	slaveID := uint8(0x0B)
	functionCode := uint8(0x03)
	startAddress := uint16(0x2000)
	count := uint16(0x0022)

	result := modbus.BuildModbusCommandHex(slaveID, functionCode, startAddress, count)

	// Should be 16 hex characters (8 bytes * 2)
	if len(result) != 16 {
		t.Errorf("BuildModbusCommandHex() length = %d, expected 16", len(result))
	}

	// Should start with the expected command bytes
	if result[:12] != "0B0320000022" {
		t.Errorf("BuildModbusCommandHex() command bytes = %s, expected to start with 0B0320000022", result[:12])
	}

	t.Logf("Generated command: %s", result)
}

func TestParseModbusCommand(t *testing.T) {
	tests := []struct {
		name             string
		hexString        string
		expectedSlaveID  uint8
		expectedFunction uint8
		expectedAddress  uint16
		expectedCount    uint16
		expectedValid    bool
	}{
		{
			name:             "valid instant command",
			hexString:        "0B0320000022XXXX", // CRC will be calculated
			expectedSlaveID:  0x0B,
			expectedFunction: 0x03,
			expectedAddress:  0x2000,
			expectedCount:    0x0022,
			expectedValid:    true,
		},
		{
			name:             "invalid CRC",
			hexString:        "0B0320000022FFFF",
			expectedSlaveID:  0,
			expectedFunction: 0,
			expectedAddress:  0,
			expectedCount:    0,
			expectedValid:    false,
		},
		{
			name:             "too short",
			hexString:        "0B03",
			expectedSlaveID:  0,
			expectedFunction: 0,
			expectedAddress:  0,
			expectedCount:    0,
			expectedValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For valid test case, build the command with correct CRC
			testHex := tt.hexString
			if tt.expectedValid {
				cmd := modbus.BuildModbusCommand(tt.expectedSlaveID, tt.expectedFunction, tt.expectedAddress, tt.expectedCount)
				testHex = modbus.BuildModbusCommandHex(tt.expectedSlaveID, tt.expectedFunction, tt.expectedAddress, tt.expectedCount)
				t.Logf("Testing with command: %s (bytes: %X)", testHex, cmd)
			}

			slaveID, functionCode, address, count, valid := modbus.ParseModbusCommand(testHex)

			if valid != tt.expectedValid {
				t.Errorf("ParseModbusCommand() valid = %v, expected %v", valid, tt.expectedValid)
			}

			if tt.expectedValid {
				if slaveID != tt.expectedSlaveID {
					t.Errorf("ParseModbusCommand() slaveID = 0x%02X, expected 0x%02X", slaveID, tt.expectedSlaveID)
				}
				if functionCode != tt.expectedFunction {
					t.Errorf("ParseModbusCommand() functionCode = 0x%02X, expected 0x%02X", functionCode, tt.expectedFunction)
				}
				if address != tt.expectedAddress {
					t.Errorf("ParseModbusCommand() address = 0x%04X, expected 0x%04X", address, tt.expectedAddress)
				}
				if count != tt.expectedCount {
					t.Errorf("ParseModbusCommand() count = 0x%04X, expected 0x%04X", count, tt.expectedCount)
				}
			}
		})
	}
}

func TestCRCRoundTrip(t *testing.T) {
	// Test that we can build a command and verify its CRC
	slaveID := uint8(0x0B)
	functionCode := uint8(0x03)
	startAddress := uint16(0x2000)
	count := uint16(0x0022)

	// Build command with CRC
	command := modbus.BuildModbusCommand(slaveID, functionCode, startAddress, count)

	// Verify CRC
	if !modbus.VerifyCRC(command) {
		t.Error("Round trip CRC verification failed")
	}

	// Parse command back
	parsedSlaveID, parsedFunction, parsedAddress, parsedCount, valid := modbus.ParseModbusCommand(
		modbus.BuildModbusCommandHex(slaveID, functionCode, startAddress, count),
	)

	if !valid {
		t.Error("Round trip parse failed")
	}

	if parsedSlaveID != slaveID || parsedFunction != functionCode ||
		parsedAddress != startAddress || parsedCount != count {
		t.Error("Round trip data mismatch")
	}
}
