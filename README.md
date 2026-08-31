# OMG General Ledger Server

This project implements a server for the OMG General Ledger Facility specification
version 1.0.

The goal is to provide a modern HTTP service for general ledger operations while
preserving the core semantics of the original OMG specification: ledgers, charts of
accounts, balanced transaction posting, immutable entries, retrieval operations,
administrative lifecycle operations, session-scoped access, and integrity checks.

The implementation direction is:

- Go for the server implementation.
- SQLite3 for durable storage.
- `zombiezen.com/go/sqlite` for SQLite access.
- `zombiezen.com/go/sqlite/sqlitemigration` for schema migrations.
- OpenAPI for the REST API contract.

## Documentation

- [Architecture](docs/architecture.md) - the design of record, including the
  settled design decisions
- [OpenAPI contract](api/openapi.yaml)
- OMG General Ledger Facility specification: <https://www.omg.org/spec/LEDG/1.0>

The local OMG specification PDF is kept in `docs/` for development reference but is
ignored by git.

## Status

The OpenAPI contract and the design decisions behind it are settled. `money` and
`cerrs` are implemented. The server itself is not started yet: step 1 below is the
next piece of work.

## Implementation Plan

Each step carries its own tests. Testing is not a later phase.

1. **Domain package.** `internal/domain` defines identifiers, dates with optional
   inclusive bounds, entry types, chart kinds, debit and credit, and the domain
   errors, declared as `cerrs.Error` constants mirroring the OpenAPI `ErrorCode`
   enum. Reuses `money` as it stands.
2. **Store connection setup and the initial migration.** `internal/store/sqlite`
   opens a `zombiezen.com/go/sqlite` pool whose connection-init hook applies WAL,
   `PRAGMA foreign_keys = ON`, and a busy timeout, so every borrowed connection is
   already configured. The first `internal/migrations` script creates `ledgers`,
   `accounts`, `entry_types`, `transactions`, `entries`, `sessions`, `principals`,
   `permissions`, and the retrieval indexes.
3. **Store interfaces** in `internal/store` and their SQLite implementation, tested
   against temporary databases migrated through the same pool path as production.
4. **One vertical slice, end to end.** `cmd/gl-server` with configuration and a
   health endpoint, then `internal/service` operations and `internal/api` handlers
   for `createLedgerChartOfAccounts`, `openSession`, `createAccount`,
   `postTransaction`, and `getTransaction`, with tests at both layers. Posting is
   atomic here or nowhere: transaction row, entry rows, and balance updates commit
   together or roll back together. This step exists to put weight on the contract
   early, while changing it is still cheap.
5. **LedgerLifecycle and Profile.** `modifyAccount`, `removeAccount` and
   `removeLedger` with `CannotRemove` enforced, `setLedgerCurrency`, `setEntryTypes`,
   `getProfile`, `getLedgerCurrency`, `getEntryTypes`, and `closeSession`.
6. **Retrieval.** The account, transaction, entry, and count operations, by
   identifier, date range, account, and entry type.
7. **`postTransactionList`**, reporting per-transaction failures as
   `BadTransactionsInList` with the failing `position`.
8. **Integrity checks**, returning which invariant failed and where, not just a
   boolean. Concurrency and rollback tests belong here.
9. **Operational polish.** Structured logging, request IDs, graceful shutdown, and
   backup support.

The project grows from the store and service invariants outward. REST routes are the
public contract, but the accounting rules belong in the domain and service layers
where they can be tested directly.
