APP := mavlink-doctor
MODULE := github.com/hitraa/mavlink-doctor
BIN_DIR := bin
DIST_DIR := dist

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | awk '{print $$3}')

LDFLAGS := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Build=$(BUILD)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(DATE)' \
	-X '$(MODULE)/internal/version.GoVersion=$(GO_VERSION)'

.PHONY: all build cross-build test test-race test-coverage fmt vet tidy clean run

all: tidy fmt vet test build

build:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building $(APP) $(VERSION)..."
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP) ./cmd/mavlink-doctor

cross-build:
	@./build.sh

test:
	@echo "==> Running tests..."
	go test -v ./...

test-race:
	@echo "==> Running tests with race detector..."
	go test -race -v ./...

test-coverage:
	@echo "==> Running test coverage..."
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

fmt:
	@echo "==> Formatting code..."
	gofmt -s -w .

vet:
	@echo "==> Vetting code..."
	go vet ./...

tidy:
	@echo "==> Tidying Go modules..."
	go mod tidy

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out coverage.html *.test

run: build
	@./$(BIN_DIR)/$(APP)
