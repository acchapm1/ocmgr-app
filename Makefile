BINARY_NAME := ocmgr
MODULE := github.com/acchapm1/ocmgr-app
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/internal/cli.Version=$(VERSION)"
GO := /usr/bin/go

.PHONY: all build install clean test lint run

all: build

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/ocmgr

install: build
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
	cp bin/$(BINARY_NAME) $(HOME)/go/bin/$(BINARY_NAME) 2>/dev/null || \
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

clean:
	rm -rf bin/ dist/
	$(GO) clean

test:
	$(GO) test ./... -v

lint:
	$(GO) vet ./...

run: build
	./bin/$(BINARY_NAME)

# Development: build and run with args
# Usage: make dev ARGS="init --profile base ."
dev: build
	./bin/$(BINARY_NAME) $(ARGS)
