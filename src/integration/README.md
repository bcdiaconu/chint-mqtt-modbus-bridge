# Integration Tests

This directory contains integration tests for the MQTT-Modbus Bridge.

## Running Tests

```bash
# Run all integration tests
go test ./integration/... -v

# Run with coverage
go test ./integration/... -v -coverprofile=integration-coverage.out

# Run specific test
go test ./integration/... -v -run TestDeviceManagerCreation
```

## Test Structure

### diagnostics_test.go
Tests for device diagnostics manager functionality:

- **TestDeviceManagerCreation**: Verify manager initialization and metrics setup
- **TestRecordSuccess**: Test successful read recording
- **TestRecordError**: Test error recording
- **TestPublishDiscovery**: Test discovery message publishing for multiple devices
- **TestNilHomeAssistantConfig**: Regression test for nil pointer handling

### Mock Implementations

**MockPublisher**: Implements `mqtt.PublisherInterface` for testing:
- Tracks discovery and state publications
- No actual MQTT connections required
- Enables fast, isolated testing

## Test Coverage

Current integration test coverage focuses on:
- ✅ Device diagnostics manager
- ✅ Metrics tracking and state management
- ✅ Discovery publishing
- ✅ Error handling and edge cases

## Future Tests

Planned additions:
- [ ] MQTT connection and publishing integration
- [ ] Modbus gateway integration
- [ ] Strategy executor integration
- [ ] End-to-end configuration loading
- [ ] Concurrent access and thread safety

## Best Practices

1. **Isolation**: Tests use mocks to avoid external dependencies
2. **Fast**: No network calls, tests complete in milliseconds
3. **Deterministic**: No random data or timing dependencies
4. **Readable**: Clear test names and assertion messages
5. **Coverage**: Test both happy path and error cases

## CI/CD Integration

These tests are automatically run in GitHub Actions on:
- Every push
- Every pull request
- Manual workflow dispatch

See `.github/workflows/test.yml` for CI configuration.
