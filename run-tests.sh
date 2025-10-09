#!/bin/bash
# Test runner script for CHINT MQTT-Modbus Bridge

echo "Running all tests for CHINT MQTT-Modbus Bridge..."
echo "=================================================="
echo

cd tests || exit 1

echo "Running unit tests..."
go test ./unit/... -v -cover

echo
echo "Running integration tests..."
go test ./integration/... -v -cover

echo
echo "=================================================="
echo "Test execution completed!"
