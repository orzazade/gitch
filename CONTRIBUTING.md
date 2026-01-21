# Contributing to gitch

Thank you for your interest in contributing to gitch! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check existing [issues](https://github.com/orzazade/gitch/issues) to avoid duplicates
2. Use the latest version of gitch
3. Collect relevant information (OS, Go version, steps to reproduce)

When creating a bug report:
- Use the bug report template
- Provide clear steps to reproduce
- Include expected vs actual behavior
- Add relevant logs or screenshots

### Suggesting Features

We welcome feature suggestions! Please:
1. Check if the feature is already planned in our [roadmap](.planning/ROADMAP.md)
2. Use the feature request template
3. Explain the use case and why it would benefit users

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Write tests** for new functionality
4. **Run the test suite** to ensure nothing is broken
5. **Update documentation** if needed
6. **Submit a pull request** using our template

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/gitch.git
cd gitch

# Add upstream remote
git remote add upstream https://github.com/orzazade/gitch.git

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

### Project Structure

```
gitch/
├── cmd/           # Cobra commands
├── internal/      # Internal packages
│   ├── config/    # Configuration management
│   ├── git/       # Git operations
│   ├── ssh/       # SSH key management
│   └── ui/        # TUI components
├── main.go        # Entry point
└── .planning/     # Project planning docs (local only)
```

### Coding Standards

- Follow standard Go conventions and `gofmt`
- Use meaningful variable and function names
- Write comments for exported functions
- Keep functions focused and small
- Handle errors explicitly

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific package tests
go test ./internal/config/...
```

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Adding tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

Examples:
```
feat(ssh): add support for ed25519 keys
fix(config): handle missing config file gracefully
docs(readme): update installation instructions
```

### Branch Naming

- `feat/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

## Review Process

1. All PRs require at least one review
2. CI must pass (tests, linting)
3. Changes should be focused and atomic
4. Large changes should be discussed in an issue first

## Getting Help

- Read the [documentation](README.md)
- Check existing [issues](https://github.com/orzazade/gitch/issues)
- Ask in [GitHub Discussions](https://github.com/orzazade/gitch/discussions)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
