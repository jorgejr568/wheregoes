# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=wheregoes
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

.PHONY: all build clean test coverage coverage-html test-verbose help deps

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./

test:
	$(GOTEST) -v ./...

test-short:
	$(GOTEST) -short ./...

coverage:
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

coverage-html: coverage
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-verbose:
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) ./...

test-race:
	$(GOTEST) -race -short ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

docker-build:
	docker build -t wheregoes .

docker-run:
	docker run -p 8080:8080 wheregoes

lint:
	golangci-lint run

benchmark:
	$(GOTEST) -bench=. -benchmem ./...

help:
	@echo "Available targets:"
	@echo "  all          - Run tests and build"
	@echo "  build        - Build the binary"
	@echo "  test         - Run all tests"
	@echo "  test-short   - Run tests with -short flag"
	@echo "  test-race    - Run tests with race detector"
	@echo "  test-verbose - Run tests with verbose output and coverage"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  coverage-html- Generate HTML coverage report"
	@echo "  benchmark    - Run benchmarks"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  build-linux  - Build for Linux"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  lint         - Run linter (requires golangci-lint)"
	@echo "  help         - Show this help message"