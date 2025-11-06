.PHONY: build clean install test run-worker help

# Binary name
BINARY_NAME=queuectl

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) cmd/queuectl/main.go
	@echo "Build complete: ./$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf ~/.queuectl/workers/*.pid
	@echo "Clean complete"

# Install to system (requires sudo on Unix)
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run worker (for testing)
run-worker:
	@./$(BINARY_NAME) worker start --count 1

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install to system"
	@echo "  test        - Run tests"
	@echo "  tidy        - Tidy dependencies"
	@echo "  fmt         - Format code"
	@echo "  run-worker  - Start a test worker"
	@echo "  help        - Show this help message"