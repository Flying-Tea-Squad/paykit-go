# Contributing to paykit-go

First off, thank you for considering contributing to **paykit-go**! 🎉

Our goal is to build a production-ready Go SDK that provides a unified interface for integrating African payment providers such as M-Pesa, Airtel Money, and Pesapal.
Every contribution—whether it's code, documentation, bug reports, or feature suggestions—is appreciated.

## Getting Started

### 1. Fork the Repository

Fork the repository to your GitHub account and clone it locally.

```bash
git clone https://github.com/<your-username>/paykit-go.git
cd paykit-go
```

Add the upstream repository:

```bash
git remote add upstream https://github.com/Flying-Tea-Squad/paykit-go.git
```

### 2. Create a Branch

Create a new branch from `main`.

```bash
git checkout -b feat/my-feature
```

Branch naming examples:

- `feat/stk-push`
- `fix/token-cache`
- `docs/update-readme`
- `test/mpesa-client`

## Development Setup

Ensure you have [Mise](https://mise.jdx.dev/getting-started.html) and Git installed. You do not need to install Go or the project's development tools separately.

Review and trust the repository's Mise configuration, then install the toolchain:

```bash
mise trust
mise install
```

The SDK supports Go 1.22.2 and later. Mise installs the latest Go 1.25 patch for development because the pinned versions of `goimports` and `golangci-lint` require Go 1.25 to build. Compatibility tasks automatically use the latest Go 1.22 patch and disable automatic toolchain upgrades, so changes are still compiled and tested against the SDK's minimum supported Go release.

Mise provides the following development tasks:

| Task | Description |
|---|---|
| `mise run build` | Build all packages with Go 1.22. |
| `mise run test` | Test all packages with Go 1.22. |
| `mise run vet` | Run `go vet` with Go 1.22. |
| `mise run format-check` | Check formatting and imports without changing files. |
| `mise run format` | Format Go files and organize imports with `goimports`. |
| `mise run lint-check` | Check the code with `golangci-lint`. |
| `mise run lint` | Apply fixes supported by `golangci-lint`. |
| `mise run ci` | Run all build, test, vet, formatting, and lint checks. |

Run the complete local CI suite before opening a pull request:

```bash
mise run ci
```

Mise automatically activates and installs task-specific tools when running these commands, so shell activation is optional. If you want the configured `go`, `goimports`, and `golangci-lint` commands available directly in your shell, follow Mise's shell activation instructions.

When changing module dependencies, tidy the module explicitly:

```bash
go mod tidy
```

## Project Structure

```terminal
paykit-go/
├── gateway/          # Provider registry
├── bogus/            # Test gateway implementation
├── mpesa/            # M-Pesa provider
├── airtelmoney/      # Airtel Money provider
├── pesapal/          # Pesapal provider
├── callback/         # Shared callback utilities
├── driver/           # Import all providers
└── docs/             # Project documentation
```

Each provider should be self-contained and implement the shared gateway interfaces defined by the root package.

## Coding Guidelines

- Format Go code and imports with `mise run format`.
- Write clear, idiomatic Go code.
- Keep functions small and focused.
- Avoid unnecessary abstractions.
- Prefer composition over inheritance.
- Add comments for exported types and functions.

## Testing

Every new feature or bug fix should include tests whenever practical.

Run all tests before submitting a pull request:

```bash
mise run test
```

Fixtures for provider responses should be placed inside each provider's `fixtures/` directory.

## Commit Messages

This project follows the Conventional Commits specification.

Examples:

```terminal
feat(mpesa): add OAuth token caching
fix(callback): validate callback signature
docs: update contributing guide
test(mpesa): add stk push fixtures
refactor(http): simplify retry logic
chore: initialize project structure
```

## Pull Requests

Before opening a Pull Request, make sure:

- Your branch is up to date with `main`.
- `mise run ci` passes.
- Your code is formatted with `goimports`.
- The code builds and tests pass with the minimum supported Go version.
- `go vet` and `golangci-lint` report no issues.
- New functionality includes tests where appropriate.
- Documentation has been updated if necessary.

Your Pull Request should include:

- A short description of the change.
- The motivation behind it.
- Screenshots or logs if applicable.
- A reference to the related issue (e.g. `Closes #12`).

## Reporting Issues

When opening an issue, please include:

- A clear description of the problem.
- Steps to reproduce.
- Expected behavior.
- Actual behavior.
- Go version.
- Operating system.

## Adding a New Payment Provider

Each provider should:

- Live in its own package.
- Implement the shared gateway interface.
- Handle provider-specific authentication internally.
- Normalize responses into the common `Response` type.
- Include fixtures and tests.
- Document any provider-specific configuration.

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all experience levels and strive to maintain a friendly, collaborative environment.

Happy coding! 🚀
