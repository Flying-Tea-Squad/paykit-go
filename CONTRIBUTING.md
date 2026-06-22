# Contributing to PAYKIT-GO

First off, thank you for considering contributing to PAYKIT-GO! 🎉

This project aims to provide a unified Go library for integrating Kenyan payment providers such as M-Pesa, Airtel Money, Pesapal, Flutterwave, and others.

We welcome contributions of all kinds, including:

- Bug fixes
- New payment provider integrations
- Documentation improvements
- Tests
- Performance improvements
- Feature requests

## Getting Started

### Fork the Repository

Fork the repository and clone your fork:

```bash
git clone https://github.com/<your-username>/paykit-go.git
cd paykit-go
```

### Create a Branch

Create a new branch for your changes:

```bash
git checkout -b feature/my-feature
```

Examples:

```bash
git checkout -b feature/airtel-money
git checkout -b fix/stk-push-timeout
git checkout -b docs/update-readme
```

## Development Setup

Install Go:

- Go 1.25 or newer is recommended.

Verify installation:

```bash
go version
```

Download dependencies:

```bash
go mod tidy
```

Run tests:

```bash
go test ./...
```

## Coding Standards

Please follow these guidelines:

### Formatting

Format code before committing:

```bash
go fmt ./...
```

### Static Analysis

Run:

```bash
go vet ./...
```

### Naming

- Use clear and descriptive names.
- Exported functions must include comments.
- Keep APIs simple and idiomatic.

Example:

```go
// STKPush initiates an M-Pesa STK push request.
func (c *Client) STKPush(req STKRequest) error {
    // ...
}
```

## Adding a New Payment Provider

Payment providers should live under:

```text
providers/
├── mpesa/
├── airtelmoney/
├── pesapal/
└── flutterwave/
```

A provider implementation should include:

- Authentication
- Payment requests
- Transaction status checks
- Error handling
- Tests
- Documentation

## Testing

All new functionality should include tests.

Run all tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

## Documentation

If you add a new feature:

- Update README.md
- Add usage examples
- Document exported functions

Good documentation helps adoption.

## Commit Messages

Use descriptive commit messages.

Examples:

```text
feat: add mpesa stk push support
fix: handle oauth token expiration
docs: update installation instructions
test: add stk push unit tests
```

## Pull Requests

Before opening a pull request:

- Ensure tests pass.
- Ensure code is formatted.
- Update documentation if necessary.
- Keep pull requests focused on a single change.

When creating a PR:

1. Describe the change.
2. Explain why it is needed.
3. Include screenshots or logs if applicable.
4. Reference related issues.

## Reporting Bugs

Please open an issue and include:

- Go version
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior

## Feature Requests

Feature requests are welcome.

When opening a feature request, include:

- Use case
- Proposed API
- Expected behavior

## Code of Conduct

Be respectful and constructive.

We are committed to providing a welcoming environment for everyone regardless of experience level.

## Questions

If you have questions, open a discussion or issue and we will be happy to help.

Thank you for contributing to PAYKIT-GO! 🚀
