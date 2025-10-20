# CI/CD Integration Tests Fix

## Problem
GitHub Actions was failing with:
```
pattern ./integration/...: lstat ./integration/: no such file or directory
```

## Root Cause
Integration tests are located in `src/integration/` but the workflow was looking for them in `tests/integration/`.

## Solution

### 1. Updated GitHub Actions Workflow
**File**: `.github/workflows/go-tests.yml`

**Changes**:
- Run integration tests from `./src` working directory instead of `./tests`
- Updated coverage report paths:
  - Unit tests: `tests/unit-coverage.out`
  - Integration tests: `src/integration-coverage.out`
- Separate coverage report generation for unit and integration tests

**Before**:
```yaml
- name: Run integration tests
  working-directory: ./tests
  run: go test ./integration/... -v -coverprofile=integration-coverage.out
```

**After**:
```yaml
- name: Run integration tests
  working-directory: ./src
  run: go test ./integration/... -v -coverprofile=integration-coverage.out
```

### 2. Updated .gitignore
Added coverage files to ignore:
- `unit-coverage.out`
- `integration-coverage.out`
- `*-coverage.txt`
- `integration-coverage`

### 3. Removed Tracked Coverage File
```bash
git rm --cached src/integration-coverage
```

## Project Structure

```
chint-mqtt-modbus-bridge/
├── src/                          # Main application
│   ├── pkg/                      # Packages
│   │   ├── config/
│   │   ├── diagnostics/          # Device diagnostics
│   │   ├── mqtt/
│   │   └── ...
│   ├── integration/              # Integration tests (NEW)
│   │   ├── diagnostics_test.go
│   │   ├── suite_test.go
│   │   └── README.md
│   ├── go.mod
│   └── main.go
└── tests/                        # Unit tests
    ├── unit/
    │   └── ...
    └── go.mod
```

## Verification

### Local Testing
```bash
# From project root
cd src
go test ./integration/... -v

# With coverage
go test ./integration/... -v -coverprofile=integration-coverage.out
```

### CI/CD
GitHub Actions now successfully:
1. ✅ Runs unit tests from `tests/unit/`
2. ✅ Runs integration tests from `src/integration/`
3. ✅ Generates separate coverage reports
4. ✅ Uploads coverage artifacts

## Test Results
All 6 integration tests passing:
- ✅ TestDeviceManagerCreation
- ✅ TestRecordSuccess
- ✅ TestRecordError
- ✅ TestPublishDiscovery
- ✅ TestNilHomeAssistantConfig
- ✅ TestIntegrationSuite

## Impact
- **No breaking changes** to existing functionality
- **Improved CI/CD** reliability
- **Better organization** with integration tests alongside source code
- **Proper coverage tracking** for both unit and integration tests

## Related Commits
1. `test: Add integration tests for device diagnostics` (1817ebf)
2. `ci: Fix integration tests path in GitHub Actions` (3af7b5e)
