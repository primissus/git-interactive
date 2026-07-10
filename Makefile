BINARY := gint
CMD := ./cmd/gint

.PHONY: build install test lint fmt

build:
	go build -o $(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	go vet ./...
