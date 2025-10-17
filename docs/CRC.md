# Modbus CRC-16 Implementation

## Overview

This package provides automatic CRC-16 (Modbus) calculation and verification for Modbus RTU commands. The CRC is calculated automatically when building commands, eliminating manual CRC calculation errors.

## Features

- ✅ **Automatic CRC Calculation**: Commands are automatically appended with correct CRC-16
- ✅ **CRC Verification**: Incoming responses are verified for data integrity
- ✅ **Standard Compliance**: Implements Modbus RTU CRC-16 standard (polynomial 0xA001)
- ✅ **Easy to Use**: Simple API for building and parsing commands

## CRC-16 Algorithm

The implementation uses the standard Modbus CRC-16 algorithm:
- **Polynomial**: 0xA001 (reflected/reversed form of 0x8005)
- **Initial Value**: 0xFFFF
- **Byte Order**: Little-endian (low byte first, high byte second)
- **Reflection**: Input and output reflected

## Usage Examples

### Building a Modbus Command

```go
import "mqtt-modbus-bridge/pkg/modbus"

// Build a command to read 34 registers starting at address 0x2000
// CRC is automatically calculated and appended
command := modbus.BuildModbusCommand(
    0x0B,   // Slave ID
    0x03,   // Function Code (Read Holding Registers)
    0x2000, // Start Address
    0x0022, // Register Count (34 in decimal)
)
// Result: 0B032000002247BF (last 2 bytes are CRC)
```

### Building a Command as Hex String

```go
// Same command, but returned as hex string
hexCommand := modbus.BuildModbusCommandHex(0x0B, 0x03, 0x2000, 0x0022)
// Result: "0B032000002247BF"
```

### Verifying CRC of Received Data

```go
response := []byte{0x0B, 0x03, 0x24, 0x43, 0x63, 0x00, 0x00, /* ... */, 0xAB, 0xCD}

if modbus.VerifyCRC(response) {
    fmt.Println("CRC is valid ✅")
} else {
    fmt.Println("CRC is invalid ❌")
}
```

### Parsing a Modbus Command

```go
hexString := "0B032000002247BF"

slaveID, functionCode, address, count, valid := modbus.ParseModbusCommand(hexString)

if valid {
    fmt.Printf("Slave ID: 0x%02X\n", slaveID)           // 0x0B
    fmt.Printf("Function: 0x%02X\n", functionCode)      // 0x03
    fmt.Printf("Address: 0x%04X\n", address)            // 0x2000
    fmt.Printf("Count: 0x%04X\n", count)                // 0x0022
}
```

### Manual CRC Calculation

```go
data := []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22}
crc := modbus.CalculateCRC16(data)
fmt.Printf("CRC: 0x%04X\n", crc) // 0xBF47 (will be stored as 47 BF in little-endian)
```

### Appending CRC to Data

```go
data := []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22}
commandWithCRC := modbus.AppendCRC(data)
// Result: []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22, 0x47, 0xBF}
```

## Integration with Gateway

The gateway automatically uses the CRC functions:

```go
// In gateway.go
func (g *USRGateway) buildModbusCommand(slaveID uint8, functionCode uint8, address uint16, count uint16) []byte {
    // CRC is automatically calculated and appended
    return modbus.BuildModbusCommand(slaveID, functionCode, address, count)
}
```

Received messages are automatically verified:

```go
func (g *USRGateway) onMessage(client mqtt.Client, msg mqtt.Message) {
    data := msg.Payload()
    
    // Verify CRC before processing
    if !modbus.VerifyCRC(data) {
        logger.LogWarn("Received message with invalid CRC, ignoring")
        return
    }
    
    // Process valid data...
}
```

## Command Examples

### Read Instant Registers (0x2000-0x2020)

```go
// Read 34 registers for voltage, current, power, frequency, power factor
command := modbus.BuildModbusCommand(0x0B, 0x03, 0x2000, 0x0022)
// Hex: 0B032000002247BF
```

**Breakdown**:
- `0B` - Slave ID (device 11)
- `03` - Function code (Read Holding Registers)
- `2000` - Start address (0x2000)
- `0022` - Register count (34 registers = 68 bytes)
- `47BF` - CRC-16 (automatically calculated)

### Read Energy Registers (0x4000-0x4014)

```go
// Read 22 registers for energy values
command := modbus.BuildModbusCommand(0x0B, 0x03, 0x4000, 0x0016)
// Hex: 0B034000001690B4
```

**Breakdown**:
- `0B` - Slave ID
- `03` - Function code
- `4000` - Start address (0x4000)
- `0016` - Register count (22 registers)
- `90B4` - CRC-16 (automatically calculated)

## Benefits

### Before (Manual CRC)

```go
// Manual calculation - error prone! ❌
cmd := make([]byte, 8)
cmd[0] = 0x0B
cmd[1] = 0x03
cmd[2] = 0x20
cmd[3] = 0x00
cmd[4] = 0x00
cmd[5] = 0x22

// Calculate CRC manually
crc := calculateCRC16(cmd[:6])
cmd[6] = byte(crc & 0xFF)
cmd[7] = byte(crc >> 8)
```

### After (Automatic CRC)

```go
// Automatic calculation - no errors! ✅
cmd := modbus.BuildModbusCommand(0x0B, 0x03, 0x2000, 0x0022)
```

## Testing

The CRC implementation is thoroughly tested:

```bash
# Run CRC tests
cd tests
go test ./unit -run TestCRC -v

# Run all tests
./run-tests.ps1
```

Test coverage includes:
- ✅ CRC-16 calculation accuracy
- ✅ Append CRC functionality
- ✅ CRC verification
- ✅ Command building
- ✅ Command parsing
- ✅ Round-trip integrity

## CRC Calculation Details

### Algorithm Steps

1. **Initialize**: CRC = 0xFFFF
2. **For each byte**:
   - XOR byte with CRC low byte
   - For 8 bits:
     - If LSB is 1: shift right and XOR with 0xA001
     - If LSB is 0: just shift right
3. **Result**: 16-bit CRC value

### Example Calculation

For command `0B 03 20 00 00 22`:

```
Initial CRC:     0xFFFF
After byte 0x0B: 0xF4F4
After byte 0x03: 0xF7F7
After byte 0x20: 0xD7E7
After byte 0x00: 0x6BF3
After byte 0x00: 0x35F9
After byte 0x22: 0xBF47
```

Final CRC: `0xBF47` → stored as `47 BF` (little-endian)

## Error Detection

The CRC-16 can detect:
- All single-bit errors
- All double-bit errors
- All errors with odd number of bits
- All burst errors of length ≤ 16 bits
- 99.998% of all other errors

## References

- [Modbus Application Protocol Specification V1.1b3](http://www.modbus.org/docs/Modbus_Application_Protocol_V1_1b3.pdf)
- [Modbus over Serial Line Specification V1.02](http://www.modbus.org/docs/Modbus_over_serial_line_V1_02.pdf)
- CRC Polynomial: 0x8005 (normal) / 0xA001 (reflected)

## License

Part of the CHINT MQTT-Modbus Bridge project.
