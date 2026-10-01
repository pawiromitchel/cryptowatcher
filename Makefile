.PHONY: all build test test-unit test-e2e cover vet fmt run run-mock clean release

BINARY_NAME=cryptowatcher
DIST_DIR=dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-s -w -X main.version=$(VERSION)

all: fmt vet test build

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/cryptowatcher

run:
	go run ./cmd/cryptowatcher

run-mock:
	go run ./cmd/cryptowatcher -mock

fmt:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

test-unit:
	go test -race -count=1 ./internal/...

test-e2e:
	go test -race -count=1 -v ./e2e/...

cover:
	go test -count=1 -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | tail -1

clean:
	rm -rf $(BINARY_NAME) $(DIST_DIR) coverage.out

release: clean
	mkdir -p $(DIST_DIR)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/cryptowatcher
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/cryptowatcher
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/cryptowatcher
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/cryptowatcher
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/cryptowatcher
	@echo "All binaries built in $(DIST_DIR)/"
