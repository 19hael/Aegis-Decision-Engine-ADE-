# Aegis Decision Engine Makefile

.PHONY: all build build-cli build-all test test-verbose test-coverage lint clean run run-docker migrate-up migrate-down proto generate docker-build docker-push help

# Variables
BINARY_NAME=ade-server
CLI_NAME=ade-cli
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
DOCKER_IMAGE=ade:latest

# Default target
all: build

## build: Build the server binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/ade-server

## build-cli: Build the CLI binary
build-cli:
	@echo "Building $(CLI_NAME)..."
	go build $(LDFLAGS) -o bin/$(CLI_NAME) ./cmd/ade-cli

## build-all: Build all binaries
build-all: build build-cli

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v -race ./... 2>&1 | grep -E "(PASS|FAIL|RUN|---)"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-coverage-func: Show coverage by function
test-coverage-func:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./... || go vet ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

## run: Run the server locally
run: build
	@echo "Starting server..."
	./bin/$(BINARY_NAME)

## run-docker: Run with Docker Compose
run-docker:
	@echo "Starting with Docker Compose..."
	docker-compose -f deployments/docker-compose.yaml up -d

## stop-docker: Stop Docker Compose
stop-docker:
	@echo "Stopping Docker Compose..."
	docker-compose -f deployments/docker-compose.yaml down

## logs: View Docker logs
logs:
	docker-compose -f deployments/docker-compose.yaml logs -f

## migrate-up: Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	migrate -path migrations -database "postgres://ade:ade@localhost:5432/ade?sslmode=disable" up

## migrate-down: Run database migrations down
migrate-down:
	@echo "Running migrations down..."
	migrate -path migrations -database "postgres://ade:ade@localhost:5432/ade?sslmode=disable" down

## migrate-create: Create a new migration
migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq "$$name"

## proto: Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

## generate: Run all code generation
generate: proto
	@echo "Running code generation..."
	go generate ./...

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

## docker-push: Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE)

## install: Install binaries to GOPATH/bin
install: build-all
	@echo "Installing binaries..."
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/
	cp bin/$(CLI_NAME) $(GOPATH)/bin/

## demo: Run the demo script
demo:
	@echo "Running demo..."
	./scripts/demo.sh

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## security-scan: Run security scan
security-scan:
	@echo "Running security scan..."
	gosec ./... || echo "Install gosec: go install github.com/securego/gosec/v2/cmd/gosec@latest"

## check: Run all checks
check: fmt vet lint test

## ci: CI pipeline
ci: deps check test-coverage

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## //g'

# Development environment
## dev: Start development environment
dev: run-docker
	@echo "Waiting for services..."
	@sleep 5
	make migrate-up
	@echo "Development environment ready!"

## dev-stop: Stop development environment
dev-stop: stop-docker
