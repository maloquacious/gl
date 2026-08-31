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

- [Architecture](docs/architecture.md)
- [OpenAPI skeleton](api/openapi.yaml)
- OMG General Ledger Facility specification: <https://www.omg.org/spec/LEDG/1.0>

The local OMG specification PDF is kept in `docs/` for development reference but is
ignored by git.

## Implementation Plan

1. Finalize the first-pass OpenAPI contract around the OMG service groups and domain
   types.
2. Define Go domain types for ledgers, accounts, transactions, entries, money, dates,
   sessions, permissions, and specification-style errors.
3. Add the SQLite store package using ZombieZen's connection and migration packages.
4. Create the initial schema migration for ledgers, accounts, entry types,
   transactions, entries, sessions, and supporting indexes.
5. Implement store tests against temporary SQLite databases migrated from scratch.
6. Implement service-layer operations for ledger discovery, session creation, account
   lifecycle, account retrieval, and ledger metadata.
7. Implement atomic transaction posting, including balance validation, immutable entry
   creation, and account balance updates in a single SQLite transaction.
8. Implement transaction and entry retrieval by ID, date range, account, and entry
   type.
9. Implement integrity checks for balanced transactions, valid references, valid entry
   types, and stored account balances.
10. Add HTTP handlers that bind the tested service layer to the OpenAPI contract.
11. Add API-level tests for response schemas, error mapping, authorization behavior,
   and important ledger workflows.
12. Add operational polish: configuration, structured logging, health checks, graceful
   shutdown, and backup support.

The project should grow from the store and service invariants outward. REST routes are
the public contract, but the accounting rules belong in the domain and service layers
where they can be tested directly.
