BINARY_NAME := ocmgr
MODULE := github.com/acchapm1/ocmgr
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/internal/cli.Version=$(VERSION)"
GO := $(shell command -v go 2>/dev/null || echo "go")

.PHONY: all build install clean test lint run dist

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

# Cross-compile for all release platforms
# Output goes to dist/ with the naming pattern the install script expects
dist: clean
	@echo "Building $(VERSION) for all platforms..."
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o dist/darwin_arm64/$(BINARY_NAME)  ./cmd/ocmgr
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o dist/linux_arm64/$(BINARY_NAME)   ./cmd/ocmgr
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o dist/linux_amd64/$(BINARY_NAME)   ./cmd/ocmgr
	@echo ""
	@echo "Packaging tarballs..."
	@for platform in darwin_arm64 linux_arm64 linux_amd64; do \
		os=$$(echo $$platform | cut -d_ -f1); \
		arch=$$(echo $$platform | cut -d_ -f2); \
		tarball="$(BINARY_NAME)_$(VERSION)_$${os}_$${arch}.tar.gz"; \
		tar -czf "dist/$${tarball}" -C "dist/$${platform}" $(BINARY_NAME); \
		echo "  dist/$${tarball}"; \
	done
	@echo ""
	@echo "Done. Upload dist/*.tar.gz to a GitHub Release."
