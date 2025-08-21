#!/bin/bash

# FastCache Test Script
# Runs comprehensive tests with coverage reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_status "Using Go version: $GO_VERSION"
}

# Clean previous test artifacts
clean_artifacts() {
    print_status "Cleaning previous test artifacts..."
    rm -f coverage.out coverage.html
    rm -f cpu.prof mem.prof
    rm -f *.test
}

# Run go mod tidy
tidy_modules() {
    print_status "Tidying Go modules..."
    go mod tidy
    go mod verify
}

# Run go vet
run_vet() {
    print_status "Running go vet..."
    if go vet ./...; then
        print_success "go vet passed"
    else
        print_error "go vet failed"
        exit 1
    fi
}

# Run tests with race detection
run_tests() {
    print_status "Running tests with race detection..."
    
    if go test -v -race -timeout=30s ./...; then
        print_success "All tests passed"
    else
        print_error "Some tests failed"
        exit 1
    fi
}

# Run tests with coverage
run_coverage_tests() {
    print_status "Running tests with coverage..."
    
    if go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; then
        print_success "Coverage tests completed"
        
        # Generate HTML coverage report
        if command -v go &> /dev/null; then
            go tool cover -html=coverage.out -o coverage.html
            print_success "Coverage report generated: coverage.html"
            
            # Show coverage percentage
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
            print_status "Total coverage: $COVERAGE"
            
            # Check if coverage is acceptable
            COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
            if (( $(echo "$COVERAGE_NUM >= 80" | bc -l) )); then
                print_success "Coverage is acceptable (≥80%)"
            else
                print_warning "Coverage is below 80%: $COVERAGE"
            fi
        fi
    else
        print_error "Coverage tests failed"
        exit 1
    fi
}

# Run benchmarks
run_benchmarks() {
    print_status "Running benchmarks..."
    
    if go test -bench=. -benchmem -run=^$ ./...; then
        print_success "Benchmarks completed"
    else
        print_warning "Some benchmarks failed"
    fi
}

# Run short tests (for quick validation)
run_short_tests() {
    print_status "Running short tests..."
    
    if go test -v -race -short -timeout=10s ./...; then
        print_success "Short tests passed"
    else
        print_error "Short tests failed"
        exit 1
    fi
}

# Run load tests
run_load_tests() {
    print_status "Running load tests..."
    
    if go test -v -run=TestHighLoad -timeout=60s ./...; then
        print_success "Load tests completed"
    else
        print_warning "Load tests failed or timed out"
    fi
}

# Check test files
check_test_files() {
    print_status "Checking test coverage..."
    
    # Find Go files without corresponding test files
    for gofile in $(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -not -path "./examples/*" -not -path "./tools/*"); do
        testfile="${gofile%%.go}_test.go"
        if [[ ! -f "$testfile" ]]; then
            print_warning "Missing test file for: $gofile"
        fi
    done
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help       Show this help message"
    echo "  -s, --short      Run only short tests"
    echo "  -c, --coverage   Run tests with coverage"
    echo "  -b, --bench      Run benchmarks"
    echo "  -l, --load       Run load tests"
    echo "  -a, --all        Run all tests (default)"
    echo "  --no-vet         Skip go vet"
    echo "  --no-clean       Don't clean artifacts before running"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all tests"
    echo "  $0 --short           # Quick test run"
    echo "  $0 --coverage        # Run with coverage"
    echo "  $0 --bench           # Run benchmarks only"
}

# Main function
main() {
    local run_short=false
    local run_coverage=false
    local run_bench=false
    local run_load=false
    local run_all=true
    local skip_vet=false
    local skip_clean=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -s|--short)
                run_short=true
                run_all=false
                shift
                ;;
            -c|--coverage)
                run_coverage=true
                run_all=false
                shift
                ;;
            -b|--bench)
                run_bench=true
                run_all=false
                shift
                ;;
            -l|--load)
                run_load=true
                run_all=false
                shift
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            --no-vet)
                skip_vet=true
                shift
                ;;
            --no-clean)
                skip_clean=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    echo "🧪 FastCache Test Runner"
    echo "========================"
    
    # Initial checks
    check_go
    
    if [[ "$skip_clean" != true ]]; then
        clean_artifacts
    fi
    
    tidy_modules
    
    if [[ "$skip_vet" != true ]]; then
        run_vet
    fi
    
    # Run tests based on options
    if [[ "$run_short" == true ]]; then
        run_short_tests
    elif [[ "$run_coverage" == true ]]; then
        run_coverage_tests
    elif [[ "$run_bench" == true ]]; then
        run_benchmarks
    elif [[ "$run_load" == true ]]; then
        run_load_tests
    elif [[ "$run_all" == true ]]; then
        run_tests
        run_coverage_tests
        run_benchmarks
        check_test_files
        print_status "Running load tests (this may take a while)..."
        run_load_tests
    fi
    
    print_success "Test execution completed!"
    
    # Final summary
    echo ""
    echo "📊 Test Summary"
    echo "==============="
    
    if [[ -f "coverage.out" ]]; then
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        echo "Coverage: $COVERAGE"
    fi
    
    if [[ -f "coverage.html" ]]; then
        echo "Coverage report: coverage.html"
    fi
    
    echo "Artifacts in current directory:"
    ls -la *.out *.html *.prof 2>/dev/null || echo "No test artifacts found"
}

# Run main function with all arguments
main "$@"