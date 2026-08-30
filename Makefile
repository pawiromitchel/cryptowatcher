.PHONY: all build test run clean release

BINARY_NAME=cryptowatcher
DIST_DIR=dist

all: test build

build:
	go build -o $(BINARY_NAME) ./cmd/cryptowatcher

run:
	go run ./cmd/cryptowatcher

test:
	go test -v ./...

clean:
	rm -rf $(BINARY_NAME) $(DIST_DIR)

release: clean
	mkdir -p $(DIST_DIR)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/cryptowatcher
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/cryptowatcher
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/cryptowatcher
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/cryptowatcher
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/cryptowatcher
	@echo "All binaries built in $(DIST_DIR)/"
