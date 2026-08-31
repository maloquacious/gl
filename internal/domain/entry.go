// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"errors"

	"github.com/maloquacious/gl/money"
)

// DebitOrCredit records which side of an account an entry posts to. The IDL
// declares an enum; the value is kept as its contract name because that is also
// what the entries table stores.
type DebitOrCredit string

const (
	Credit DebitOrCredit = "CREDIT"
	Debit  DebitOrCredit = "DEBIT"
)

// Valid reports whether the value is one of the two sides.
func (d DebitOrCredit) Valid() bool { return d == Credit || d == Debit }

// ParseDebitOrCredit converts a contract name to a side. A value that is neither
// side arrives while posting, so it reports BadTransaction.
func ParseDebitOrCredit(value string) (DebitOrCredit, error) {
	side := DebitOrCredit(value)
	if !side.Valid() {
		return "", Faultf(ErrBadTransaction, "entry must be either DEBIT or CREDIT").WithBadValue(value)
	}
	return side, nil
}

// EntryTypeInfo is an entry type mnemonic and its description.
type EntryTypeInfo struct {
	Type        EntryType
	Description string
}

// ValidateEntryTypeInfoList validates a replacement set of entry types.
//
// set_entry_types replaces the whole list, so it is checked as a whole: every bad
// member is collected rather than only the first, which is why
// BadEntryTypeInfoList is the one exception carrying a list of offending values.
func ValidateEntryTypeInfoList(entryTypes []EntryTypeInfo) error {
	var bad []EntryTypeInfo
	seen := make(map[EntryType]bool, len(entryTypes))
	for _, info := range entryTypes {
		switch {
		case identifierFlaw(string(info.Type)) != "":
			bad = append(bad, info)
		case identifierFlaw(info.Description) != "":
			bad = append(bad, info)
		case seen[info.Type]:
			bad = append(bad, info)
		default:
			seen[info.Type] = true
		}
	}
	if len(bad) != 0 {
		return Faultf(ErrBadEntryTypeInfoList, "%d of %d entry types are unusable: each needs a mnemonic and a description, and no mnemonic may repeat", len(bad), len(entryTypes)).WithBadValues(bad...)
	}
	return nil
}

// Entry is a single accounting entry.
//
// An entry carries two amounts. Amount is denominated in the ledger reporting
// currency and is the one that has to balance; OriginalAmount records what the
// source transaction was denominated in and may be any registered currency. The
// server converts between them never - a client supplies both, which is what the
// specification intends by pointing at the Currency Facility for rates.
type Entry struct {
	TransactionID  TransactionID
	EntryID        EntryID
	EntryDate      Date
	EntryType      EntryType
	AccountID      AccountID
	OriginalAmount money.Money
	Amount         money.Money
	DebitOrCredit  DebitOrCredit
	Description    string
	VoucherRef     string
}

// IsDebit reports whether the entry posts to the debit side.
func (e Entry) IsDebit() bool { return e.DebitOrCredit == Debit }

// IsCredit reports whether the entry posts to the credit side.
func (e Entry) IsCredit() bool { return e.DebitOrCredit == Credit }

// Validate checks an entry against the ledger reporting currency.
//
// It validates the entry as a client supplies it for posting, so it ignores
// TransactionID and EntryID: the server allocates both inside the posting
// transaction and neither is accepted on input.
func (e Entry) Validate(reporting money.Currency) error {
	if reporting == "" {
		return Faultf(ErrBadCurrencyMnemonic, "the ledger has no reporting currency and cannot accept a posting")
	}
	if !e.EntryDate.IsSet() {
		return Faultf(ErrBadTransaction, "entry date must be set").WithField("entryDate")
	}
	if _, err := NewEntryType(string(e.EntryType)); err != nil {
		return withField(err, "entryType")
	}
	if _, err := NewAccountID(string(e.AccountID)); err != nil {
		return withField(err, "accountId")
	}
	if !e.DebitOrCredit.Valid() {
		return Faultf(ErrBadTransaction, "entry must be either DEBIT or CREDIT").
			WithBadValue(string(e.DebitOrCredit)).
			WithField("debitOrCredit")
	}
	if e.Amount.Currency() != reporting {
		return Faultf(ErrBadCurrencyMnemonic, "entry amount must be in the ledger reporting currency %s", reporting).
			WithBadValue(string(e.Amount.Currency())).
			WithField("amount")
	}
	if _, ok := money.Scale(e.OriginalAmount.Currency()); !ok {
		return Faultf(ErrBadCurrencyMnemonic, "original amount currency is not registered").
			WithBadValue(string(e.OriginalAmount.Currency())).
			WithField("originalAmount")
	}
	// The side an entry posts to carries its sign, so a negative amount would
	// state the direction twice and could state it two different ways.
	if e.Amount.IsNegative() {
		return Faultf(ErrBadTransaction, "entry amount must not be negative; use DEBIT or CREDIT to give it direction").
			WithBadValue(e.Amount.Decimal()).
			WithField("amount")
	}
	if e.OriginalAmount.IsNegative() {
		return Faultf(ErrBadTransaction, "original amount must not be negative; use DEBIT or CREDIT to give it direction").
			WithBadValue(e.OriginalAmount.Decimal()).
			WithField("originalAmount")
	}
	return nil
}

// withField records a request field path on a fault that does not already name
// one, so that the innermost rule to reject a value gets to name it.
func withField(err error, field string) error {
	if fault, ok := errors.AsType[*Fault](err); ok && fault.Field() == "" {
		return fault.WithField(field)
	}
	return err
}

// nestField qualifies a fault's field path with the path of the member that
// contains it, so that a rule rejecting an entry's type reports
// entries[1].entryType rather than either half alone.
func nestField(err error, prefix string) error {
	fault, ok := errors.AsType[*Fault](err)
	if !ok {
		return err
	}
	if fault.Field() == "" {
		return fault.WithField(prefix)
	}
	return fault.WithField(prefix + "." + fault.Field())
}
