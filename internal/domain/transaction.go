// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"fmt"
	"slices"

	"github.com/maloquacious/gl/money"
)

// TransactionInfo is the summary of a transaction.
//
// TransactionID is allocated by the server, which section 2.1.7 leaves to the
// implementation, so it is empty on a transaction being posted and set on one
// read back. VoucherRef is how a client correlates the two: it documents the
// action that caused the posting, and section 2.1.14 defines it for that purpose.
type TransactionInfo struct {
	TransactionID         TransactionID
	VoucherRef            string
	VoucherDate           Date
	ActualTransactionDate Date
	PeriodID              PeriodID
}

// Transaction is a transaction and the entries that comprise it.
type Transaction struct {
	Info    TransactionInfo
	Entries []Entry
}

// MinEntriesPerTransaction is the smallest number of entries that can balance:
// one debit and one credit.
const MinEntriesPerTransaction = 2

// Validate checks a transaction against the ledger reporting currency.
//
// It validates a transaction as a client supplies it for posting, so it ignores
// the identifiers the server allocates. What it cannot check is existence: that
// each account and entry type belongs to the ledger is the service's to verify,
// because only the store knows the chart of accounts.
func (t Transaction) Validate(reporting money.Currency) error {
	if reporting == "" {
		return Faultf(ErrBadCurrencyMnemonic, "the ledger has no reporting currency and cannot accept a posting")
	}
	if len(t.Entries) < MinEntriesPerTransaction {
		return Faultf(ErrBadTransaction, "a transaction needs at least %d entries to balance, got %d", MinEntriesPerTransaction, len(t.Entries)).
			WithField("entries")
	}
	if _, err := NewPeriodID(string(t.Info.PeriodID)); err != nil {
		return withField(err, "transactionInfo.periodId")
	}
	for i, entry := range t.Entries {
		if err := entry.Validate(reporting); err != nil {
			return nestField(err, fmt.Sprintf("entries[%d]", i))
		}
	}
	debits, credits, err := t.Totals(reporting)
	if err != nil {
		return err
	}
	if balanced, err := debits.Equals(credits); err != nil {
		return Faultf(ErrBadTransaction, "cannot total the entries: %v", err).WithField("entries")
	} else if !balanced {
		return Faultf(ErrBadTransaction, "debits of %s do not balance credits of %s", debits, credits).
			WithField("entries")
	}
	return nil
}

// Totals sums the entry amounts on each side in the ledger reporting currency.
func (t Transaction) Totals(reporting money.Currency) (debits, credits money.Money, err error) {
	debits, err = money.Zero(reporting)
	if err != nil {
		return money.Money{}, money.Money{}, Faultf(ErrBadCurrencyMnemonic, "reporting currency is not registered").
			WithBadValue(string(reporting))
	}
	credits = debits
	for i, entry := range t.Entries {
		side := &credits
		if entry.IsDebit() {
			side = &debits
		}
		sum, err := side.Add(entry.Amount)
		if err != nil {
			return money.Money{}, money.Money{}, Faultf(ErrBadCurrencyMnemonic, "entry amount must be in the ledger reporting currency %s", reporting).
				WithBadValue(string(entry.Amount.Currency())).
				WithField(fmt.Sprintf("entries[%d].amount", i))
		}
		*side = sum
	}
	return debits, credits, nil
}

// SequenceEntries returns the entries in audit trail order.
//
// Entry order within a transaction is part of the audit trail, so the order the
// client supplied is preserved, with one exception the specification names in
// section 2.1.7: when a transaction debits and credits the same account, the
// debit comes first. Only the positions an account already occupies are
// rearranged, so entries for other accounts do not move.
func SequenceEntries(entries []Entry) []Entry {
	ordered := slices.Clone(entries)
	positions := make(map[AccountID][]int, len(ordered))
	for i, entry := range ordered {
		positions[entry.AccountID] = append(positions[entry.AccountID], i)
	}
	for _, slots := range positions {
		if len(slots) < 2 {
			continue
		}
		reordered := make([]Entry, 0, len(slots))
		for _, i := range slots {
			if ordered[i].IsDebit() {
				reordered = append(reordered, ordered[i])
			}
		}
		for _, i := range slots {
			if !ordered[i].IsDebit() {
				reordered = append(reordered, ordered[i])
			}
		}
		for n, i := range slots {
			ordered[i] = reordered[n]
		}
	}
	return ordered
}
