# PowerShell test runner script for CHINT MQTT-Modbus Bridge

Write-Host "Running all tests for CHINT MQTT-Modbus Bridge..." -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ""

Set-Location -Path "tests"

Write-Host "Running unit tests..." -ForegroundColor Yellow
go test ./unit/... -v -cover

Write-Host ""
Write-Host "Running integration tests..." -ForegroundColor Yellow
go test ./integration/... -v -cover

Write-Host ""
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Test execution completed!" -ForegroundColor Green
