# Contributing to FastCache

Thank you for your interest in contributing to FastCache! This guide will help you get started with contributing to this high-performance in-memory cache library.

## 🚀 Quick Start

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/fastcache.git
   cd fastcache
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/amazing-feature
   ```
4. **Make your changes** and add tests
5. **Run the test suite** to ensure everything works
6. **Submit a pull request**

## 📋 Development Setup

### Prerequisites

- **Go 1.21+** - [Download Go](https://golang.org/dl/)
- **Git** - [Install Git](https://git-scm.com/downloads)
- **Make** (optional) - For convenient development commands

### Environment Setup

```bash
# Clone the repository
git clone https://github.com/nayan9229/fastcache.git
cd fastcache

# Install development dependencies
make dev-deps

# Verify setup
make test
```

### Development Tools

We recommend installing these tools for the best development experience:

```bash
# Linting and formatting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Documentation
go install golang.org/x/tools/cmd/godoc@latest
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run short tests (for quick feedback)
make test-short

# Run benchmarks
make benchmark

# Run load tests
make load-test
```

### Test Structure

- **Unit tests**: Test individual functions and methods
- **Integration tests**: Test component interactions
- **Benchmark tests**: Performance and memory usage tests
- **Load tests**: High concurrency and stress tests

### Writing Tests

- All new features must include tests
- Aim for **80%+ code coverage**
- Use **table-driven tests** where appropriate
- Include **benchmarks** for performance-critical code
- Test **error conditions** and edge cases

Example test structure:

```go
func TestCacheOperation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "test", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## 📝 Code Standards

### Code Style

- Follow **Go standard formatting** (`gofmt`)
- Use **clear, descriptive names**
- Write **self-documenting code**
- Add **comments for public APIs**
- Follow **Go best practices**

### Linting

Run the linter before submitting:

```bash
make lint
```

### Documentation

- **All public functions** must have Go doc comments
- **Examples** should be provided for complex APIs
- **Update README.md** if adding new features
- **Include usage examples** in appropriate directories

## 🔄 Pull Request Process

### Before Submitting

1. **Run the full test suite**:
   ```bash
   make full
   ```

2. **Verify your changes** don't break existing functionality

3. **Update documentation** if necessary

4. **Add tests** for new features

5. **Update CHANGELOG** if applicable

### Pull Request Guidelines

- **Use descriptive titles** and descriptions
- **Reference related issues** using `Fixes #123` or `Closes #123`
- **Keep changes focused** - one feature per PR
- **Include tests** and documentation updates
- **Ensure CI passes** before requesting review

### PR Template

When submitting a PR, please include:

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Benchmarks run (if applicable)

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Changes generate no new warnings
```

## 🐛 Reporting Issues

### Bug Reports

When reporting bugs, please include:

- **Go version** (`go version`)
- **Operating system** and architecture
- **Minimal code example** that reproduces the issue
- **Expected vs actual behavior**
- **Stack trace** if applicable

Use this template:

```markdown
**Environment:**
- Go version: 
- OS: 
- Architecture: 

**Description:**
Clear description of the bug

**Steps to Reproduce:**
1. 
2. 
3. 

**Expected Behavior:**
What should happen

**Actual Behavior:**
What actually happens

**Code Example:**
```go
// Minimal example
```

**Additional Context:**
Any other relevant information
```

### Feature Requests

For new features, please:

1. **Check existing issues** first
2. **Describe the use case** clearly
3. **Explain the expected API** (if applicable)
4. **Consider backwards compatibility**
5. **Discuss performance implications**

## 🏗️ Architecture Guidelines

### Design Principles

1. **Performance First**: All changes should maintain or improve performance
2. **Thread Safety**: All public APIs must be goroutine-safe
3. **Memory Efficiency**: Minimize memory allocation and GC pressure
4. **Backwards Compatibility**: Avoid breaking changes when possible
5. **Simple APIs**: Keep interfaces clean and intuitive

### Code Organization

```
fastcache/
├── cache.go          # Main cache implementation
├── config.go         # Configuration structures
├── stats.go          # Statistics and monitoring
├── errors.go         # Error types and handling
├── *_test.go         # Test files
├── examples/         # Usage examples
├── tools/            # Development tools
└── docs/             # Documentation
```

### Performance Considerations

- **Minimize allocations** in hot paths
- **Use sync.Pool** for frequently allocated objects
- **Prefer atomic operations** over mutexes when possible
- **Profile performance** changes with benchmarks
- **Consider memory usage** impact

## 🚀 Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

### Release Checklist

For maintainers preparing releases:

1. **Update version** in relevant files
2. **Update CHANGELOG.md**
3. **Run full test suite**
4. **Create git tag**: `git tag v1.2.3`
5. **Push tag**: `git push origin v1.2.3`
6. **GitHub Actions** will handle the rest

## 🎯 Areas for Contribution

We welcome contributions in these areas:

### High Priority
- **Performance optimizations**
- **Memory usage improvements**
- **Additional eviction policies**
- **Metrics and monitoring enhancements**
- **Documentation improvements**

### Medium Priority
- **Additional configuration options**
- **More comprehensive examples**
- **Integration guides**
- **Docker improvements**
- **Kubernetes examples**

### Low Priority
- **Additional utility functions**
- **Code cleanup and refactoring**
- **Test coverage improvements**

## 💬 Communication

### Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Code Review**: Submit PRs for collaborative review

### Code of Conduct

Please be respectful and constructive in all interactions. We want FastCache to be a welcoming project for contributors of all backgrounds and experience levels.

## 🏆 Recognition

Contributors are recognized in several ways:

- **CONTRIBUTORS.md**: Listed in the contributors file
- **Release Notes**: Mentioned in release changelogs
- **GitHub**: Automatic contribution tracking
- **Documentation**: Examples and guides attribution

## 📚 Additional Resources

### Go Resources
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Performance Tips](https://github.com/golang/go/wiki/Performance)

### Project Resources
- [Architecture Documentation](docs/architecture.md)
- [Performance Guide](docs/performance.md)
- [API Documentation](https://godoc.org/github.com/nayan9229/fastcache)

### Tools and Libraries
- [golangci-lint](https://golangci-lint.run/) - Linting
- [benchstat](https://godoc.org/golang.org/x/perf/cmd/benchstat) - Benchmark analysis
- [pprof](https://golang.org/pkg/net/http/pprof/) - Profiling

---

Thank you for contributing to FastCache! Your efforts help make this project better for everyone. 🙏
