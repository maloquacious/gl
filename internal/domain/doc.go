// Copyright (c) 2026 Michael D Henderson.

// Package domain defines the General Ledger vocabulary and the accounting rules
// that are clearer in Go than in SQL.
//
// The names come from the OMG General Ledger Facility specification v1.0 (LEDG)
// rather than from the HTTP contract, so a reader with the IDL in hand can find
// the same identifiers, dates, chart kinds, entry types, accounts, transactions,
// entries, and exceptions here. The HTTP spelling of each of these is the API
// layer's business.
//
// The rules this package owns are the ones worth testing without a database or a
// server: a transaction balances in the ledger reporting currency, an entry
// amount is denominated in that currency, a date range includes both of its
// bounds and either bound may be left unset, and a debit precedes a credit on the
// same account.
package domain
