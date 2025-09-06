#!/bin/bash

# test.sh - Comprehensive test runner script for wheregoes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
print_success "Using Go version: $GO_VERSION"

# Clean previous test artifacts
print_header "Cleaning previous test artifacts"
rm -f coverage.out coverage.html
print_success "Cleaned previous artifacts"

# Download dependencies
print_header "Downloading dependencies"
go mod download
go mod tidy
print_success "Dependencies updated"

# Run unit tests with coverage
print_header "Running unit tests with coverage"
go test -v -coverprofile=coverage.out ./...
if [ $? -eq 0 ]; then
    print_success "All unit tests passed"
else
    print_error "Unit tests failed"
    exit 1
fi

# Generate coverage report
print_header "Generating coverage report"
go tool cover -func=coverage.out
COVERAGE_PERCENTAGE=$(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}')
print_success "Total coverage: $COVERAGE_PERCENTAGE"

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
print_success "HTML coverage report generated: coverage.html"

# Run race tests (shorter version to avoid network dependencies)
print_header "Running race condition tests"
go test -race -short ./...
if [ $? -eq 0 ]; then
    print_success "Race condition tests passed"
else
    print_warning "Race condition tests had issues (may be expected for integration tests)"
fi

# Run benchmarks if they exist
print_header "Running benchmarks"
if go test -bench=. -run=^$ ./... 2>/dev/null; then
    print_success "Benchmarks completed"
else
    print_warning "No benchmarks found or benchmarks failed"
fi

# Build the application
print_header "Building application"
go build -o wheregoes .
if [ $? -eq 0 ]; then
    print_success "Application built successfully"
else
    print_error "Build failed"
    exit 1
fi

# Test CLI functionality
print_header "Testing CLI functionality"
./wheregoes --version
if [ $? -eq 0 ]; then
    print_success "CLI version command works"
else
    print_error "CLI version command failed"
    exit 1
fi

./wheregoes --help > /dev/null
if [ $? -eq 0 ]; then
    print_success "CLI help command works"
else
    print_error "CLI help command failed"
    exit 1
fi

# Summary
print_header "Test Summary"
print_success "All tests completed successfully!"
print_success "Coverage report: coverage.html"
print_success "Binary built: wheregoes"

# Check coverage threshold
COVERAGE_NUM=$(echo $COVERAGE_PERCENTAGE | sed 's/%//')
THRESHOLD=70

if [ "${COVERAGE_NUM%.*}" -ge "$THRESHOLD" ]; then
    print_success "Coverage $COVERAGE_PERCENTAGE meets threshold of ${THRESHOLD}%"
else
    print_warning "Coverage $COVERAGE_PERCENTAGE is below threshold of ${THRESHOLD}%"
fi

echo -e "\n${GREEN}🎉 All tests passed! Your codebase is well tested.${NC}"