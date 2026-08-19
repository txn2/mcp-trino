.PHONY: all build build-all test test-short test-integration lint lint-fix fmt clean \
	coverage coverage-html patch-coverage security semgrep dead-code \
	build-check release-check mutate codeql tools-check tidy \
	verify verify-release verify-gates verify-sentinel verify-checks \
	verify-go verify-lint verify-build \
	docker-trino docker-trino-stop help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-trino
COVERAGE_FILE=coverage.out
GOVULN_REPORT=govulncheck-report.json

# Tool versions, pinned to what CI installs. tools-check refuses to run verify
# when a local tool drifts from these, because a local scanner that differs
# from CI's turns `make verify` into a gate for a suite nobody runs: a newer
# local gosec can drop a rule CI still enforces, and the diff ships green
# locally and red on the PR.
#
# Each pin below MUST equal the version in .github/workflows/ci.yml, and each is
# compared against the installed tool by tools-check. Do not add a pin here that
# nothing enforces: a decorative version constant reads as a guarantee and is not
# one.
#   GOLANGCI_LINT_VERSION -> the `version:` input of golangci-lint-action
#   GOSEC_VERSION         -> the `go install ...gosec@` line in the security job
#   GOVULNCHECK_VERSION   -> the `go install ...govulncheck@` line in the same job
GOLANGCI_LINT_VERSION := v2.11.4
GOSEC_VERSION := v2.28.0
GOVULNCHECK_VERSION := v1.1.4

# Coverage gates. COVERAGE_MIN mirrors codecov.yml's project target and
# PATCH_COVERAGE_MIN its patch target, so a diff that codecov would reject
# fails here first.
COVERAGE_MIN := 82
PATCH_COVERAGE_MIN := 80

# TOOLS_CHECK_STRICT=0 downgrades a version mismatch to a warning. Use it only
# with a stated reason; it re-opens the parity gap the pins exist to close.
TOOLS_CHECK_STRICT ?= 1

# The sentinel `verify` writes on success: the short SHA-256 of the working-tree
# diff it passed on. The pre-commit review gate compares it to the live diff, so
# a verify run cannot be credited to a tree it never saw. The hash computation
# must stay byte-identical to the gate's.
VERIFY_SENTINEL := .claude/.last-verify-passed

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

test: ## Run tests with race detection, writing the coverage profile
	@# -shuffle=on and -count=1 keep an order-dependent test from passing by
	@# luck and stop a cached PASS from standing in for a run. The profile is
	@# written here because coverage and patch-coverage both read it.
	$(GOTEST) -v -race -shuffle=on -count=1 -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...

test-short: ## Run tests (short mode)
	$(GOTEST) -v -short ./...

test-integration: ## Run integration tests (requires Trino: make docker-trino)
	$(GOTEST) -v -tags=integration ./pkg/client/...

coverage: ## Enforce the total coverage floor (reads the profile from `test`)
	@if [ ! -f $(COVERAGE_FILE) ]; then \
		echo "ERROR: $(COVERAGE_FILE) not found. Run 'make test' first."; \
		exit 1; \
	fi
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | tail -1
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep '^total:' | awk '{print $$NF}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_MIN)" | bc -l) -eq 1 ]; then \
		echo "FAIL: coverage $$COVERAGE% is below the $(COVERAGE_MIN)% floor (codecov.yml project target)"; \
		exit 1; \
	fi; \
	echo "Coverage $$COVERAGE% meets the $(COVERAGE_MIN)% floor."

patch-coverage: ## Enforce coverage of changed lines vs main (mirrors codecov patch)
	@PATCH_COVERAGE_THRESHOLD=$(PATCH_COVERAGE_MIN) COVERAGE_FILE=$(COVERAGE_FILE) ./scripts/patch-coverage.sh

coverage-html: ## Generate HTML coverage report (run `make test` first)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run golangci-lint + go vet
	@# Both run unconditionally. An earlier version skipped a missing tool with
	@# a friendly message, which reports success for a check that never ran —
	@# tools-check is where a missing tool is reported now.
	golangci-lint run --timeout=5m
	$(GOCMD) vet ./...

lint-fix: ## Run linters and fix issues
	golangci-lint run --fix --timeout=5m

fmt: ## Format code
	$(GOCMD) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/txn2/mcp-trino .; \
	fi

security: ## Run gosec + govulncheck (fail-closed)
	@echo "Running gosec..."
	gosec -quiet ./...
	@echo "Running govulncheck..."
	@# govulncheck exits 3 when our code calls a vulnerable symbol, whether or
	@# not a fixed version exists, and has no way to accept a finding. The gate
	@# judges the report against .govulncheck-allow.txt, where an accepted
	@# advisory carries the reason it is accepted and expires the moment a fix
	@# ships or the advisory stops being reported.
	@govulncheck -format json ./... > $(GOVULN_REPORT) 2>/dev/null; \
	status=$$?; \
	if [ $$status -ne 0 ] && [ $$status -ne 3 ]; then \
		echo "ERROR: govulncheck failed to run (exit $$status)"; \
		rm -f $(GOVULN_REPORT); \
		exit 1; \
	fi
	@python3 scripts/govulncheck-gate.py $(GOVULN_REPORT); \
	status=$$?; \
	rm -f $(GOVULN_REPORT); \
	exit $$status

semgrep: ## Run Semgrep SAST (not in verify — CI does not run it; see verify)
	semgrep scan --config p/golang --error --quiet .

dead-code: ## Report unreachable functions (advisory)
	@echo "Checking for dead code..."
	@OUTPUT=$$(deadcode ./... 2>&1 | grep -v "^$$") || true; \
	if [ -n "$$OUTPUT" ]; then \
		echo "Dead code detected (review for false positives; exported API is expected here):"; \
		echo "$$OUTPUT"; \
	else \
		echo "No dead code found."; \
	fi

build-check: ## Verify the build and the module graph
	$(GOCMD) build ./...
	$(GOMOD) verify

release-check: ## GoReleaser dry-run
	@# sbom and sign are skipped, as in mcp-data-platform: both need tools
	@# (syft, cosign) that only the release runner installs, and their absence
	@# would fail this check for a reason unrelated to the diff. The release
	@# workflow exercises them on the tag.
	goreleaser release --snapshot --clean --skip=publish,sign,sbom

mutate: ## Mutation testing (slow; verify-release only)
	@# gremlins is not version-pinned: it runs only in verify-release, never in
	@# CI, so there is no CI version for a pin to mirror.
	gremlins unleash --workers 1 --timeout-coefficient 3 --threshold-efficacy 60 ./pkg/...

codeql: ## CodeQL security-and-quality analysis (slow; verify-release only)
	@rm -rf /tmp/mcp-trino-codeql-db
	codeql database create /tmp/mcp-trino-codeql-db --language=go --source-root=. --overwrite
	codeql database analyze /tmp/mcp-trino-codeql-db \
		--format=sarif-latest --output=codeql-results.sarif \
		codeql/go-queries:codeql-suites/go-security-and-quality.qls

## The Docker check is not incidental: release-check runs goreleaser's dockers_v2
## block, which --skip=publish,sign,sbom does not cover, so every `make verify`
## needs a running daemon. Finding that out a minute in, from a Docker error
## unrelated to the diff, is what this check prevents.
tools-check: ## Verify local tool versions match the CI pins
	@echo "Checking required tools (presence AND pinned versions)..."
	@missing=""; mismatch=""; \
	if ! command -v golangci-lint > /dev/null 2>&1; then \
		missing="$$missing  golangci-lint: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)\n"; \
	else \
		v=$$(golangci-lint version 2>&1 | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
		case "$$v" in v*) ;; *) v="v$$v";; esac; \
		if [ "$$v" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			mismatch="$$mismatch  golangci-lint: have $$v, want $(GOLANGCI_LINT_VERSION) — go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)\n"; \
		fi; \
	fi; \
	if ! command -v gosec > /dev/null 2>&1; then \
		missing="$$missing  gosec: go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)\n"; \
	else \
		v=$$(gosec --version 2>&1 | grep -oE 'Version: v?[0-9]+\.[0-9]+\.[0-9]+' | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
		case "$$v" in v*) ;; *) v="v$$v";; esac; \
		if [ "$$v" != "$(GOSEC_VERSION)" ]; then \
			mismatch="$$mismatch  gosec: have $$v, want $(GOSEC_VERSION) — go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)\n"; \
		fi; \
	fi; \
	if ! command -v govulncheck > /dev/null 2>&1; then \
		missing="$$missing  govulncheck: go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"; \
	fi; \
	if command -v govulncheck > /dev/null 2>&1; then \
		v=$$(go version -m $$(command -v govulncheck) 2>/dev/null | awk '$$1=="mod" && $$2=="golang.org/x/vuln" {print $$3}'); \
		if [ -n "$$v" ] && [ "$$v" != "$(GOVULNCHECK_VERSION)" ]; then \
			mismatch="$$mismatch  govulncheck: have $$v, want $(GOVULNCHECK_VERSION) — go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"; \
		fi; \
	fi; \
	command -v deadcode > /dev/null 2>&1   || missing="$$missing  deadcode: go install golang.org/x/tools/cmd/deadcode@latest\n"; \
	command -v goreleaser > /dev/null 2>&1 || missing="$$missing  goreleaser: brew install goreleaser\n"; \
	command -v bc > /dev/null 2>&1         || missing="$$missing  bc: required by the coverage gate\n"; \
	command -v python3 > /dev/null 2>&1    || missing="$$missing  python3: required by scripts/govulncheck-gate.py\n"; \
	docker info > /dev/null 2>&1           || missing="$$missing  docker: a running Docker daemon (release-check builds images)\n"; \
	if [ -n "$$missing" ]; then \
		echo ""; \
		echo "FAIL: Missing required tools:"; \
		printf '%b' "$$missing"; \
		echo "Install all missing tools before running make verify."; \
		exit 1; \
	fi; \
	if [ -n "$$mismatch" ]; then \
		echo ""; \
		echo "FAIL: Tool version mismatch (local differs from CI-pinned)."; \
		echo "A local scanner that differs from CI's lets make verify pass on a diff CI rejects."; \
		echo ""; \
		printf '%b' "$$mismatch"; \
		echo ""; \
		if [ "$(TOOLS_CHECK_STRICT)" != "0" ]; then exit 1; fi; \
		echo "WARN: proceeding with mismatched tool versions (TOOLS_CHECK_STRICT=0)."; \
	else \
		echo "All required tools found at pinned CI versions."; \
	fi

tidy: ## Tidy and verify dependencies
	$(GOMOD) tidy
	$(GOMOD) verify

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* $(COVERAGE_FILE) coverage.html \
		$(GOVULN_REPORT) codeql-results.sarif
	$(GOCMD) clean -cache -testcache

verify: ## Run the CI-equivalent suite, then write the pre-commit gate sentinel
	@# The steps that REWRITE the working tree run first, one at a time,
	@# because everything after them reads what they produce: tidy rewrites
	@# go.mod/go.sum and fmt rewrites sources. Each is its own $(MAKE) line so
	@# they stay ordered even when the outer make was given -j.
	@$(MAKE) --no-print-directory tools-check
	@$(MAKE) --no-print-directory tidy
	@$(MAKE) --no-print-directory fmt
	@# semgrep and codeql are deliberately absent. The rule for this target is
	@# CI-equivalence, and .github/workflows/ci.yml runs neither; codeql.yml
	@# runs CodeQL on the pull request, where it blocks the merge, and its
	@# local run costs minutes per commit for a finding CI catches anyway.
	@# Both are available on demand (`make semgrep`, `make codeql`), and
	@# codeql runs in verify-release before a tag.
	@$(MAKE) --no-print-directory -j3 verify-checks
	@echo ""
	@echo "=== All checks passed ==="
	@$(MAKE) --no-print-directory verify-sentinel

verify-gates: ## Everything verify runs except the sentinel write (used by verify-release)
	@$(MAKE) --no-print-directory tools-check
	@$(MAKE) --no-print-directory tidy
	@$(MAKE) --no-print-directory fmt
	@$(MAKE) --no-print-directory -j3 verify-checks
	@echo ""
	@echo "=== All checks passed ==="

verify-sentinel: ## Write the pre-commit gate sentinel (run only after the gates pass)
	@# The short SHA-256 of the working-tree diff (staged + unstaged) at the moment
	@# the gates passed. The pre-commit review gate compares this hash to the live
	@# diff at commit time — if they match, this run is proof the checks passed on
	@# the exact code being committed. Hash computation MUST stay byte-identical to
	@# compute_diff_hash() in ~/.claude/hooks/review-gate.sh, or the gate rejects
	@# every commit.
	@mkdir -p .claude
	@{ git diff --cached HEAD 2>/dev/null; git diff 2>/dev/null; } \
		| shasum -a 256 | cut -c1-16 > $(VERIFY_SENTINEL)
	@echo "Wrote $(VERIFY_SENTINEL) (gate sentinel)"

verify-release: ## Full verify plus CodeQL and mutation testing (pre-tag)
	@# Ordered recipe lines, not prerequisites. As prerequisites these run
	@# concurrently under -j, and `gremlins unleash` rewrites sources in place to
	@# inject mutants — beside verify's own test and goreleaser lanes that mutated
	@# tree is what they would be reading, and what the sentinel hash is taken from.
	@# Ordering also keeps the sentinel honest: it is written by verify-sentinel
	@# below, after every gate here has passed, so a failed mutation run cannot
	@# leave a sentinel the pre-commit gate accepts.
	@$(MAKE) --no-print-directory verify-gates
	@$(MAKE) --no-print-directory codeql
	@$(MAKE) --no-print-directory mutate
	@$(MAKE) --no-print-directory verify-sentinel
	@echo ""
	@echo "=== Release verification complete (incl. CodeQL + mutation testing) ==="

verify-checks: verify-go verify-lint verify-build ## The read-only half of verify, run concurrently
	@:

verify-go: ## Test, coverage gates, security, dead code — ordered, they share the profile
	@echo "[lane start $$(date +%T)] verify-go"
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory coverage
	@$(MAKE) --no-print-directory patch-coverage
	@$(MAKE) --no-print-directory security
	@$(MAKE) --no-print-directory dead-code
	@echo "[lane done  $$(date +%T)] verify-go"

verify-lint: ## golangci-lint + go vet, alone in its lane
	@# golangci-lint refuses to start while another instance is running, so it
	@# gets a lane to itself rather than sharing one with another lint step.
	@echo "[lane start $$(date +%T)] verify-lint"
	@$(MAKE) --no-print-directory lint
	@echo "[lane done  $$(date +%T)] verify-lint"

verify-build: ## Build, module verification, and the release dry-run
	@echo "[lane start $$(date +%T)] verify-build"
	@$(MAKE) --no-print-directory build-check
	@$(MAKE) --no-print-directory release-check
	@echo "[lane done  $$(date +%T)] verify-build"

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
