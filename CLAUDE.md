# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`github.com/maloquacious/gl` is a Go server implementing the OMG General Ledger
Facility specification v1.0 (LEDG) over HTTP. Go 1.26.4; the only non-stdlib
dependency so far is `github.com/maloquacious/semver`.

The repository is early. Only `money/` and `version.go` exist as code; `internal/`
is an empty placeholder. `docs/architecture.md` is the design of record and
`api/openapi.yaml` is the public contract — read both before adding packages, and
keep them updated when a decision changes. `docs/architecture.md` ends with an
"Open Questions" section; those are genuinely undecided, so raise them rather than
silently picking an answer.

## Commands

```sh
go build ./...
go vet ./...
go test ./...
go test ./money/                       # one package
go test ./money/ -run TestParseDecimal # one test (regexp, matches subtests too)
go test ./money/ -run 'TestParseDecimal/USD_cents'  # one subtest; spaces become underscores
gofmt -l .                             # must print nothing
```

There is no Makefile, linter config, or CI. Store tests will run against temporary
SQLite databases migrated from scratch through the same migration path as
production — do not hand-write test schemas.

## Architecture

Layering, outermost first (see `docs/architecture.md` for the full rationale):

```
HTTP handlers → OpenAPI types → application services → domain validation → store interfaces → SQLite
```

Accounting rules belong in the domain and service layers where they are directly
testable. HTTP handlers are a thin adaptation boundary: they translate requests
into service calls and domain errors into documented responses. The planned
layout is `cmd/gl-server`, `internal/api`, `internal/service`, `internal/domain`,
`internal/store`, `internal/store/sqlite`, `internal/migrations`.

Storage is SQLite via `zombiezen.com/go/sqlite` (not `database/sql`) with
`sqlitemigration` for schema. ZombieZen connections are not concurrency-safe:
borrow one connection per request or unit of work from the pool. Migrations are
embedded SQL and append-only after release.

### Invariants the design exists to protect

These come from the specification and drive both the schema and the service layer:

- Ledger names are unique in a facility; account IDs, entry type codes, and
  transaction IDs are unique within a ledger.
- Every transaction's debits and credits must balance in the ledger reporting
  currency.
- Posting is atomic: insert transaction, insert entries, update account balances,
  commit — or roll everything back.
- Posted transactions and entries are immutable. Do not add update or delete
  routes for them.
- Accounts in use (entries, transactions, or non-zero balance) cannot be removed.
- Money is never floating point. Store exact integer minor units or exact decimal
  text.

### OpenAPI and the error model

`api/openapi.yaml` (OpenAPI 3.1) is the contract, and it deliberately mirrors the
IDL rather than inventing REST names. Operation IDs are the specification's
operation names (`postTransaction`, `getTransInfoByDate`, `getDynamicSelection`),
and tags are the IDL interfaces (Arbitrator, Profile, Retrieval, BookKeeping,
LedgerLifecycle, Integrity, FacilityLifecycle). Keep that mapping when adding
endpoints.

Error payloads preserve the IDL exception names in `code` (`BadAccountId`,
`BadTransId`, `CannotRemove`, `PermissionDenied`, `UnknownLedger`, …; the full set
is the `ErrorCode` enum). The same code can appear under different HTTP statuses —
a missing account is 404 `BadAccountId`, a malformed one is 400 `BadAccountId`. So
the HTTP status carries the transport meaning and `code` carries the specification
meaning; keep both accurate. `field` and `position` locate the failure inside a
request body or a list operation.

## The money package

`money` is written to stand alone (it carries its own full MIT header block, unlike
the rest of the repo) and is currency-agnostic. `Money` is an unexported `int64` of
minor units plus a `Currency`, so the zero value is deliberately not usable — build
values with `NewMinor`, `MustNewMinor`, `ParseDecimal`, or `Zero`.

Conventions to follow when extending it:

- Every operation that combines or compares two values returns an error on
  currency mismatch (`ErrCurrencyMismatch`). There is no unchecked arithmetic, and
  no float conversion helpers — that omission is intentional.
- Scale comes from a mutex-guarded registry (`RegisterCurrency`, `Scale`); an
  unregistered currency is an error, not a default of 2.
- Every method in `money.go` has a mirrored package-level function in
  `functions.go` (`m.Add(x)` / `money.Add(m, x)`). Add both when adding an
  operation.
- JSON is `{"amount":"123.45","currency":"USD"}` — a canonical decimal string,
  matching the OpenAPI `Money` schema.

Ledger entries carry two money values: `orig_amount` in the source currency and
`amount` in the ledger reporting currency. Services validate that `amount` matches
the ledger's reporting currency while allowing `orig_amount` to differ.

## Conventions

- Tests use external test packages (`package money_test`), table-driven with named
  cases and `t.Run`.
- Files outside `money/` use the short header
  `// Copyright (c) <year> Michael D Henderson. All rights reserved.`
- `version.go` returns a `semver.Version` for the whole module; bump it there.
- Commit subjects are short and imperative ("Add currency-agnostic money package").
- The OMG specification PDF lives in `docs/` for reference but is gitignored, as
  are `*.db` files.
