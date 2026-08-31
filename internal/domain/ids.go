// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"strings"
	"unicode"

	"github.com/maloquacious/gl/cerrs"
	"github.com/maloquacious/gl/money"
)

// The specification types every identifier as a wstring. These named string types
// keep an account reference from being passed where a transaction reference
// belongs, which the compiler can check and a wstring cannot.
type (
	// LedgerName names a ledger. Ledger names are unique within a facility.
	LedgerName string

	// AccountID references an account. Account references are unique within a ledger.
	AccountID string

	// TransactionID references a transaction. The server allocates these.
	TransactionID string

	// EntryID is the audit trail number for an entry. The server allocates these.
	EntryID string

	// EntryType is the mnemonic of an entry type, such as JournalDebit.
	EntryType string

	// PeriodID references a user-defined accounting period. It may be empty.
	PeriodID string
)

// NewLedgerName validates a ledger name.
//
// The specification gives create_ledger_chart_of_accounts and remove_ledger only
// UnknownLedger to raise for a name it will not accept, so a malformed name
// reports the same code as a missing one.
func NewLedgerName(value string) (LedgerName, error) {
	return newIdentifier[LedgerName](value, ErrUnknownLedger, "ledger name")
}

// NewAccountID validates an account reference.
func NewAccountID(value string) (AccountID, error) {
	return newIdentifier[AccountID](value, ErrBadAccountID, "account id")
}

// NewTransactionID validates a transaction reference.
func NewTransactionID(value string) (TransactionID, error) {
	return newIdentifier[TransactionID](value, ErrBadTransID, "transaction id")
}

// NewEntryID validates an entry reference.
//
// Entry references are allocated by the server rather than supplied by a client,
// so a malformed one is a defect on the way out, not bad input on the way in. It
// reports BadTransaction because an entry reaches a client only as part of a
// transaction, and the IDL declares no exception of its own for entries.
func NewEntryID(value string) (EntryID, error) {
	return newIdentifier[EntryID](value, ErrBadTransaction, "entry id")
}

// NewEntryType validates an entry type mnemonic.
func NewEntryType(value string) (EntryType, error) {
	return newIdentifier[EntryType](value, ErrBadEntryType, "entry type")
}

// NewPeriodID validates an accounting period reference. The period is optional in
// the contract, so an empty value is accepted and reports as unset.
func NewPeriodID(value string) (PeriodID, error) {
	if value == "" {
		return "", nil
	}
	return newIdentifier[PeriodID](value, ErrBadTransaction, "period id")
}

// NewCurrency validates a currency mnemonic against the money registry.
//
// An unregistered mnemonic is an error rather than an assumed scale of 2, because
// the server cannot store an exact minor-unit amount for a currency whose scale
// it does not know.
func NewCurrency(value string) (money.Currency, error) {
	currency := money.Currency(value)
	if _, ok := money.Scale(currency); !ok {
		return "", Faultf(ErrBadCurrencyMnemonic, "currency mnemonic is not registered").WithBadValue(value)
	}
	return currency, nil
}

// ValidateAccountName validates an account description. The IDL calls the
// description an account name when it rejects one, hence BadAccountName.
func ValidateAccountName(value string) error {
	if flaw := identifierFlaw(value); flaw != "" {
		return Faultf(ErrBadAccountName, "account description %s", flaw).WithBadValue(value)
	}
	return nil
}

// newIdentifier validates value and converts it to an identifier type, reporting
// the specification code that belongs to that identifier.
func newIdentifier[T ~string](value string, code cerrs.Error, what string) (T, error) {
	if flaw := identifierFlaw(value); flaw != "" {
		var zero T
		return zero, Faultf(code, "%s %s", what, flaw).WithBadValue(value)
	}
	return T(value), nil
}

// identifierFlaw returns why value is unusable as an identifier, or an empty
// string when it is usable.
//
// The contract asks only for a minimum length of one. The rest of these rules
// exist because identifiers are compared for uniqueness and echoed back in error
// payloads: surrounding whitespace makes two visibly identical identifiers
// distinct rows, and a control character corrupts whatever renders it.
func identifierFlaw(value string) string {
	switch {
	case value == "":
		return "must not be empty"
	case strings.TrimSpace(value) != value:
		return "must not begin or end with whitespace"
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "must not contain control characters"
		}
	}
	return ""
}
