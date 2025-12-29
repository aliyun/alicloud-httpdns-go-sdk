# HTTPDNS Go SDK Makefile

.PHONY: all build test test-unit test-integration test-e2e test-benchmark clean fmt vet lint coverage help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Package info
PKG=./pkg/httpdns
INTERNAL_PKG=./internal/...
TEST_PKG=./test/...
EXAMPLES_PKG=./examples/...

# Test parameters
TIMEOUT=30s
COVERAGE_OUT=reports/coverage.out
COVERAGE_HTML=reports/coverage.html

all: fmt vet test build

# Build the library
build:
	$(GOBUILD) -v $(PKG)

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	@mkdir -p reports
	$(GOTEST) -v -timeout $(TIMEOUT) -coverprofile=$(COVERAGE_OUT) $(PKG) $(INTERNAL_PKG)
	$(GOCMD) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -timeout $(TIMEOUT) $(TEST_PKG)

# Run end-to-end tests (requires environment variables)
test-e2e:
	@echo "Running end-to-end tests..."
	$(GOTEST) -v -timeout 60s -tags=e2e $(TEST_PKG)

# Run benchmark tests
test-benchmark:
	@echo "Running benchmark tests..."
	$(GOTEST) -v -bench=. -benchmem $(TEST_PKG)

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Vet code
vet:
	@echo "Vetting code..."
	$(GOVET) $(PKG) $(INTERNAL_PKG) $(TEST_PKG) $(EXAMPLES_PKG)

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

# Generate coverage report
coverage: test-unit
	@echo "Coverage report generated at $(COVERAGE_HTML)"
	$(GOCMD) tool cover -func=$(COVERAGE_OUT)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf reports/

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run examples
run-basic:
	$(GOCMD) run examples/basic/main.go

run-advanced:
	$(GOCMD) run examples/advanced/main.go

run-server:
	$(GOCMD) run examples/server/main.go

run-no-retry:
	$(GOCMD) run examples/no_retry/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Format, vet, test, and build"
	@echo "  build         - Build the library"
	@echo "  test          - Run unit and integration tests"
	@echo "  test-unit     - Run unit tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e      - Run end-to-end tests (requires env vars)"
	@echo "  test-benchmark - Run benchmark tests"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  coverage      - Generate coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  run-basic     - Run basic example"
	@echo "  run-advanced  - Run advanced example"
	@echo "  run-server    - Run server example"
	@echo "  run-no-retry  - Run no-retry example"
	@echo "  help          - Show this help"