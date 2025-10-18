# Tests Directory

This directory contains comprehensive tests for the CHINT MQTT-Modbus Bridge.

## Structure

```
tests/
├── unit/                              # Unit tests
│   ├── crc_test.go                    # CRC16 calculation tests
│   ├── config_test.go                 # Configuration loading/parsing tests
│   ├── config_devices_test.go         # Device configuration validation tests
│   ├── formula_validation_test.go     # Formula syntax validation tests
│   └── version_test.go                # Version compatibility tests
├── integration/                       # Integration tests (currently empty)
├── go.mod                             # Test module definition
└── README.md                          # This file
```

## Test Categories

### Unit Tests (`unit/`)

Tests for individual components in isolation:

- **crc_test.go** - Tests CRC16 calculation, verification, and round-trip consistency for Modbus RTU protocol
- **config_test.go** - Tests YAML configuration loading, validation, and register configuration
- **config_devices_test.go** - Tests device-based configuration structure, validation rules, and calculated values
- **formula_validation_test.go** - Tests formula syntax parsing and variable extraction for calculated values
- **version_test.go** - Tests configuration version compatibility and migration logic

### Integration Tests (`integration/`)

Tests for component interactions (to be implemented for Strategy Pattern integration testing)

## Running Tests

From the project root:

```bash
./run-tests.sh      # Linux/macOS
./run-tests.ps1     # Windows PowerShell
```

Or directly:

```bash
# All tests
cd tests
go test ./... -v

# Unit tests only
go test ./unit/... -v

# Integration tests only
go test ./integration/... -v
```

## Test Coverage

```bash
cd tests

# Generate coverage for all tests
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out
```

## Module Configuration

The `go.mod` file uses a `replace` directive to reference the main application module in `../src`:

```go
module mqtt-modbus-bridge-tests

replace mqtt-modbus-bridge => ../src

require mqtt-modbus-bridge v0.0.0-00010101000000-000000000000
```

This allows tests to import packages from the main application while keeping tests separate from source code.

## Continuous Integration

Tests run automatically on every push and pull request via GitHub Actions. The CI workflow:

1. Runs all unit tests with coverage
2. Runs all integration tests with coverage
3. Generates coverage reports
4. Builds the application
5. Runs linting checks
6. Performs security scans

## Adding New Tests

1. Place unit tests in `tests/unit/`
2. Place integration tests in `tests/integration/`
3. Use package name `unit` or `integration`
4. Import packages from `mqtt-modbus-bridge/pkg/...`
5. Follow table-driven test pattern for multiple test cases
6. Include error handling tests
7. Run tests locally before committing

Example test structure:

```go
package unit

import (
    "testing"
    "mqtt-modbus-bridge/pkg/modbus"
)

func TestSomething(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test logic here
        })
    }
}
```
