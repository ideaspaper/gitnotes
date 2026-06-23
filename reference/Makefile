.PHONY: build install clean test lint release release-snapshot

# Build variables
BINARY_NAME := projector
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/ideaspaper/projector/cmd.version=$(VERSION)

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Install to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Test release locally (no publish)
release-snapshot:
	goreleaser release --snapshot --clean

# Create a new release (requires GITHUB_TOKEN in .env)
release:
	set -a && source .env && set +a && goreleaser release --clean

# Create a new tag and release
# Usage: make release-tag VERSION=v1.0.0
release-tag:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release-tag VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	set -a && source .env && set +a && goreleaser release --clean

# Run the application
run:
	go run . $(ARGS)

# Development: build and run
dev: build
	./$(BINARY_NAME) $(ARGS)
