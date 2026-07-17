BINARY := gint
CMD := ./cmd/gint

# VERSION is the tracked semver release (bump it by hand, e.g. in a release
# commit) — stable and easy to eyeball regardless of git tags. COMMIT pins that
# build to an exact source state, so `gint version` can tell two builds of the
# same VERSION apart. Both are injected via -ldflags at build/install time.
VERSION := $(shell cat VERSION 2>/dev/null || echo 0.0.0)
COMMIT := $(shell git describe --always --dirty 2>/dev/null || echo unknown)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# Cross-compile matrix for `make release`. No cgo, so every target builds from
# any host with the Go toolchain alone — no C cross-compilers needed.
DIST := dist
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: help build install test lint fmt release

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
	@echo "  release  Cross-compile all platforms into $(DIST)/ as zips + checksums"

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

# Build every platform in PLATFORMS, bundling each binary with LICENSE + README
# into its own zip, then emit SHA-256 checksums. Runs tests first so a broken
# build never ships. Upload the $(DIST)/*.zip and checksums.txt to a release.
release: test
	@rm -rf $(DIST)
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		name=$(BINARY)_$(VERSION)_$${os}_$${arch}; \
		echo "-> $$name"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build $(LDFLAGS) -o "$(DIST)/$$name/$(BINARY)$$ext" $(CMD) || exit 1; \
		cp LICENSE README.md "$(DIST)/$$name/"; \
		( cd $(DIST) && zip -qr "$$name.zip" "$$name" && rm -rf "$$name" ) || exit 1; \
	done
	@cd $(DIST) && shasum -a 256 *.zip > checksums.txt
	@echo ""
	@echo "Artifacts in $(DIST)/:"
	@ls -1 $(DIST)
