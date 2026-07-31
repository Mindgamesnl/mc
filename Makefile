# Makefile for mc - Minecraft Server Utility

BINARY_NAME=mc
VERSION?=dev
LDFLAGS=-ldflags="-w -s -X main.Version=$(VERSION)"

PLATFORMS=linux-amd64 linux-arm64 windows-amd64.exe darwin-amd64 darwin-arm64

.PHONY: help build test test-coverage lint vet fmt fmt-check clean install \
        release-local checksums deps mod-tidy dev-setup all

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build $(LDFLAGS) -o $(BINARY_NAME)

test: ## Run tests with the race detector and coverage
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and open the coverage report
	go tool cover -html=coverage.out

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go code
	go fmt ./...

fmt-check: ## Fail if any file is not gofmt-clean
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi

lint: fmt-check vet ## Run formatting and vet checks

clean: ## Clean build artifacts and generated server files
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.out checksums.txt
	rm -f *.jar mc.yml eula.txt server.properties

install: build ## Install binary to /usr/local/bin
	sudo install -m 0755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

release-local: ## Build release binaries locally
	@echo "Building release binaries..."
	@for platform in $(PLATFORMS); do \
		goos=$${platform%%-*}; \
		rest=$${platform#*-}; \
		goarch=$${rest%.exe}; \
		echo "  $$goos/$$goarch"; \
		GOOS=$$goos GOARCH=$$goarch CGO_ENABLED=0 \
			go build $(LDFLAGS) -o $(BINARY_NAME)-$$platform || exit 1; \
	done
	@echo "Release binaries built:"
	@ls -la $(BINARY_NAME)-*

checksums: release-local ## Generate checksums for release binaries
	sha256sum $(BINARY_NAME)-* > checksums.txt
	@cat checksums.txt

deps: ## Download and verify dependencies
	go mod download
	go mod verify

mod-tidy: ## Tidy and verify go.mod
	go mod tidy
	go mod verify

dev-setup: ## Set up development environment
	@echo "Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Development setup complete!"

all: deps lint test build ## Run all checks and build

.DEFAULT_GOAL := help
