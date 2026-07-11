BINARY := gint
CMD := ./cmd/gint

# VERSION is the tracked semver release (bump it by hand, e.g. in a release
# commit) — stable and easy to eyeball regardless of git tags. COMMIT pins that
# build to an exact source state, so `gint version` can tell two builds of the
# same VERSION apart. Both are injected via -ldflags at build/install time.
VERSION := $(shell cat VERSION 2>/dev/null || echo 0.0.0)
COMMIT := $(shell git describe --always --dirty 2>/dev/null || echo unknown)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

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
