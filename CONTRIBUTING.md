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

Review and trust the repository's Mise configuration, then set up the development environment:

```bash
mise trust
mise run setup
```

The minimum supported Go version is declared in `go.mod`. Development tool versions are declared in `mise.toml` and resolve to the latest compatible patch releases. Compatibility tasks disable automatic Go toolchain upgrades so changes are still compiled and tested against the SDK's declared minimum release.

Mise provides the following development tasks:

| Task | Description |
|---|---|
| `mise run setup` | Install development tools, compatibility toolchains, and module dependencies. |
| `mise run build` | Build all packages with the minimum supported Go version. |
| `mise run test` | Test all packages with the minimum supported Go version. |
| `mise run test-dev` | Test all packages with the development Go version. |
| `mise run vet` | Run `go vet` with the minimum supported Go version. |
| `mise run format-check` | Check formatting and imports without changing files. |
| `mise run format` | Format Go files and organize imports with `goimports`. |
| `mise run lint-check` | Check the code with `golangci-lint`. |
| `mise run lint` | Apply fixes supported by `golangci-lint`. |
| `mise run ci` | Run all build, compatibility and development tests, vet, formatting, and lint checks. |

Run the complete local CI suite before opening a pull request:

```bash
mise run ci
```

Mise automatically activates and installs task-specific tools when running these commands, so shell activation is optional. If you want the configured `go`, `goimports`, and `golangci-lint` commands available directly in your shell, follow Mise's shell activation instructions.

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

Each provider should be self-contained and implement the shared capability interfaces defined by the root package (`Gateway`, `Disburser`, `WebhookHandler`, `BalanceChecker`).
Refer to [ADR 0001: Capability-Based Interface Architecture](docs/adr/0001-capability-based-architecture.md) for detailed rationale, capability composition patterns, and provider contracts.

## Coding Guidelines

- Format Go code and imports with `mise run format` (or `goimports`).
- Write clear, idiomatic Go code.
- Keep functions small and focused.
- Avoid unnecessary abstractions.
- Prefer composition over inheritance.
### Package Design & Naming
- **Shallow Nesting**: Avoid deep package hierarchies (e.g. avoid `internal/transport/http/client/v1`). Keep the package layout shallow, clean, and flat (maximum 1–2 levels deep).
- **Idiomatic Naming**: Use short, lowercase, single-word package names (`paykit`, `mpesa`, `airtelmoney`, `pesapal`, `gateway`). Never use underscores, dashes, or mixedCaps in package names.
- **No Stutter**: Avoid redundant names where the package name repeats the enclosing directory or type (e.g., avoid `transport.TransportClient` or `client.ClientConfig`).

### Cross-Package Exported Symbol Documentation
- **Purpose & Intent**: Every exported type, interface, function, method, and package-level constant must be documented with conventional Godoc comments.
- **Consumer-Centric Focus**: Godoc comments must clearly explain **when** to use the symbol and **how** to use it without exposing or coupling consumers to its inner workings. The intent of each exported symbol is to expose a specific public capability or piece of data.
- **Side Effects**: Document caller-facing side effects explicitly in Godoc comments (e.g. consuming or mutating an `io.Reader` such as `r.Body`, acquiring locks, modifying inputs, or mutating persistent state).
- **Implementation Comments**: Internal mechanics, algorithmic choices, and implementation details must be documented using regular internal comments (`// ...`) within function bodies or unexported symbols, never in public Godoc comments.

## Testing

Every logical contribution to the project must include comprehensive tests.

### Testing Hierarchy: Blackbox vs. Whitebox
- **Blackbox Tests (`package <pkg>_test`)**:
  - Focus on verifying business logic, public API contracts, and integration flows from the consumer's perspective.
  - Must remain immune to internal implementation changes.
  - Blackbox tests are long-lived; any failure indicates a breaking change in core behavior that must be explicitly accounted for and documented.
- **Whitebox Tests (`package <pkg>`)**:
  - Focus on testing internal implementation edge cases, nil/zero-value defenses, unexported helper functions, and specific error-mapping paths that the implementation may fall prey to.

Run all tests before submitting a pull request:

```bash
mise run test # or go test -v -race ./...
```

Fixtures for provider responses should be placed inside each provider's `fixtures/` directory. Fixtures must be deterministic, fictional, centralized, resettable, and compliant with API contracts (zero production data, zero PII, and zero secrets).

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
- `mise run ci` passes; it covers builds, tests, vet, formatting, and lint checks.
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
