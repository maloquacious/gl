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
- Retrieval must support queries by transaction ID, date range, account, and entry
  type.
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

The first schema should include these tables:

- `ledgers`
- `accounts`
- `entry_types`
- `transactions`
- `entries`
- `sessions`

Likely supporting tables:

- `users` or `principals`
- `permissions`
- `integrity_checks`
- `schema_metadata` if not already covered by the migration package

Core relationships:

- `accounts.ledger_id` references `ledgers.id`.
- `entry_types.ledger_id` references `ledgers.id`.
- `transactions.ledger_id` references `ledgers.id`.
- `entries.transaction_id` references `transactions.id`.
- `entries.account_id` references `accounts.id`.
- `sessions.ledger_id` references `ledgers.id`.

Important constraints:

- Unique ledger names.
- Unique account IDs per ledger.
- Unique entry type codes per ledger.
- Unique transaction IDs per ledger.
- Unique entry IDs per ledger.
- Non-null debit-or-credit value constrained to `DEBIT` or `CREDIT`.
- Money amounts stored as exact integer minor units or exact decimal text, never
  floating point.

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

The first implementation should store amount values exactly. Prefer integer minor units
if scale is fixed per currency. If multi-currency scale handling is not ready yet, store
canonical decimal strings plus currency mnemonics and centralize arithmetic in the
domain package.

## Immutability And Audit Trail

Transactions and entries are append-only after posting. The REST API should not expose
update or delete operations for posted transactions or entries.

Administrative removal operations from the specification apply to ledgers and accounts
only under policy constraints. Removing an account must fail if the account has entries,
transactions, or a non-zero balance.

Entry IDs are audit trail identifiers. If the client does not provide them, the server may
allocate them. The allocation strategy must be stable, unique within the ledger, and safe
inside the posting transaction.

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
- `message`: human-readable detail.
- `field`: optional field path.
- `position`: optional list position for list operations.

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

## Open Questions

- Will the server aim for strict CORBA semantic compatibility or a pragmatic REST
  adaptation?
- Should transaction and entry IDs be client-supplied, server-supplied, or configurable
  per ledger?
- What authentication mechanism should replace the CORBA environment contract?
- Should account balances be stored eagerly, derived on read, or both?
- What exact representation should be used for multi-currency `Money` values?
- Should account and ledger removal physically delete rows, soft-delete rows, or be
  policy-configurable?

## Initial Milestones

1. Create the OpenAPI skeleton for ledgers, sessions, accounts, transactions, entries,
   and errors.
2. Add the Go package layout and server entry point.
3. Add the ZombieZen SQLite dependency and migration-aware pool.
4. Create the initial schema migration.
5. Implement session creation and ledger discovery.
6. Implement account lifecycle and retrieval.
7. Implement transaction posting atomically.
8. Implement transaction and entry retrieval.
9. Implement integrity checks.
10. Add conformance-oriented tests around the specification invariants.
