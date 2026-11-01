# Contributing to paykit-go

First off, thank you for considering contributing to **paykit-go**! 🎉

Our goal is to build a production-ready, capability-based Go SDK providing a unified interface for integrating African payment providers such as M-Pesa, Airtel Money, and Pesapal.
Every contribution—whether it's code, documentation, bug reports, or feature suggestions—is deeply appreciated.

This guide covers the shared development toolchain, local development and verification workflows, coding and architectural standards, and pull request guidelines.

---

## Getting Started

### 1. Fork the Repository

Fork the repository to your GitHub account and clone it locally:

```bash
git clone https://github.com/<your-username>/paykit-go.git
cd paykit-go
```

Add the upstream repository:

```bash
git remote add upstream https://github.com/Flying-Tea-Squad/paykit-go.git
```

### 2. Create a Branch

Create a new branch from `main`:

```bash
git checkout -b feat/my-feature
```

Branch naming examples:

- `feat/stk-push`
- `fix/token-cache`
- `docs/update-contributing`
- `test/mpesa-client`
- `chore/upgrade-deps`

---

## Development Setup & Toolchain

### Mise Toolchain

Ensure you have [Mise](https://mise.jdx.dev/getting-started.html) and Git installed. You do not need to install Go or the project's development tools separately.

Review and trust the repository's Mise configuration, then set up the development environment:

```bash
mise trust
mise run setup
```

The minimum supported Go version is declared in `go.mod` (Go 1.22). Development tool versions are declared in `mise.toml` (Go 1.25 dev toolchain, `golangci-lint`, and `goimports`). Compatibility tasks disable automatic Go toolchain upgrades so changes are reliably compiled and tested against the SDK's declared minimum release.

### Mise Tasks

The root [`mise.toml`](mise.toml) defines the project's development commands. The complete, live list of available tasks, flags, and descriptions is always discoverable directly from the command line:

```bash
mise tasks
```

The table below highlights the primary tasks essential for setup and the core contribution lifecycle:

| Task                             | Purpose                                                                       |
| -------------------------------- | ----------------------------------------------------------------------------- |
| `mise run setup`                 | Install development tools, compatibility toolchains, and module dependencies. |
| `mise run build`                 | Build all packages with the minimum supported Go version.                     |
| `mise run check`                 | Perform all static checks (Go vet/format/lint, shell, markdown) across files. |
| `mise run test`                  | Test all packages with the minimum supported Go version.                      |
| `mise run test-dev`              | Test all packages with the development Go version.                            |
| `mise run ci`                    | Run all continuous integration checks (`build`, `check`, `test`, `test-dev`). |
| `mise run lint` (alias `fix`)    | Run all formatters and autofix linters across Go, shell, and markdown.        |
| `mise run pre-push`              | Run formatters/linters (`mise run lint`), then execute `mise run ci`.         |
| `mise run vet`                   | Vet all packages with the minimum supported Go version.                       |
| `mise run format-check`          | Check Go formatting and imports without modifying files.                      |
| `mise run format`                | Format Go source files and organize imports with `goimports`.                 |
| `mise run lint-check`            | Check Go code with `golangci-lint`.                                           |
| `mise run sh:format-check`       | Check shell scripts formatting with `shfmt`.                                  |
| `mise run sh:format`             | Format shell scripts with `shfmt`.                                            |
| `mise run sh:lint-check`         | Lint shell scripts with `shellcheck`.                                         |
| `mise run markdown:format-check` | Check markdown formatting with `prettier`.                                    |
| `mise run markdown:format`       | Format markdown documentation with `prettier`.                                |

Mise automatically activates and installs task-specific tools when running these commands. If you want `go`, `goimports`, and `golangci-lint` available directly in your shell, follow Mise's shell activation instructions.

### Linting and Formatting Tools

| Scope                     | Tool            | Command (Check)                  | Command (Format/Fix)       |
| ------------------------- | --------------- | -------------------------------- | -------------------------- |
| Go Formatting & Imports   | `goimports`     | `mise run format-check`          | `mise run format`          |
| Go Vetting                | `go vet`        | `mise run vet`                   | N/A                        |
| Go Static Analysis / Lint | `golangci-lint` | `mise run lint-check`            | `golangci-lint run --fix`  |
| Shell Script Linting      | `shellcheck`    | `mise run sh:lint-check`         | N/A                        |
| Shell Script Formatting   | `shfmt`         | `mise run sh:format-check`       | `mise run sh:format`       |
| Markdown Documentation    | `prettier`      | `mise run markdown:format-check` | `mise run markdown:format` |
| All Formatters & Linters  | Multi-tool      | `mise run check`                 | `mise run lint` (or `fix`) |

---

## Local Verification & Pre-Push Checks

Contributors must test and verify changes locally before pushing, whether GitHub Actions is currently available or not. Relying solely on hosted CI is not permitted.

### The Standard Pre-Push Workflow

Run the automated pre-push convenience task before pushing your branch:

```bash
mise run pre-push
```

This task automatically formats all Go, shell, and markdown files, applies supported linter fixes, and then executes the complete `mise run ci` suite (`build`, `check`, and `test`).

### Targeted Checks During Development

- **All Static Checks (`mise run check`)**: Runs all static tests across Go, shell scripts, and markdown documentation without running logical tests or modifying files.
- **Logical Tests (`mise run test` or `mise run test-dev`)**: Runs the Go test suite under the minimum supported Go version (1.22) or development version (1.25). To run with the race detector: `go test -v -race ./...`.
- **Formatting and Linting (`mise run lint` or `mise run fix`)**: Formats all files and applies autofixes across Go, shell scripts, and markdown.
- **Full CI Suite (`mise run ci`)**: Runs build, all static checks (`check`), and logical test suites (`test`, `test-dev`).

---

## Project Structure & Architecture

```terminal
paykit-go/
├── gateway/          # Provider registry and gateway abstractions
├── bogus/            # Test gateway implementation
├── mpesa/            # Safaricom M-Pesa provider adapter and client
├── airtel/           # Airtel Money provider adapter and client
├── pesapal/          # Pesapal provider adapter and client
├── callback/         # Shared webhook and callback parsing/verification utilities
├── driver/           # Import-all entrypoint for drivers
└── docs/             # Architecture decision records and specifications
```

### Capability-Based Interface Architecture

Each provider lives in its own package and implements shared capability interfaces defined by the root package (`Gateway`, `Disburser`, `WebhookHandler`, `BalanceChecker`).
Providers normalize vendor-specific payloads into standard PayKit-Go domain types (`ChargeRequest`, `ChargeResponse`, `StatusResponse`, etc.).

Refer to [ADR 0001: Capability-Based Interface Architecture](docs/adr/0001-capability-based-architecture.md) for detailed rationale, capability composition patterns, and provider contracts.

---

## Coding Guidelines

- Format all Go code and imports with `mise run format` (or `goimports -w .`).
- Write clean, idiomatic, and concurrent-safe Go code.
- Keep functions small, focused, and free of unnecessary abstractions.
- Prefer composition over inheritance.

### Package Design & Naming

- **Shallow Nesting**: Avoid deep package hierarchies (e.g., avoid `internal/transport/http/client/v1`). Keep the package layout shallow, clean, and flat (maximum 1–2 levels deep).
- **Idiomatic Naming**: Use short, lowercase, single-word package names (`paykit`, `mpesa`, `airtel`, `pesapal`, `gateway`). Never use underscores, dashes, or mixedCaps in package names.
- **No Stutter**: Avoid redundant names where the package name repeats the enclosing directory or type (e.g., avoid `transport.TransportClient` or `client.ClientConfig`).

### Cross-Package Exported Symbol Documentation

- **Purpose & Intent**: Every exported type, interface, function, method, and package-level constant must be documented with conventional Godoc comments explaining its purpose and consumer usage.
- **Consumer-Centric Focus**: Godoc comments must clearly explain **when** to use the symbol and **how** to use it without coupling consumers to internal mechanics. The intent of each exported symbol is to expose a specific public capability or piece of data.
- **Caller-Facing Side Effects**: Disclose caller-facing side effects explicitly in Godoc comments (e.g., consuming or mutating an `io.Reader` such as `r.Body`, acquiring locks, modifying inputs, or mutating persistent state).
- **Implementation Comments**: Internal mechanics, algorithmic choices, and implementation details must be documented using regular internal comments (`// ...`) within function bodies or unexported symbols, never in public Godoc comments.

---

## Testing Standards

Every logical contribution to the project must include comprehensive tests.

### Testing Hierarchy: Blackbox vs. Whitebox

- **Blackbox Tests (`package <pkg>_test`)**:
  - Focus on verifying business logic, public API contracts, and integration flows from the consumer's perspective.
  - Must remain immune to internal implementation changes.
  - Blackbox tests are long-lived; any failure indicates a breaking change in core behavior that must be explicitly accounted for and documented.
- **Whitebox Tests (`package <pkg>`)**:
  - Focus on testing internal implementation edge cases, nil/zero-value defenses, unexported helper functions, and specific error-mapping paths that the implementation may fall prey to.

Run all tests with the race detector before submitting a pull request:

```bash
go test -v -race ./...
```

### Test Fixtures & Mock Data

- Fixtures for provider responses belong inside each provider's `fixtures/` directory.
- Fixtures and mocks must be **deterministic, fictional, centralized, resettable**, and compliant with provider API contracts.
- **Never** copy real production records or personally identifiable information (PII) into mock data or test fixtures.
- **Zero secrets** committed to the repository.

---

## Commit Messages

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <short description>
```

Common types include:

- `feat`: A new feature or capability.
- `fix`: A bug fix or defect correction.
- `docs`: Documentation updates only.
- `refactor`: Code refactoring that neither fixes a bug nor adds a feature.
- `test`: Adding missing tests or correcting existing tests.
- `chore`: Tooling, configuration, or repository maintenance.
- `perf`: Performance improvements.
- `ci`: CI configuration or script updates.

Common scopes include:

- `mpesa`, `airtel`, `pesapal`, `gateway`, `callback`, `http`, `core`, `types`.

Examples:

```terminal
feat(mpesa): add OAuth token caching
fix(callback): validate callback signature
docs: update contributing guide and pr template
test(mpesa): add stk push fixtures
refactor(http): simplify retry logic
chore: initialize project structure
```

---

## Pull Request Guidelines

### Pull Request Template Conformance

When opening a pull request, use the repository template at [`.github/pull_request_template.md`](.github/pull_request_template.md).

The template provides a structured format to help reviewers understand author intent, architectural impact, and verification evidence:

- **Summary**: Concise overview and problem context.
- **Background & Context (Optional)**: Root cause analysis, provider specifications, or architectural motivations.
- **Key Changes**: Explaining what changed, how it changed, and why it changed for each component or area.
- **Architecture & Public API Impact (Optional)**: Exported API additions/modifications, capability interface conformance, ADR alignment, or breaking changes.
- **Verification & Testing**: Concrete commands executed (`mise run ci`, `go test -v -race ./...`) and pass results.
- **Acceptance Evidence**: Tangible artifacts demonstrating that acceptance criteria are met.
- **Notes & Follow-ups (Optional)**: Ancillary context, provider limitations, or follow-up items.

Contributors are encouraged to adapt or remove optional sections when they do not apply, keeping the review focused and clutter-free.

> Note: We avoid markdown checkboxes (`- [ ]`) in PR descriptions to prevent GitHub from displaying misleading task list counters on pull requests.

### Acceptance Evidence Expectations

To maintain high code quality and streamline review, provide tangible evidence of verification matching the modified area(s):

- **Automated Test Output**: `go test -v -race ./...` test runner summary or coverage output.
- **Provider Sandbox / Simulator Logs**: Sanitized request/response exchanges against provider sandbox environments (e.g., Safaricom Daraja sandbox, Airtel Money simulator, Pesapal sandbox).
- **Deterministic Fixture Assertions**: Output demonstrating deterministic test fixture passes.

### Recommended Pre-Submission Checklist

Before finalizing your pull request and requesting review, verify the following:

- **Branch Freshness**: Your branch is rebased on current `origin/main`.
- **Local CI Cleanliness**: `mise run pre-push` or `mise run ci` passes cleanly (`build`, `check`, `test`, `test-dev`).
- **Tests & Coverage**: Comprehensive blackbox and whitebox tests are included.
- **Exported Symbols**: Exported types, interfaces, and methods are documented with consumer-centric Godoc comments disclosing side effects.
- **Architecture Alignment**: Changes conform to ADR 0001 capability interfaces and error envelopes.
- **Acceptance Evidence**: Verification evidence is included in the PR description.
- **Clean Git Footprint**: Commits are atomic and clean, with zero leaked machine paths, local worktrees, or internal planning scratchpads.

---

## Adding a New Payment Provider

When implementing a new payment provider:

1. Create a dedicated package (`<providername>/`).
2. Implement the shared capability interfaces (`Gateway`, `Disburser`, `WebhookHandler`, etc.) as defined in `gateway.go`.
3. Handle provider authentication internally (e.g., token management, caching, signature generation).
4. Normalize vendor responses into common PayKit-Go response types (`ChargeResponse`, `StatusResponse`, etc.).
5. Include deterministic test fixtures and mocked HTTP test coverage.
6. Provide comprehensive blackbox and whitebox tests.
7. Document provider-specific configuration and register the driver if appropriate.

---

## Reporting Issues

When reporting an issue, please include:

- A clear description of the problem.
- Minimal reproducible example or steps to reproduce.
- Expected behavior vs. actual behavior.
- Go version (`go version`) and operating system.
- Provider and endpoint involved (if applicable).

---

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all experience levels and strive to maintain a friendly, collaborative environment.

Happy coding! 🚀
