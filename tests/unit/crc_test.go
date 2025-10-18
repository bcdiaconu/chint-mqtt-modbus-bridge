package unit

import (
"mqtt-modbus-bridge/pkg/crc"
"testing"
)

func TestCRC16_Deterministic(t *testing.T) {
tests := []struct {
name string
data []byte
}{
{name: "Empty data", data: []byte{}},
{name: "Single byte", data: []byte{0x01}},
{name: "Modbus command", data: []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22}},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result1 := crc.CRC16(tt.data)
result2 := crc.CRC16(tt.data)
if result1 != result2 {
t.Errorf("CRC16() not deterministic")
}
})
}
}

func TestVerifyCRC(t *testing.T) {
tests := []struct {
name     string
data     []byte
expected bool
}{
{name: "Too short", data: []byte{0x01, 0x02}, expected: false},
{name: "Invalid CRC", data: []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22, 0xFF, 0xFF}, expected: false},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := crc.VerifyCRC(tt.data)
if result != tt.expected {
t.Errorf("VerifyCRC() = %v, want %v", result, tt.expected)
}
})
}

t.Run("Valid from AppendCRC", func(t *testing.T) {
testData := [][]byte{
{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22},
{0x11, 0x03, 0x00, 0x00, 0x00, 0x02},
}
for _, data := range testData {
withCRC := crc.AppendCRC(data)
if !crc.VerifyCRC(withCRC) {
t.Errorf("VerifyCRC failed for AppendCRC output")
}
}
})
}

func TestAppendCRC(t *testing.T) {
tests := []struct {
name string
data []byte
}{
{name: "Modbus command", data: []byte{0x0B, 0x03, 0x20, 0x00, 0x00, 0x22}},
{name: "Minimum frame", data: []byte{0x01, 0x03}},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := crc.AppendCRC(tt.data)
if len(result) != len(tt.data)+2 {
t.Errorf("AppendCRC() length incorrect")
return
}
for i := 0; i < len(tt.data); i++ {
if result[i] != tt.data[i] {
t.Errorf("AppendCRC() modified original data")
}
}
if len(result) >= 4 && !crc.VerifyCRC(result) {
t.Errorf("AppendCRC() produced invalid CRC")
}
})
}
}
