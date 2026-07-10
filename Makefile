BINARY := gint
CMD := ./cmd/gint

# Version stamp: the latest git tag (e.g. v1.0.0), else the short commit, else
# "dev". Injected into main.version via -ldflags at build/install time.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test lint fmt

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
