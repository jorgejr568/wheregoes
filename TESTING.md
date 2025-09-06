# Testing Documentation

## Overview

This document describes the comprehensive test suite implemented for the Wheregoes CLI/server project. The tests are designed to ensure code quality, prevent regressions, and maintain reliability for future changes.

## Test Structure

### Unit Tests

#### 1. Utils Package (`internal/utils/`)
- **File**: `url.utils_test.go`
- **Coverage**: 100%
- **Tests**: URL validation functionality
- **Key scenarios**: HTTP/HTTPS URLs, invalid protocols, malformed URLs

#### 2. Clients Package (`internal/clients/`)
- **File**: `fetcher.client_test.go` 
- **Coverage**: 90.9%
- **Tests**: HTTP client functionality with mock servers
- **Key scenarios**: 
  - Successful requests (200, 404, 500 status codes)
  - Redirect handling (Location headers)
  - Request headers (User-Agent, Accept)
  - Context timeout handling
  - Invalid URLs
  - Redirect prevention (no automatic following)

#### 3. Services Package (`internal/services/`)
- **File**: `tracker.service_test.go`
- **Coverage**: ~88%
- **Tests**: Core URL tracking logic with mock clients
- **Key scenarios**:
  - Single URL tracking (no redirects)
  - Redirect following (absolute and relative URLs)
  - Circular redirect detection
  - Error handling (network failures)
  - Channel-based tracking for real-time updates

#### 4. Set Package (`internal/pkg/set/`)
- **File**: `set_test.go`
- **Coverage**: 100%
- **Tests**: Generic set data structure
- **Key scenarios**: Add, Remove, Contains, Len, Values operations

### Integration Tests

#### 1. Server Package (`internal/server/`)
- **File**: `server_test.go`
- **Coverage**: ~10% (focused on core endpoints)
- **Tests**: HTTP server and WebSocket functionality
- **Key scenarios**:
  - Health endpoint
  - CORS configuration
  - WebSocket upgrade and origin checking
  - Environment variable handling

#### 2. Command Package (`internal/cmd/`)
- **Files**: `root.command_test.go`, `track.command_test.go`, `serve.command_test.go`
- **Tests**: CLI command functionality
- **Key scenarios**:
  - Command structure and help text
  - Flag parsing (version, port, json)
  - Argument validation
  - Subcommand relationships

## Test Infrastructure

### Build and Coverage Tools

#### Makefile
Provides convenient targets for:
- `make test` - Run all tests
- `make coverage` - Generate coverage reports
- `make coverage-html` - Generate HTML coverage report
- `make test-race` - Run race condition tests
- `make benchmark` - Run performance benchmarks

#### Test Script (`scripts/test.sh`)
Comprehensive test runner that:
- Validates Go installation
- Runs tests with coverage reporting
- Generates HTML reports
- Tests CLI functionality
- Provides colored output and summary

#### GitHub Actions CI (`.github/workflows/ci.yml`)
Automated CI/CD pipeline that:
- Tests against multiple Go versions (1.20, 1.21, 1.22)
- Runs race condition detection
- Linting with golangci-lint
- Docker build and integration testing
- Coverage reporting to Codecov

## Coverage Summary

### Overall Coverage by Package:
- **Utils**: 100% - Complete coverage of URL validation
- **Set**: 100% - Complete coverage of set operations  
- **Clients**: 90.9% - Excellent coverage of HTTP client functionality
- **Services**: ~88% - Good coverage of core tracking logic
- **Server**: ~10% - Basic endpoint testing (integration focused)
- **Total Core Logic**: 95.5%

### What's Tested:
✅ URL validation and parsing  
✅ HTTP client with various response codes  
✅ Redirect handling (absolute, relative, circular)  
✅ Error handling and timeouts  
✅ Set data structure operations  
✅ CLI command structure and flags  
✅ Server health checks and CORS  
✅ WebSocket origin validation  

### Areas for Improvement:
- Server integration tests (WebSocket message handling)
- CLI integration with actual network requests
- More edge cases in redirect handling
- Performance benchmarking

## Running Tests

### Quick Start
```bash
# Run all tests
go test ./...

# Run with coverage
make coverage

# Run comprehensive test suite
./scripts/test.sh

# Run only unit tests (no network)
go test -short ./...
```

### Test Categories
- **Unit Tests**: Fast, isolated, no external dependencies
- **Integration Tests**: Test component interactions
- **CLI Tests**: Test command-line interface functionality  
- **Server Tests**: Test HTTP/WebSocket endpoints

## Test Philosophy

The test suite follows these principles:

1. **Isolation**: Tests don't depend on external services
2. **Mocking**: Mock external dependencies (HTTP clients, etc.)
3. **Coverage**: Aim for high coverage of business logic
4. **Reliability**: Tests should be deterministic and fast
5. **Documentation**: Tests serve as usage examples

## Future Enhancements

- Add benchmark tests for performance regression detection
- Implement property-based testing for URL validation
- Add end-to-end tests with real server instances
- Improve WebSocket integration testing
- Add load testing for concurrent request handling

## Conclusion

The test suite provides excellent coverage of core functionality and ensures the reliability of the Wheregoes application. The combination of unit tests, integration tests, and CI automation provides confidence for making future changes without introducing regressions.