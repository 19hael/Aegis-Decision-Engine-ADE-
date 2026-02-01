.PHONY: all build run test lint clean up down

# Variables
BINARY_NAME=ade-server
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/ade-server

# Run the server locally
run:
	go run ./cmd/ade-server

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Docker Compose commands
up:
	docker-compose -f deployments/docker-compose.yaml up -d

down:
	docker-compose -f deployments/docker-compose.yaml down

# View logs
logs:
	docker-compose -f deployments/docker-compose.yaml logs -f

# Full dev environment
dev: up
	@echo "Waiting for services to be ready..."
	@sleep 5
	go run ./cmd/ade-server

# Database migrations
migrate-up:
	migrate -path migrations -database "postgres://ade:ade@localhost:5432/ade?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://ade:ade@localhost:5432/ade?sslmode=disable" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq "$$name"

# Install dependencies
deps:
	go mod download
	go mod tidy
