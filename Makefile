.PHONY: build run test clean lint fmt help

# Default target
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  run      - Run the application"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"
	@echo "  lint     - Run golangci-lint"
	@echo "  fmt      - Format code"

# Build the application
build:
	go build -o bin/http-mock-server ./cmd

# Run the application
run:
	go run ./cmd

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run linter (requires golangci-lint to be installed)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	go mod tidy

# Install dependencies
deps:
	go mod download
	go mod verify
