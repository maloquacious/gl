# Architecture

## Purpose

This project implements a server for the OMG General Ledger Facility specification
version 1.0. The server provides the General Ledger back end described by the
specification: client session establishment, discriminatory access to ledger services,
transaction posting, retrieval, ledger lifecycle operations, facility lifecycle operations,
and integrity checks.

The initial implementation will use:

- Go for the server.
- SQLite3 for durable storage.
- `zombiezen.com/go/sqlite` for SQLite access.
- `zombiezen.com/go/sqlite/sqlitemigration` for schema migrations.
- OpenAPI for documenting and validating the REST API.

## References

- OMG General Ledger Facility specification: [`docs/omg-spec-LEDG-1.0-01-02-67.pdf`](omg-spec-LEDG-1.0-01-02-67.pdf)
- OMG specification page: <https://www.omg.org/spec/LEDG/1.0>
- ZombieZen SQLite package: <https://pkg.go.dev/zombiezen.com/go/sqlite>
- ZombieZen migration package: <https://pkg.go.dev/zombiezen.com/go/sqlite/sqlitemigration>

## Architectural Drivers

The specification defines a distributed object interface using CORBA IDL, but the
server will expose equivalent service semantics over HTTP. The REST API is an
adaptation boundary, not a redesign of the ledger model.

The most important constraints are:

- A facility contains one or more uniquely named ledgers.
- Each ledger has exactly one chart of accounts.
- Account identifiers are unique within a ledger.
- Accounts are created with a zero balance.
- Accounts in use cannot be removed.
- Transaction identifiers are unique.
- Posted transactions and entries are immutable.
- Each transaction must contain balanced debit and credit entries.
- Posting a transaction must persist the transaction, persist its entries, and update
  affected account balances atomically.
- Entry order within a transaction is part of the audit trail. When a transaction
  debits and credits the same account, the debit must precede the credit
  (specification section 2.1.7).
- Retrieval must support queries by transaction ID, date range, account, and entry
  type.
- Retrieval date ranges are inclusive of both bounds, and either bound may be left
  unset (specification section 2.1.12).
- Integrity checks are explicit service operations.
- Access is session and permission dependent.

These constraints strongly favor a relational schema with database-enforced uniqueness,
foreign keys, indexed query paths, and explicit transactions.

## System Shape

The server is organized as a small set of layers:

```text
HTTP handlers
OpenAPI request/response types
Application services
Domain validation
Store interfaces
SQLite implementation
```

HTTP handlers translate requests into application service calls and translate domain
errors into documented HTTP responses.

Application services implement the GL operations from the specification. They own
authorization checks, transaction boundaries, validation order, and orchestration across
store methods.

Domain validation enforces accounting rules that are clearer in Go than SQL, such as
balanced transaction totals and date-range semantics.

Store interfaces keep SQLite details out of the service layer. The first store
implementation is SQLite-only; the interface boundary exists to keep tests fast and to
avoid leaking low-level database details upward.

## REST API Model

The REST API should map closely to the specification's interfaces:

- Arbitrator: ledger discovery and session creation.
- Profile: session metadata and access to service capabilities.
- Retrieval: accounts, transactions, entries, counts, and summaries.
- BookKeeping: posting one transaction or a list of transactions.
- LedgerLifecycle: account, reporting currency, and entry type administration.
- Integrity: available integrity checks and check execution.
- FacilityLifecycle: ledger creation and removal.

The API should be documented in OpenAPI from the beginning. The OpenAPI document is
the public contract for HTTP clients and should define:

- Paths, methods, parameters, and request bodies.
- Response schemas for all domain types.
- Error response schemas corresponding to specification exceptions.
- Authentication and authorization scheme placeholders.
- Stable operation IDs named after the GL operations where practical.

The IDL exception names should be preserved in API error payloads even when the HTTP
status code differs by context. For example, a missing account can return HTTP 404 with
an error code of `BadAccountId`, while an invalid account identifier can return HTTP
400 with the same specification-level code and a more precise message.

## Go Implementation

The Go code should favor explicit, small packages:

```text
cmd/gl-server        server executable
internal/api         generated or hand-written OpenAPI-facing types and handlers
internal/service     GL operation implementations
internal/domain      IDs, money, dates, validation, domain errors
internal/store       storage interfaces
internal/store/sqlite SQLite-backed store
internal/migrations  embedded SQL migration scripts
```

This layout keeps the server entry point thin and makes the specification operations
easy to find.

Go is a good fit because the service is primarily I/O-bound, operationally simple, and
benefits from easy deployment. The implementation should avoid reflection-heavy
frameworks until there is a demonstrated need.

## SQLite Access

The project will use `zombiezen.com/go/sqlite` rather than `database/sql`.
ZombieZen's package is a low-level, cgo-free SQLite interface backed by
`modernc.org/sqlite`. Connections are not used concurrently, so the application should
use the ZombieZen pool facilities and borrow a connection per request or per unit of
work.

Recommended connection setup:

- Enable foreign keys for every connection.
- Use WAL journal mode for normal server operation.
- Set a busy timeout.
- Keep all write operations inside explicit transactions.
- Use immediate transactions for posting operations that update balances.
- Keep read-only operations short and index-supported.

Posting a transaction should happen in one SQLite transaction:

1. Validate session and permission.
2. Validate transaction shape and entry references.
3. Validate debit and credit totals in reporting currency.
4. Insert the transaction row.
5. Insert entry rows.
6. Update account balances.
7. Commit.

If any step fails, the whole post rolls back.

## Migrations

Migrations live as embedded SQL scripts in `internal/migrations`. The server starts by
opening a migration-aware pool through `zombiezen.com/go/sqlite/sqlitemigration`.

Migration rules:

- Migrations are append-only after release.
- Each migration is deterministic SQL.
- Schema changes that need data backfills happen in the same migration transaction
  when practical.
- Foreign keys stay enabled by default.
- Tests should create temporary databases by running the same migration path as
  production.

The initial migration should create the core schema, indexes, and constraints. Later
migrations can add optional features such as audit metadata, idempotency keys, or
additional integrity-check bookkeeping.

## Initial Data Model

The first schema must include these tables:

- `ledgers`
- `accounts`
- `entry_types`
- `transactions`
- `entries`
- `sessions`
- `principals`
- `permissions`

`principals` and `permissions` belong in the first migration rather than a later one.
Opening a session takes a login name and password, so without them there is nothing to
authenticate against and no capability set to authorize against.

Possible later tables:

- `integrity_checks`
- `schema_metadata` if not already covered by the migration package

Core relationships:

- `accounts.ledger_id` references `ledgers.id`.
- `entry_types.ledger_id` references `ledgers.id`.
- `transactions.ledger_id` references `ledgers.id`.
- `entries.transaction_id` references `transactions.id`.
- `entries.account_id` references `accounts.id`.
- `sessions.ledger_id` references `ledgers.id`.
- `sessions.principal_id` references `principals.id`.
- `permissions.principal_id` references `principals.id`.
- `permissions.ledger_id` references `ledgers.id`.

Important constraints:

- Unique ledger names.
- Unique account IDs per ledger.
- Unique entry type codes per ledger.
- Unique transaction IDs per ledger.
- Unique entry IDs per ledger.
- Non-null debit-or-credit value constrained to `DEBIT` or `CREDIT`.
- `entries` carries a `sequence` column, unique per transaction, that fixes audit
  trail order. Entries are always read back in `sequence` order, and the posting
  service assigns it so that a debit precedes a credit on the same account.
- Money amounts stored as exact integer minor units plus a currency mnemonic, never
  floating point. An entry therefore carries four columns: `amount_minor` with
  `currency` for the reporting amount, and `orig_amount_minor` with `orig_currency`
  for the source amount.

Indexes should support all retrieval operations:

- Transactions by ledger and transaction ID.
- Transactions by ledger and posting date.
- Entries by ledger, account, and entry date.
- Entries by ledger, entry type, and entry date.
- Entries by transaction.

## Money And Currency

The specification uses the OMG Currency Facility `Money` type and distinguishes
original amount from reporting amount. The server should represent money explicitly:

- `orig_amount` records the amount and currency used by the source transaction.
- `amount` records the amount in the ledger reporting currency.
- Ledger reporting currency is a single ISO-style mnemonic per ledger.

Amounts are stored exactly, as integer minor units plus a currency mnemonic, with the
scale for each currency coming from the `money` package registry. See Decisions below.

The server performs no currency conversion. A client supplies both `orig_amount` and
`amount`, and the service validates only that `amount` is denominated in the ledger
reporting currency; `orig_amount` may be any registered currency. Converting between
them is the client's responsibility, which is what the specification intends when it
points at the OMG Currency Facility for historical exchange rates.

## Immutability And Audit Trail

Transactions and entries are append-only after posting. The REST API should not expose
update or delete operations for posted transactions or entries.

Administrative removal operations from the specification apply to ledgers and accounts
only under policy constraints. Removing an account must fail if the account has entries,
transactions, or a non-zero balance.

Entry IDs are audit trail identifiers and are always allocated by the server; the API
does not accept one on input. The allocation strategy must be stable, unique within the
ledger, and safe inside the posting transaction. Transaction IDs are allocated the same
way. See Decisions below.

Because entry order is itself part of the audit trail, the posting service assigns each
entry its `sequence` within the transaction and preserves the order the client supplied,
except that a debit on the same account is forced ahead of the credit.

## Sessions And Authorization

The specification assumes authentication and access control are handled by the
environment. The HTTP server still needs an adaptation for modern clients.

Initial approach:

- `POST /sessions` opens a session for a ledger.
- A session token identifies the selected ledger and principal.
- Handler middleware loads the session and selected ledger.
- Application services check operation permissions before doing domain work.

The first version may use a simple local user store. The architecture should keep this
replaceable so that external authentication can be added later.

## Error Model

Domain errors should preserve specification names:

- `BadDate`
- `BadChartKind`
- `BadTransaction`
- `BadTransactionsInList`
- `BadEntryType`
- `BadEntryTypeInfoList`
- `BadCurrencyMnemonic`
- `BadAccountId`
- `BadTransId`
- `CannotRemove`
- `PermissionDenied`
- `UnknownLedger`
- `BadIntegritySelection`
- `BadAccountName`

Each API error response should include:

- `code`: the specification error name.
- `message`: human-readable detail, from the exception's `error` member.
- `badValue`: the value that was rejected, from the exception's `bad_value`
  member. Nine of the fourteen exceptions declare one, so an error such as
  `BadAccountId` names the account it rejected instead of only saying that some
  account was bad.
- `badValues`: the rejected entry types, from `BadEntryTypeInfoList.bad_values`.
  This is the only exception carrying a list of offending values.
- `position`: optional list position for list operations, from
  `BadTransactionsInList.position_in_list`.
- `field`: optional field path. This one has no counterpart in the IDL; it is a
  REST convenience for locating a failure inside a request body.

## Integrity Checks

Integrity checks are named operations. The first implementation should include checks
for:

- Every transaction is balanced.
- Every entry references an existing account.
- Every entry type is valid for its ledger.
- Stored account balances match the sum of posted entries.
- Ledger, account, transaction, and entry uniqueness constraints hold.

The database should prevent many integrity failures by construction. The integrity
interface still matters because it gives operators an explicit way to verify ledger
health.

## Testing Strategy

Tests should focus on specification behavior:

- Unit tests for money arithmetic, date ranges, and validation.
- Store tests against temporary SQLite databases migrated from scratch.
- Service tests for each GL operation and exception mapping.
- API tests generated from or checked against the OpenAPI document.
- Concurrency tests for simultaneous reads and posting attempts.
- Crash/rollback tests for failed transaction posting.

Golden tests are useful for OpenAPI responses and error payloads.

## Operational Notes

The server should be deployable as a single Go binary. Runtime state is the SQLite
database file plus any configured backups.

Operational defaults:

- Structured logs.
- Request IDs.
- Health endpoint that checks database connectivity and migration health.
- Graceful shutdown.
- Periodic SQLite backup support.
- Configuration through environment variables and flags.

## Decisions

These were open questions during the first design pass. They are settled now because
each one shapes the schema or the public contract, and leaving them open blocks both.
Where the specification speaks, its text decides.

### Specification fidelity: a pragmatic REST adaptation

The server preserves the specification's vocabulary and invariants, not its object
model. Operation IDs are the IDL operation names, tags are the IDL interfaces, and
error codes are the IDL exception names, but resources, verbs, and status codes are
REST-native.

The clearest case is `Profile`. In the IDL it hands back service references
(`book_keeping()`, `retrieval()`, `integrity()`, and so on), each raising
`PermissionDenied`. Handing out object references means nothing over HTTP, so the
adaptation reports the same information as a capability list on the session profile.

### Identifiers: the server allocates both

The server allocates transaction references and entry references. Clients do not
supply either.

Section 2.1.7 makes this the implementation's choice outright: "GL Transaction
references may be ignored by implementations that automatically allocate these
references at the server." Server allocation is also why `post_transaction` returns a
`TransactionId` at all. Clients correlate a posted transaction with its source
document through `voucher_ref`, which section 2.1.14 defines for exactly that purpose.

Entry references are audit trail numbers. They are allocated inside the posting
transaction, monotonically per ledger, alongside the `sequence` that fixes entry order.

Making allocation configurable per ledger is deferred; nothing needs it yet.

### Authentication: opaque bearer token over a local principal store

`open_session(ledger_name, login_name, password)` is already the specification's
signature, so the REST contract keeps it and returns an opaque bearer token.

Behind it, `principals` holds login names and Argon2id password hashes, `permissions`
holds a capability set per principal per ledger, and `sessions` holds issued tokens
with an expiry. Closing a session deletes the row.

The service layer talks to an `Authenticator` interface rather than to the tables
directly, so an external identity provider can replace the local store later without
touching the operations.

Facility lifecycle operations are session-scoped like everything else. In the IDL,
`facility_lifecycle()` is reached through a `Profile`, so creating and removing
ledgers requires a session; the session's selected ledger is simply not consulted for
those two operations, only the principal's `facilityLifecycle` capability.

### Balances: stored eagerly, derived only to check

Account balances are maintained in the `accounts` row, updated inside the posting
transaction.

Deriving on read is not viable: `get_all_accounts` returns a balance for every
account, which would make each call scan every entry in the ledger. Derivation still
happens, but as the integrity check that reconciles stored balances against the sum of
posted entries. Both halves were already implied by the posting sequence and the
integrity check list; this states them together.

### Money: integer minor units plus a currency mnemonic

A monetary value is an `int64` count of minor units and a currency, which is what the
`money` package already implements. Scale per currency comes from the package's
registry, seeded at startup, and an unregistered currency is an error rather than a
default of 2.

`set_ledger_currency` rejects a mnemonic the registry does not know with
`BadCurrencyMnemonic`. Over the wire a money value stays a canonical decimal string
plus a mnemonic, matching the OpenAPI `Money` schema.

The specification's `create_ledger_chart_of_accounts` takes no currency, so a ledger
begins with no reporting currency and cannot accept a posting until one is set. The
REST contract adds an optional `currency` on ledger creation as a convenience, but the
underlying rule stands: no reporting currency, no posting.

### Removal: physical deletes behind an in-use check

Accounts and ledgers are deleted outright, never flagged.

This is safe because removal is only legal when nothing references the row. An account
with entries, transactions, or a non-zero balance fails with `CannotRemove`, and so
does a ledger holding transactions or entries. A row that survives the check has no
history to lose, and the foreign keys enforce that independently.

Soft deletion would add a "not deleted" predicate to every query in the system and buy
nothing here. Making the policy configurable is deferred.

## Implementation Sequence

The implementation plan lives in [`../README.md`](../README.md) and is the single
ordered list. This document describes the design; the README describes the order in
which it gets built.
