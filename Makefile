.PHONY: help build test test-verbose clean run run-terminal install lint fmt vet coverage

# Default target
help:
	@echo "Available commands:"
	@echo "  make build         - Build the chess-ai binary"
	@echo "  make test          - Run all tests"
	@echo "  make test-verbose  - Run all tests with verbose output"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make run           - Run in web mode"
	@echo "  make run-terminal  - Run in terminal mode"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run golangci-lint (if installed)"
	@echo "  make install       - Install the binary"

# Build the application
build:
	go build -o chess-ai .

# Run all tests
test:
	go test ./...

# Run all tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run in web mode
run: build
	./chess-ai

# Run in terminal mode
run-terminal: build
	./chess-ai --terminal

# Install the binary
install:
	go install .

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (if installed)
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping..."

# Clean build artifacts
clean:
	rm -f chess-ai chess-ai.exe
	rm -f coverage.out coverage.html
	rm -f neural/weights.gob
	rm -f stats/games.json
	go clean
