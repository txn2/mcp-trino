.PHONY: all build test test-short test-integration lint clean coverage security help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-trino
COVERAGE_FILE=coverage.out

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

all: lint test build ## Run lint, test, and build

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/mcp-trino

build-all: ## Build for all platforms
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/mcp-trino
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/mcp-trino
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/mcp-trino
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/mcp-trino
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/mcp-trino

test: ## Run tests
	$(GOTEST) -v -race ./...

test-short: ## Run tests (short mode)
	$(GOTEST) -v -short ./...

test-integration: ## Run integration tests (requires Trino: make docker-trino)
	$(GOTEST) -v -tags=integration ./pkg/client/...

coverage: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

coverage-html: coverage ## Generate HTML coverage report
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

lint-fix: ## Run linters and fix issues
	golangci-lint run --fix --timeout=5m

fmt: ## Format code
	$(GOCMD) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/txn2/mcp-trino .; \
	fi

security: ## Run security scanners
	@echo "Running gosec..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo "Running govulncheck..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

tidy: ## Tidy and verify dependencies
	$(GOMOD) tidy
	$(GOMOD) verify

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* $(COVERAGE_FILE) coverage.html
	$(GOCMD) clean -cache -testcache

verify: tidy lint test ## Verify code quality (tidy, lint, test)

docker-trino: ## Start local Trino for testing
	docker run -d -p 8080:8080 --name trino-test trinodb/trino:latest || true
	@echo "Trino starting at http://localhost:8080"
	@echo "Wait a few seconds for it to be ready..."

docker-trino-stop: ## Stop local Trino
	docker stop trino-test || true
	docker rm trino-test || true

help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
