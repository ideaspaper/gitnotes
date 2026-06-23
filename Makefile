.PHONY: build install test test-coverage vet fmt check clean run release-snapshot release release-tag

BINARY := gitnotes
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X gitnotes/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

fmt:
	gofmt -l -w .

check: fmt vet test

run:
	go run -ldflags "$(LDFLAGS)" . $(ARGS)

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist/

release-snapshot:
	goreleaser release --snapshot --clean

release:
	set -a && [ -f .env ] && . ./.env; set +a && goreleaser release --clean

release-tag:
	@if [ -z "$(TAG)" ]; then echo "Usage: make release-tag TAG=v1.0.0"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	set -a && [ -f .env ] && . ./.env; set +a && goreleaser release --clean
