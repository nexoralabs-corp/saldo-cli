BIN ?= saldo
DIST_DIR ?= dist
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: tidy test build build-prod clean

tidy:
	go mod tidy

test:
	go test ./...

build:
	go build -o $(BIN) ./cmd/saldo

build-prod:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BIN) ./cmd/saldo

clean:
	rm -rf $(BIN) $(DIST_DIR)
