# Makefile for backend_v3 testing
# Usage: make test, make test-unit, make test-integration, etc.

.PHONY: help test test-unit test-integration test-verbose test-unit-verbose test-integration-verbose test-all test-coverage clean

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "Available targets:"
	@echo ""
	@echo "  make test                    - Run all unit tests (non-verbose)"
	@echo "  make test-unit               - Run unit tests only (non-verbose)"
	@echo "  make test-integration        - Run integration tests only (non-verbose)"
	@echo "  make test-verbose            - Run all tests with verbose output"
	@echo "  make test-unit-verbose       - Run unit tests with verbose output"
	@echo "  make test-integration-verbose - Run integration tests with verbose output"
	@echo "  make test-all                - Run all tests including integration (verbose)"
	@echo "  make test-coverage           - Run tests with coverage report"
	@echo "  make test-quick              - Run quick unit tests (exclude slow tests)"
	@echo "  make clean                   - Clean test cache and artifacts"
	@echo ""

## test: Run unit tests only (default, non-verbose)
test: test-unit

## test-unit: Run unit tests excluding integration tests
test-unit:
	@echo "🧪 Running unit tests (non-verbose)..."
	@go test -timeout 30s ./domain/... ./jobs/... -run '^Test' -short

## test-integration: Run integration tests only
test-integration:
	@echo "🔗 Running integration tests (non-verbose)..."
	@go test -timeout 60s ./integration_tests/...

## test-verbose: Run all tests with verbose output
test-verbose:
	@echo "🧪 Running all tests (verbose)..."
	@go test -v -timeout 30s ./domain/... ./jobs/... ./integration_tests/...

## test-unit-verbose: Run unit tests with verbose output
test-unit-verbose:
	@echo "🧪 Running unit tests (verbose)..."
	@go test -v -timeout 30s ./domain/... ./jobs/... -run '^Test' -short

## test-integration-verbose: Run integration tests with verbose output
test-integration-verbose:
	@echo "🔗 Running integration tests (verbose)..."
	@go test -v -timeout 60s ./integration_tests/...

## test-all: Run all tests including integration (verbose by default)
test-all:
	@echo "🚀 Running all tests (unit + integration, verbose)..."
	@go test -v -timeout 60s ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "📊 Running tests with coverage..."
	@go test -timeout 60s -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

## test-quick: Run quick unit tests (exclude integration and slow tests)
test-quick:
	@echo "⚡ Running quick tests..."
	@go test -timeout 15s -short ./domain/... ./jobs/...

## test-domain: Run tests for a specific domain package
## Usage: make test-domain PKG=submission
test-domain:
	@if [ -z "$(PKG)" ]; then \
		echo "❌ Error: PKG variable is required. Usage: make test-domain PKG=submission"; \
		exit 1; \
	fi
	@echo "🧪 Running tests for domain/$(PKG)..."
	@go test -v -timeout 30s ./domain/$(PKG)/...

## clean: Clean test cache and artifacts
clean:
	@echo "🧹 Cleaning test cache and artifacts..."
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@rm -rf test_artifacts
	@echo "✅ Clean complete"

## test-watch: Run tests in watch mode (requires entr or similar tool)
test-watch:
	@echo "👀 Starting test watch mode..."
	@echo "Press Ctrl+C to stop"
	@find . -name '*.go' | entr -c make test-unit

## test-failed: Re-run only failed tests from last run
test-failed:
	@echo "🔁 Re-running failed tests..."
	@go test -v -timeout 30s ./... -run '^(TestRepository_Create_WithAll11Frameworks|TestRepository_Create_JSONBSerialization|TestRepository_Update_Success|TestRepository_Update_NotFound|TestRepository_UpdateWithTx_TransactionHandling|TestRepository_GetLatestVersionBySubmissionID_OrdersByVersionDESC|TestEnrichmentWorkflow_Integration)$$'

## tidy: Run go mod tidy
tidy:
	@echo "🧹 Running go mod tidy..."
	@go mod tidy
	@echo "✅ Dependencies tidied"
