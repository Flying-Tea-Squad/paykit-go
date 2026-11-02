# paykit-go Implementation Plan

## Repository Layout

```
paykit-go/
├── go.mod
├── go.sum
├── .github/workflows/ci.yml
│
├── gateway.go                  # package paykit  — Gateway interface
├── response.go                 # Response struct
├── errors.go                   # Standardized error codes
├── http.go                     # HTTPClient (retry, TLS, slog)
├── client.go                   # High-level Client wrapper (Phase 4)
├── scrubbing.go                # Scrubber interface (Phase 4)
│
├── gateway/
│   └── registry.go             # package gateway — Register(), New()
│
├── bogus/
│   ├── client.go               # BogusGateway (test double)
│   ├── client_test.go
│   └── fixture_test.go
│
├── mpesa/
│   ├── client.go               # MpesaGateway struct, constructor
│   ├── client_test.go
│   ├── auth.go                 # OAuth2 token fetching + caching
│   ├── signing.go              # password generation
│   ├── callback.go             # STK Push callback parsing
│   ├── types.go                # request/response structs
│   └── fixtures/               # recorded test responses
│       ├── stk_push_success.json
│       ├── stk_push_failure.json
│       ├── c2b_register.json
│       ├── b2c_success.json
│       └── ...
│
├── airtelmoney/
│   ├── client.go               # AirtelMoneyGateway
│   ├── client_test.go
│   ├── auth.go
│   ├── types.go
│   └── fixtures/
│
├── pesapal/
│   ├── client.go               # PesapalGateway
│   ├── client_test.go
│   ├── ipn.go                  # IPN registration + handling
│   ├── types.go
│   └── fixtures/
│
├── callback/                   # shared IPN/callback utility (Phase 3)
│   ├── parser.go               # parse common callback patterns
│   ├── verifier.go             # signature verification
│   └── parser_test.go
│
├── driver/                     # convenience — import all providers
│   └── import_all.go           // package driver; import _ "..."
│
├── docs/
│   └── CONTRIBUTING.md         # how to add a new provider
│
├── README.md
└── plan.md
```

## Naming Conventions

| Convention     | Rule                          | Example                      |
| -------------- | ----------------------------- | ---------------------------- |
| Package name   | lowercase, no underscores     | `mpesa`, `gateway`           |
| File name      | snake_case matching purpose   | `stk_push.go`, `callback.go` |
| Gateway type   | `{Provider}Gateway`           | `MpesaGateway`               |
| Constructor    | `New{Provider}`               | `NewMpesa`                   |
| Request types  | `{Action}Request`             | `STKPushRequest`             |
| Response types | `{Action}Response`            | `STKPushResponse`            |
| Tests          | `_test.go` adjacent to source | `client_test.go`             |
| Error vars     | `Err{Description}`            | `ErrAuthFailed`              |

## Design Decisions

| Decision      | Choice                                                                                                               |
| ------------- | -------------------------------------------------------------------------------------------------------------------- |
| Request types | Per-gateway (not shared) — M-Pesa STK Push and Airtel Money have different required fields                           |
| Response type | `Response` with shared fields (Success, Message, TransactionID, etc.) + `.Raw json.RawMessage` for full API response |
| Logging       | `log/slog` — no custom Logger interface                                                                              |
| Idempotency   | `IdempotencyKey string` on all mutating request structs                                                              |
| Callback/IPN  | Kept inside each provider initially; `callback/` extracted only if patterns converge (Phase 3)                       |
| Error codes   | Sentinel errors + standardized string codes mapped per gateway                                                       |
| Auth strategy | Each provider implements its own auth (M-Pesa OAuth2, Airtel Bearer, Pesapal API key)                                |

---

## Phase 0 — Foundation

**Goal:** Core types, interface, HTTP layer, and Bogus test gateway.

- Initialize `go.mod`
- Set up CI (GitHub Actions): `go test ./...`, `go vet ./...`, `golangci-lint`
- `gateway.go` — `Gateway` interface:
  ```go
  type Gateway interface {
      Purchase(ctx context.Context, req *PurchaseRequest) (*Response, error)
      Authorize(ctx context.Context, req *AuthorizeRequest) (*Response, error)
      Capture(ctx context.Context, req *CaptureRequest) (*Response, error)
      Void(ctx context.Context, req *VoidRequest) (*Response, error)
      Refund(ctx context.Context, req *RefundRequest) (*Response, error)
      QueryStatus(ctx context.Context, req *StatusRequest) (*Response, error)
      Name() string
  }
  ```
- `response.go` — `Response` struct (Success, Message, TransactionID, Authorization, Raw, ErrorCode, Metadata)
- `errors.go` — sentinel errors (`ErrCardDeclined`, `ErrProcessingError`, etc.) and a mapping function
- `http.go` — `HTTPClient` wrapping `net/http`:
  - Configurable timeouts, TLS, retry (exponential backoff, max 3 attempts)
  - `slog` integration for request/response logging
  - Dump writer support for debugging
- `gateway/registry.go` — package `gateway` with `Register()`, `New()`, internal `map[string]Factory` — imports **no** provider packages
- `bogus/` — test double implementing `Gateway`:
  - `Purchase` succeeds on amounts ending in `00`, fails on `05`
  - `Authorize`/`Capture`/`Void`/`Refund` follow same pattern
  - `init()` registers with `gateway.Register("bogus", ...)`
- Tests: `go test ./...` must pass

**Files created in this phase:**

- `go.mod`
- `.github/workflows/ci.yml`
- `gateway.go`
- `response.go`
- `errors.go`
- `http.go`
- `gateway/registry.go`
- `bogus/client.go`
- `bogus/client_test.go`

**Milestone:** `go test ./...` passes. User can `import _ "paykit-go/bogus"`.

---

## Phase 1 — M-Pesa (Daraja + Newer APIs)

**Goal:** First real provider. Full M-Pesa coverage including newer Safaricom APIs.

- `mpesa/client.go` — `MpesaGateway` struct embedding `BaseGateway`
  - Constructor: `NewMpesa(config MpesaConfig) (*MpesaGateway, error)`
  - `init()` calls `gateway.Register("mpesa", NewMpesa)`
  - Implements all `Gateway` methods
- `mpesa/auth.go` — OAuth2 token management:
  - `getAccessToken(ctx) (string, error)`
  - Caches token, refreshes on expiry
- `mpesa/signing.go` — `generatePassword(shortCode, passKey, timestamp) string` (Base64 encoded)
- `mpesa/types.go` — request/response types:
  - `STKPushRequest`, `STKPushResponse`
  - `C2BRegisterRequest`, `C2BSimulateRequest`
  - `B2CRequest`, `B2CResponse`
  - `StatusRequest`, `BalanceRequest`
  - `DynamicQRRequest`, `DynamicQRResponse`
- `mpesa/callback.go` — parse STK Push callback JSON payload, extract ResultCode, ResultDesc, Msisdn, etc.
- `mpesa/fixtures/*.json` — recorded API responses for test mocking

**Endpoints implemented:**

- STK Push (Lipa Na M-Pesa Online)
- STK Push Query
- C2B Register URL
- C2B Simulate
- B2C Payment
- Account Balance
- Transaction Status
- M-Pesa Express (Dynamic QR)
- Customer-to-Business Advanced

**Files created:**

- `mpesa/client.go`
- `mpesa/auth.go`
- `mpesa/signing.go`
- `mpesa/types.go`
- `mpesa/callback.go`
- `mpesa/client_test.go`
- `mpesa/fixtures/stk_push_success.json`
- `mpesa/fixtures/stk_push_failure.json`
- `mpesa/fixtures/c2b_register.json`
- `mpesa/fixtures/b2c_success.json`
- `mpesa/fixtures/balance.json`
- `mpesa/fixtures/status.json`
- `mpesa/fixtures/qr_success.json`

**Milestone:** `import _ "paykit-go/mpesa"` — all M-Pesa endpoints callable via `Gateway` interface.

---

## Phase 2 — Airtel Money

**Goal:** Second provider validates the interface with a different auth style.

- `airtelmoney/client.go` — `AirtelMoneyGateway` struct
  - Constructor `NewAirtelMoney(config AirtelConfig) (*AirtelMoneyGateway, error)`
  - `init()` registers with gateway registry
- `airtelmoney/auth.go` — Bearer token auth via OAuth2 client credentials
  - Token caching + refresh
- `airtelmoney/types.go` — request/response types:
  - `CollectionRequest` (USSD push)
  - `DisbursementRequest` (B2C)
  - `TransactionStatusRequest`
  - `ReconciliationRequest`
- `airtelmoney/fixtures/*.json` — test fixtures

**Files created:**

- `airtelmoney/client.go`
- `airtelmoney/auth.go`
- `airtelmoney/types.go`
- `airtelmoney/client_test.go`
- `airtelmoney/fixtures/collection_success.json`
- `airtelmoney/fixtures/disbursement_success.json`
- `airtelmoney/fixtures/status_response.json`

**Milestone:** Two providers, both implementing the same `Gateway` interface. Auth strategies differ (OAuth2 + password vs Bearer token).

---

## Phase 3 — Pesapal + Shared Callback/IPN

**Goal:** Third provider introduces IPN pattern. Extract shared IPN handling if patterns converge.

- `pesapal/client.go` — `PesapalGateway` struct
  - Constructor `NewPesapal(config PesapalConfig) (*PesapalGateway, error)`
  - `init()` registers with gateway registry
- `pesapal/ipn.go` — IPN registration flow:
  - Register IPN URL with Pesapal
  - Validate incoming IPN notifications
- `pesapal/types.go`:
  - `OrderRegistrationRequest`, `OrderStatusRequest`
  - `IPNNotification` struct
- `pesapal/fixtures/*.json`

- `callback/` — extracted only if M-Pesa + Pesapal callback/IPN patterns share structure:
  - `callback/parser.go` — reusable callback body parser
  - `callback/verifier.go` — signature verification (HMAC, RSA)
  - `callback/parser_test.go`

**Files created:**

- `pesapal/client.go`
- `pesapal/ipn.go`
- `pesapal/types.go`
- `pesapal/client_test.go`
- `pesapal/fixtures/order_success.json`
- `pesapal/fixtures/ipn_notification.json`
- `pesapal/fixtures/status_response.json`
- `callback/parser.go` (if patterns converge)
- `callback/verifier.go` (if needed)
- `callback/parser_test.go`

**Milestone:** Three providers. Shared callback package extracted if M-Pesa and Pesapal callback formats align.

---

## Phase 4 — DX & Polish

**Goal:** Developer experience, documentation, production-ready packaging.

- `client.go` — high-level `Client`:
  ```go
  type Client struct {
      Gateway Gateway
      Config  ClientConfig
  }

  func New(provider string, config ClientConfig) (*Client, error)
  ```
  - Wraps gateway selection and config populating common options (timeout, idempotency, etc.)
- `scrubbing.go` — `Scrubber` interface:
  ```go
  type Scrubber interface {
      Scrub(transcript string) string
  }
  ```
  - Each provider implements `Scrubber` to redact PII (phone numbers, tokens) from debug logs
- `driver/import_all.go` — convenience package that imports all providers:
  ```go
  // package driver
  import (
      _ "paykit-go/mpesa"
      _ "paykit-go/airtelmoney"
      _ "paykit-go/pesapal"
  )
  ```
- `README.md`:
  - Installation (`go get`)
  - Quickstart example
  - Provider feature table
  - Configuration reference
- `docs/CONTRIBUTING.md` — checklist for adding a new provider:
  1. Create `yourprovider/` package
  2. Implement `Gateway` interface
  3. Define request/response types
  4. Implement auth strategy
  5. Write tests with fixtures
  6. Optionally implement `Scrubber`
  7. Register via `init()`
- Godoc `Example*` functions on each provider

**Files created/modified:**

- `client.go`
- `scrubbing.go`
- `driver/import_all.go`
- `README.md`
- `docs/CONTRIBUTING.md`
- Godoc examples added to each provider

**Milestone:** Clean `go doc` output. Copy-paste examples in README. Contributor can follow a checklist.

---

## Phase 5 — Production Hardening (v1.0)

**Goal:** Battle-readiness for production use.

- `http.go` additions:
  - Rate limiter (per-provider config, token bucket)
  - OpenTelemetry tracing (span per HTTP call)
  - Configurable retry with exponential backoff (initial delay, max delay, max retries)
- `callback/verifier.go` — webhook signature verification helpers:
  - HMAC SHA256 verification
  - RSA public key verification
- Top-level `CHANGELOG.md`
- git tag `v1.0.0`

**Files created/modified:**

- `http.go` — rate limiter + otel + retry enhancements
- `callback/verifier.go` — webhook signature helpers
- `CHANGELOG.md`

**Milestone:** `v1.0.0` tagged and release published.

---

## Appendix: Key Interface Signatures

```go
// Gateway is the primary interface every provider implements.
type Gateway interface {
    Purchase(ctx context.Context, req *PurchaseRequest) (*Response, error)
    Authorize(ctx context.Context, req *AuthorizeRequest) (*Response, error)
    Capture(ctx context.Context, req *CaptureRequest) (*Response, error)
    Void(ctx context.Context, req *VoidRequest) (*Response, error)
    Refund(ctx context.Context, req *RefundRequest) (*Response, error)
    QueryStatus(ctx context.Context, req *StatusRequest) (*Response, error)
    Name() string
}

// Response is the standardized return type for all gateway operations.
type Response struct {
    Success       bool
    Message       string
    TransactionID string
    Authorization string
    Raw           json.RawMessage
    ErrorCode     string
    Metadata      map[string]any
}

// PurchaseRequest is the standard purchase request.
type PurchaseRequest struct {
    Amount         int
    Currency       string
    Phone          string
    Description    string
    IdempotencyKey string
    CallbackURL    string
    Metadata       map[string]string
}
```

## Appendix: Error Codes

```go
const (
    ErrIncorrectNumber   = "incorrect_number"
    ErrInvalidNumber     = "invalid_number"
    ErrInvalidExpiryDate = "invalid_expiry_date"
    ErrInvalidCVC        = "invalid_cvc"
    ErrExpiredCard       = "expired_card"
    ErrCardDeclined      = "card_declined"
    ErrProcessingError   = "processing_error"
    ErrDuplicate         = "duplicate_transaction"
    ErrAuthFailed        = "authentication_failed"
    ErrInsufficientFunds = "insufficient_funds"
    ErrTimeout           = "request_timeout"
)
```
