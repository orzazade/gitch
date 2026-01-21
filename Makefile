.PHONY: build test lint clean install coverage help

# Build variables
BINARY_NAME=gitch
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## install: Install to GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: Run tests
test:
	go test -v -race ./...

## coverage: Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter
lint:
	golangci-lint run

## fmt: Format code
fmt:
	go fmt ./...
	gofmt -s -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## release-dry: Dry run of goreleaser
release-dry:
	goreleaser release --snapshot --clean

## all: Run fmt, vet, lint, test, and build
all: fmt vet lint test build
