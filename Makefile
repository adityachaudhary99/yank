BINARY := yank
PKG := github.com/adityachaudhary99/yank
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint tidy run clean
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/yank
test:
	go test ./...
lint:
	gofmt -l . && go vet ./...
tidy:
	go mod tidy
run: build
	./$(BINARY)
clean:
	rm -f $(BINARY)
