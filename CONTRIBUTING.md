# Contributing to Olympian

Thank you for your interest in contributing to Olympian! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/ichtrojan/olympian.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test -v ./...`
6. Commit your changes: `git commit -m "Add your feature"`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- SQLite3 (for testing)
- Git

### Running Tests

```bash
go test -v ./...
```

### Building

```bash
go build ./...
```

### Running the CLI

```bash
go run cmd/olympian/main.go migrate --help
```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and concise

## Testing

- Write tests for all new features
- Ensure all tests pass before submitting PR
- Aim for high test coverage
- Test against all supported dialects (Postgres, MySQL, SQLite)

## Pull Request Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Include tests for new functionality
- Update documentation as needed
- Ensure all CI checks pass

## Adding New Features

When adding new features:

1. Check if an issue exists, or create one
2. Discuss the approach in the issue
3. Implement the feature
4. Add comprehensive tests
5. Update documentation
6. Submit a PR

## Reporting Bugs

When reporting bugs, please include:

- Go version
- Operating system
- Database system and version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Error messages and stack traces

## Feature Requests

Feature requests are welcome! Please:

- Search existing issues first
- Provide a clear use case
- Explain why the feature would be valuable
- Consider submitting a PR if you can implement it

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
