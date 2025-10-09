# Tests Directory

This directory contains comprehensive tests for the CHINT MQTT-Modbus Bridge.

## Structure

```
tests/
├── unit/                           # Unit tests
│   ├── modbus_commands_test.go    # Command parsing tests
│   ├── config_test.go              # Configuration tests
│   └── factory_test.go             # Factory pattern tests
├── integration/                    # Integration tests
│   └── groups_integration_test.go  # Group execution tests
├── go.mod                          # Test module definition
└── README.md                       # This file
```

## Test Categories

### Unit Tests (`unit/`)

Tests for individual components in isolation:

- **modbus_commands_test.go** - Tests parsing logic for all command types (voltage, current, power, frequency, power factor, energy)
- **config_test.go** - Tests configuration loading, validation, and parsing
- **factory_test.go** - Tests command factory creation and all supported command types

### Integration Tests (`integration/`)

Tests for component interactions:

- **groups_integration_test.go** - Tests group creation, execution, and end-to-end workflows

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
