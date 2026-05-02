BINARY := plexus
PKG := ./cmd/plexus
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/r13v/plexus/internal/cli.version=$(VERSION)

.PHONY: build install test vet fmt tidy lint check clean run

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) $(PKG)

install:
	go install -ldflags '$(LDFLAGS)' $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

check: fmt vet lint test

clean:
	rm -f $(BINARY)

run:
	go run $(PKG) $(ARGS)
