# paykit-go
Unified Go SDK for Kenyan payment providers including M-Pesa, Airtel Money, Pesapal, Flutterwave and more.

## Project status

This project is currently under development.

## Planned providers

* M-Pesa
* Airtel Money
* Pesapal
* Bogus provider for local testing

## Architecture

PayKit-Go utilizes a capability-based interface architecture tailored for African payment rails (STK Push, USSD push prompts, and B2C disbursements). For complete architectural context and design decisions, see:
* [ADR 0001: Capability-Based Interface Architecture](docs/adr/0001-capability-based-architecture.md)

## Development

Run all tests:

```bash
go test ./...
```
