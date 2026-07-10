BINARY := gint
CMD := ./cmd/gint

# Version stamp: the latest git tag (e.g. v1.0.0), else the short commit, else
# "dev". Injected into main.version via -ldflags at build/install time.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: help build install test lint fmt

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help     Show this help message"
	@echo "  build    Build the gint binary"
	@echo "  install  Install gint to GOPATH/bin"
	@echo "  test     Run tests with race detector"
	@echo "  lint     Run golangci-lint"
	@echo "  fmt      Format code with gofmt and run go vet"

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

install:
	go install $(LDFLAGS) $(CMD)

test:
	go test -race ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	go vet ./...
