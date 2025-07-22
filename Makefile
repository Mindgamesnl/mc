# Makefile for mc - Minecraft Server Utility

BINARY_NAME=mc
VERSION?=dev
LDFLAGS=-ldflags="-w -s -X main.Version=$(VERSION)"

.PHONY: build test clean install release-local help

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build $(LDFLAGS) -o $(BINARY_NAME)

test: ## Run tests with verbose output
	go test -v -race -coverprofile=coverage.out

test-coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f mc-*
	rm -f *.jar
	rm -f mc.yml
	rm -f eula.txt

install: build ## Install binary to /usr/local/bin
	sudo mv $(BINARY_NAME) /usr/local/bin/

release-local: ## Build release binaries locally
	@echo "Building release binaries..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64
	@echo "Release binaries built:"
	@ls -la $(BINARY_NAME)-*

checksums: release-local ## Generate checksums for release binaries
	sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "Checksums generated:"
	@cat checksums.txt

deps: ## Download and verify dependencies
	go mod download
	go mod verify

fmt: ## Format Go code
	go fmt ./...

mod-tidy: ## Tidy and verify the go.mod file
	go mod tidy
	go mod verify

dev-setup: ## Set up development environment
	@echo "Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Development setup complete!"

all: deps lint test build ## Run all checks and build

.DEFAULT_GOAL := help
