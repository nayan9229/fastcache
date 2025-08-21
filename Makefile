.PHONY: help test benchmark coverage lint format clean build examples docs

# Default target
help: ## Display this help message
	@echo "FastCache - High-Performance In-Memory Cache"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Testing
test: ## Run all tests
	@echo "Running tests..."
	go test -v -race ./...

test-short: ## Run short tests (skip long-running tests)
	@echo "Running short tests..."
	go test -v -race -short ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmarking
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

benchmark-cpu: ## Run CPU profiling benchmark
	@echo "Running CPU profiling benchmark..."
	go test -bench=BenchmarkMixed -cpuprofile=cpu.prof -memprofile=mem.prof -run=^$$ ./...
	@echo "Profiles generated: cpu.prof, mem.prof"
	@echo "View CPU profile: go tool pprof cpu.prof"
	@echo "View memory profile: go tool pprof mem.prof"

benchmark-memory: ## Run memory usage benchmark
	@echo "Running memory benchmark..."
	go test -bench=BenchmarkMemoryUsage -benchmem -run=^$$ ./...

# Performance testing
load-test: ## Run load test (requires cache to be running)
	@echo "Running load test..."
	go run tools/load-tester/main.go

performance-test: ## Run comprehensive performance test
	@echo "Running performance tests..."
	go test -v -run=TestHighLoad -timeout=60s ./...

# Code quality
lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint >/dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

format: ## Format code
	@echo "Formatting code..."
	gofmt -w .
	@which goimports >/dev/null 2>&1 || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Build and examples
build: ## Build all examples
	@echo "Building examples..."
	go build -o bin/basic-example examples/basic/main.go
	go build -o bin/api-server examples/api-server/main.go
	go build -o bin/high-concurrency examples/high-concurrency/main.go
	go build -o bin/monitoring examples/monitoring/main.go
	go build -o bin/load-tester tools/load-tester/main.go
	@echo "Binaries built in bin/ directory"

run-basic: ## Run basic example
	@echo "Running basic example..."
	go run examples/basic/main.go

run-api-server: ## Run API server example
	@echo "Running API server example..."
	go run examples/api-server/main.go

run-monitoring: ## Run monitoring example
	@echo "Running monitoring example..."
	go run examples/monitoring/main.go

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@which godoc >/dev/null 2>&1 || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Starting godoc server at http://localhost:6060"
	@echo "Visit: http://localhost:6060/pkg/github.com/nayan9229/fastcache/"
	godoc -http=:6060

# Module management
mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	go mod tidy

mod-verify: ## Verify go modules
	@echo "Verifying modules..."
	go mod verify

# Release
tag: ## Create a new git tag (usage: make tag VERSION=v1.0.0)
	@echo "Creating tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf cpu.prof mem.prof
	rm -rf *.test

# Development
dev-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/godoc@latest

# CI/CD helpers
ci-test: ## Run CI tests
	@echo "Running CI tests..."
	go test -v -race -coverprofile=coverage.out ./...

ci-lint: ## Run CI linting
	@echo "Running CI linting..."
	golangci-lint run --timeout=5m

ci-build: ## Run CI build
	@echo "Running CI build..."
	go build ./...

# Quick development cycle
quick: format vet test-short ## Quick development cycle (format, vet, short tests)

full: format vet lint test benchmark ## Full development cycle (format, vet, lint, test, benchmark)

# Statistics
stats: ## Show project statistics
	@echo "Project Statistics:"
	@echo "=================="
	@echo "Lines of code:"
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
	@echo ""
	@echo "Test files:"
	@find . -name "*_test.go" -not -path "./vendor/*" | wc -l
	@echo ""
	@echo "Example files:"
	@find examples -name "*.go" | wc -l
	@echo ""
	@echo "Go modules:"
	@go list -m all | wc -l