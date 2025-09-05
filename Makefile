.PHONY: all build clean test receiver sender help

# Default target
all: receiver

# Build the receiver binary
receiver:
	@echo "Building receiver..."
	@go build -o bin/receiver ./cmd/receiver

# Build all binaries
build: receiver

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run go mod tidy
tidy:
	@echo "Tidying modules..."
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter (if golangci-lint is installed)
lint:
	@echo "Running linter..."
	@golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build receiver (default)"
	@echo "  build        - Build all binaries"
	@echo "  receiver     - Build receiver binary"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  tidy         - Run go mod tidy"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter (requires golangci-lint)"
	@echo "  deps         - Download dependencies"
	@echo "  help         - Show this help"
