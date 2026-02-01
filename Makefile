# Moneta Makefile
.PHONY: build run test clean docker install

# Build variables
BINARY=moneta
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY) ./cmd/moneta

# Build for release (multiple platforms)
build-all:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/moneta
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/moneta
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/moneta

# Run the server
run:
	go run ./cmd/moneta serve

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	go test -bench=. -benchmem ./internal/simd/...
	go test -bench=. -benchmem ./internal/store/sqlite/...

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-*
	rm -f coverage.out coverage.html

# Build Docker image
docker:
	docker build -t moneta:latest .

# Run with Docker Compose
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Install locally
install: build
	cp $(BINARY) /usr/local/bin/

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Development: run with hot reload (requires air)
dev:
	air -c .air.toml

# Show help
help:
	@echo "Moneta - Local-first code memory system"
	@echo ""
	@echo "Usage:"
	@echo "  make build       Build the binary"
	@echo "  make run         Run the server"
	@echo "  make test        Run tests"
	@echo "  make docker      Build Docker image"
	@echo "  make install     Install to /usr/local/bin"
	@echo "  make clean       Clean build artifacts"
