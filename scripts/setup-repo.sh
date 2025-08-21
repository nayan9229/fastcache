#!/bin/bash

# FastCache Repository Setup Script
# This script sets up the complete repository structure

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

# Check if we're in the right directory
check_directory() {
    if [[ ! -f "go.mod" ]] || [[ ! -f "cache.go" ]]; then
        print_error "This script should be run from the fastcache repository root"
        print_status "Expected files: go.mod, cache.go"
        print_status "Current directory: $(pwd)"
        exit 1
    fi
}

# Create directory structure
create_directories() {
    print_status "Creating directory structure..."
    
    # Create main directories
    mkdir -p examples/{basic,api-server,high-concurrency,monitoring,docker}
    mkdir -p tools/load-tester
    mkdir -p scripts
    mkdir -p docs
    mkdir -p .github/workflows
    mkdir -p bin
    
    print_success "Directory structure created"
}

# Setup git configuration
setup_git() {
    print_status "Setting up git configuration..."
    
    # Initialize git if not already done
    if [[ ! -d ".git" ]]; then
        git init
        print_status "Git repository initialized"
    fi
    
    # Add .gitignore if it doesn't exist
    if [[ ! -f ".gitignore" ]]; then
        print_warning ".gitignore not found, creating a basic one..."
        cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test

# Output files
*.out
*.prof
coverage.html

# Build artifacts
bin/
dist/

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Logs
*.log

# Environment files
.env
.env.local
EOF
    fi
    
    print_success "Git configuration setup complete"
}

# Setup Go module
setup_go_module() {
    print_status "Setting up Go module..."
    
    # Check if go.mod exists
    if [[ ! -f "go.mod" ]]; then
        print_status "Initializing Go module..."
        go mod init github.com/nayan9229/fastcache
    fi
    
    # Tidy modules
    go mod tidy
    
    print_success "Go module setup complete"
}

# Build examples
build_examples() {
    print_status "Building examples..."
    
    # Build basic example
    if [[ -f "examples/basic/main.go" ]]; then
        go build -o bin/basic-example examples/basic/main.go
        print_success "Built basic example"
    fi
    
    # Build API server example
    if [[ -f "examples/api-server/main.go" ]]; then
        go build -o bin/api-server examples/api-server/main.go
        print_success "Built API server example"
    fi
    
    # Build high concurrency example
    if [[ -f "examples/high-concurrency/main.go" ]]; then
        go build -o bin/high-concurrency examples/high-concurrency/main.go
        print_success "Built high concurrency example"
    fi
    
    # Build monitoring example
    if [[ -f "examples/monitoring/main.go" ]]; then
        go build -o bin/monitoring examples/monitoring/main.go
        print_success "Built monitoring example"
    fi
    
    # Build load tester
    if [[ -f "tools/load-tester/main.go" ]]; then
        go build -o bin/load-tester tools/load-tester/main.go
        print_success "Built load tester"
    fi
}

# Run tests
run_tests() {
    print_status "Running tests..."
    
    if go test -v -race ./...; then
        print_success "All tests passed"
    else
        print_warning "Some tests failed, but continuing setup..."
    fi
}

# Setup development tools
setup_dev_tools() {
    print_status "Setting up development tools..."
    
    # Install golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        print_status "Installing golangci-lint..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    
    # Install goimports
    if ! command -v goimports &> /dev/null; then
        print_status "Installing goimports..."
        go install golang.org/x/tools/cmd/goimports@latest
    fi
    
    # Install godoc
    if ! command -v godoc &> /dev/null; then
        print_status "Installing godoc..."
        go install golang.org/x/tools/cmd/godoc@latest
    fi
    
    print_success "Development tools installed"
}

# Make scripts executable
setup_scripts() {
    print_status "Setting up scripts..."
    
    # Make all shell scripts executable
    find scripts -name "*.sh" -type f -exec chmod +x {} \; 2>/dev/null || true
    
    print_success "Scripts configured"
}

# Create example configuration files
create_example_configs() {
    print_status "Creating example configuration files..."
    
    # Create Docker config if docker directory exists
    if [[ -d "examples/docker" ]] && [[ ! -f "examples/docker/config.json" ]]; then
        cat > examples/docker/config.json << 'EOF'
{
  "cache": {
    "max_memory_mb": 256,
    "shard_count": 512,
    "default_ttl_minutes": 15,
    "cleanup_interval_seconds": 120
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout_seconds": 30,
    "write_timeout_seconds": 30
  }
}
EOF
        print_success "Created Docker configuration"
    fi
}

# Verify setup
verify_setup() {
    print_status "Verifying setup..."
    
    local errors=0
    
    # Check required files
    required_files=(
        "go.mod"
        "cache.go"
        "config.go"
        "stats.go"
        "errors.go"
        "cache_test.go"
        "README.md"
        "LICENSE"
        "Makefile"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            print_error "Missing required file: $file"
            errors=$((errors + 1))
        fi
    done
    
    # Check if Go module is valid
    if ! go mod verify &>/dev/null; then
        print_error "Go module verification failed"
        errors=$((errors + 1))
    fi
    
    # Check if basic build works
    if ! go build ./... &>/dev/null; then
        print_error "Build failed"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Setup verification passed"
        return 0
    else
        print_error "Setup verification failed with $errors errors"
        return 1
    fi
}

# Show next steps
show_next_steps() {
    echo ""
    echo "🎉 FastCache Repository Setup Complete!"
    echo "======================================"
    echo ""
    echo "📁 Repository structure:"
    echo "  ├── cache.go              # Main cache implementation"
    echo "  ├── config.go             # Configuration"
    echo "  ├── stats.go              # Statistics and monitoring"
    echo "  ├── errors.go             # Error types"
    echo "  ├── *_test.go             # Test files"
    echo "  ├── examples/              # Usage examples"
    echo "  ├── tools/                 # Development tools"
    echo "  ├── scripts/               # Build and test scripts"
    echo "  └── docs/                  # Documentation"
    echo ""
    echo "🚀 Quick start:"
    echo "  make test                  # Run tests"
    echo "  make benchmark             # Run benchmarks"
    echo "  make run-basic             # Run basic example"
    echo "  make run-api-server        # Run API server example"
    echo ""
    echo "📖 Development:"
    echo "  make help                  # Show all available commands"
    echo "  make dev-deps              # Install development dependencies"
    echo "  make format                # Format code"
    echo "  make lint                  # Run linter"
    echo ""
    echo "🐳 Docker:"
    echo "  cd examples/docker"
    echo "  docker-compose up          # Run with Docker Compose"
    echo ""
    echo "📚 Documentation:"
    echo "  make docs                  # Start documentation server"
    echo "  open http://localhost:6060 # View documentation"
    echo ""
    echo "🔧 Next steps:"
    echo "  1. Run 'make test' to verify everything works"
    echo "  2. Try the examples in the examples/ directory"
    echo "  3. Read the documentation in docs/"
    echo "  4. Check out CONTRIBUTING.md to contribute"
    echo ""
    
    if [[ -d ".git" ]]; then
        echo "📝 Git setup:"
        echo "  git add .                  # Stage all files"
        echo "  git commit -m 'Initial commit'"
        echo "  git branch -M main"
        echo "  git remote add origin https://github.com/nayan9229/fastcache.git"
        echo "  git push -u origin main"
        echo ""
    fi
}

# Print usage
usage() {
    echo "FastCache Repository Setup Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  --skip-build         Skip building examples"
    echo "  --skip-test          Skip running tests"
    echo "  --skip-tools         Skip installing development tools"
    echo "  --verify-only        Only run verification"
    echo ""
    echo "This script will:"
    echo "  1. Create directory structure"
    echo "  2. Setup Git configuration"
    echo "  3. Initialize Go module"
    echo "  4. Build examples"
    echo "  5. Run tests"
    echo "  6. Install development tools"
    echo "  7. Verify setup"
    echo ""
}

# Main function
main() {
    local skip_build=false
    local skip_test=false
    local skip_tools=false
    local verify_only=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            --skip-build)
                skip_build=true
                shift
                ;;
            --skip-test)
                skip_test=true
                shift
                ;;
            --skip-tools)
                skip_tools=true
                shift
                ;;
            --verify-only)
                verify_only=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    echo "🚀 FastCache Repository Setup"
    echo "============================"
    echo ""
    
    # Always check directory first
    check_directory
    
    if [[ "$verify_only" == true ]]; then
        verify_setup
        exit $?
    fi
    
    # Run setup steps
    create_directories
    setup_git
    setup_go_module
    
    if [[ "$skip_build" != true ]]; then
        build_examples
    fi
    
    if [[ "$skip_test" != true ]]; then
        run_tests
    fi
    
    if [[ "$skip_tools" != true ]]; then
        setup_dev_tools
    fi
    
    setup_scripts
    create_example_configs
    
    # Final verification
    if verify_setup; then
        show_next_steps
    else
        print_error "Setup completed with errors. Please check the output above."
        exit 1
    fi
}

# Run main function with all arguments
main "$@"