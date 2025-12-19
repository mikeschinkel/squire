.PHONY: help test test-all lint build install clean fmt vet tidy ci ensure-valid work-sync

LINTER = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2"
BINARY_NAME = squire
INSTALL_PATH = /usr/local/bin

# Default target
help:
	@echo "Available targets:"
	@echo "  make help         - Show this help message"
	@echo "  make build        - Build the squire binary to ./bin/"
	@echo "  make install      - Install squire to $(INSTALL_PATH)"
	@echo "  make test         - Run unit tests"
	@echo "  make test-all     - Run all tests with coverage and race detection"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make fmt          - Format code with gofmt"
	@echo "  make vet          - Run go vet"
	@echo "  make tidy         - Run go mod tidy (all modules)"
	@echo "  make work-sync    - Sync workspace with go work sync"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make ci           - Run all CI checks (fmt, vet, lint, test-all)"
	@echo "  make ensure-valid - Run quality checks (tidy, test, lint, vet)"

# Go environment (requires jsonv2 experiment)
GOEXPERIMENT ?= jsonv2
GO := GOEXPERIMENT=$(GOEXPERIMENT) go

# Ensure all quality checks pass
ensure-valid: tidy test lint vet
	@echo "All quality checks passed!"

# Sync workspace
work-sync:
	@echo "Syncing workspace..."
	@go work sync

# Run unit tests
test:
	@echo "Running tests in squirepkg..."
	@cd squirepkg && $(GO) test -v ./... || exit 1
	@echo "Running tests in test module..."
	@cd test && $(GO) test -v ./... || exit 1

# Run all tests with coverage and race detection
test-all:
	@echo "Running tests with coverage in squirepkg..."
	@cd squirepkg && $(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./... || exit 1
	@echo "Running tests with race detection in test module..."
	@cd test && $(GO) test -v -race ./... || exit 1
	@echo "All tests passed!"

# Run linter
lint:
	@echo "Running linter in squirepkg..."
	@cd squirepkg && $(GO) run $(LINTER) run ./... --timeout=5m || exit 1
	@echo "Running linter in cmd..."
	@cd cmd && $(GO) run $(LINTER) run ./... --timeout=5m || exit 1

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet in squirepkg..."
	@cd squirepkg && $(GO) vet ./... || exit 1
	@echo "Running go vet in cmd..."
	@cd cmd && $(GO) vet ./... || exit 1

# Run go mod tidy for all modules
tidy:
	@echo "Running go work sync..."
	@go work sync
	@echo "Running go mod tidy in squirepkg..."
	@cd squirepkg && $(GO) mod tidy || exit 1
	@echo "Running go mod tidy in cmd..."
	@cd cmd && $(GO) mod tidy || exit 1
	@echo "Running go mod tidy in test..."
	@cd test && $(GO) mod tidy || exit 1

# Build the squire binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@cd cmd && $(GO) build -o ../bin/$(BINARY_NAME) . || exit 1
	@echo "Built to ./bin/$(BINARY_NAME)"

# Install squire to system path
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@cp ./bin/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin
	@rm -f squirepkg/coverage.txt test/coverage.txt
	@cd squirepkg && $(GO) clean
	@cd cmd && $(GO) clean
	@cd test && $(GO) clean
	@echo "Clean complete!"

# Run all CI checks locally
ci: fmt vet lint test-all
	@echo "All CI checks passed!"
