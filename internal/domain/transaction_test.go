// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"

	"github.com/maloquacious/gl/cerrs"
	"github.com/maloquacious/gl/internal/domain"
	"github.com/maloquacious/gl/money"
)

func TestTransactionValidate(t *testing.T) {
	tests := []struct {
		name      string
		reporting money.Currency
		entries   []domain.Entry
		periodID  domain.PeriodID
		want      cerrs.Error
		wantField string
	}{
		{
			name:      "balanced transaction",
			reporting: money.USD,
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 12345),
				newEntry("4000", domain.Credit, 12345),
			},
		},
		{
			name:      "balanced across several entries",
			reporting: money.USD,
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 10000),
				newEntry("1100", domain.Debit, 2345),
				newEntry("4000", domain.Credit, 12345),
			},
		},
		{
			name:      "debits do not balance credits",
			reporting: money.USD,
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 12345),
				newEntry("4000", domain.Credit, 12344),
			},
			want:      domain.ErrBadTransaction,
			wantField: "entries",
		},
		{
			name:      "one entry cannot balance",
			reporting: money.USD,
			entries:   []domain.Entry{newEntry("1000", domain.Debit, 12345)},
			want:      domain.ErrBadTransaction,
			wantField: "entries",
		},
		{
			name:      "no entries",
			reporting: money.USD,
			want:      domain.ErrBadTransaction,
			wantField: "entries",
		},
		{
			name:      "ledger without a reporting currency",
			reporting: "",
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 12345),
				newEntry("4000", domain.Credit, 12345),
			},
			want: domain.ErrBadCurrencyMnemonic,
		},
		{
			name:      "an invalid entry names its position",
			reporting: money.USD,
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 12345),
				func() domain.Entry {
					e := newEntry("4000", domain.Credit, 12345)
					e.EntryType = ""
					return e
				}(),
			},
			want:      domain.ErrBadEntryType,
			wantField: "entries[1].entryType",
		},
		{
			name:      "bad period id",
			reporting: money.USD,
			periodID:  " ",
			entries: []domain.Entry{
				newEntry("1000", domain.Debit, 12345),
				newEntry("4000", domain.Credit, 12345),
			},
			want:      domain.ErrBadTransaction,
			wantField: "transactionInfo.periodId",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := domain.Transaction{
				Info:    domain.TransactionInfo{VoucherRef: "INV-1001", PeriodID: tt.periodID},
				Entries: tt.entries,
			}
			err := txn.Validate(tt.reporting)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v; want %v", err, tt.want)
			}
			if tt.wantField == "" {
				return
			}
			fault, ok := errors.AsType[*domain.Fault](err)
			if !ok {
				t.Fatalf("error %v does not carry a fault", err)
			}
			if fault.Field() != tt.wantField {
				t.Fatalf("Field() = %q; want %q", fault.Field(), tt.wantField)
			}
		})
	}
}

func TestTransactionValidateIgnoresServerAllocatedIdentifiers(t *testing.T) {
	// Clients supply neither the transaction reference nor the entry references;
	// the server allocates both inside the posting transaction.
	txn := domain.Transaction{
		Entries: []domain.Entry{
			newEntry("1000", domain.Debit, 12345),
			newEntry("4000", domain.Credit, 12345),
		},
	}
	if err := txn.Validate(money.USD); err != nil {
		t.Fatal(err)
	}
}

func TestTransactionTotals(t *testing.T) {
	txn := domain.Transaction{
		Entries: []domain.Entry{
			newEntry("1000", domain.Debit, 10000),
			newEntry("1100", domain.Debit, 2345),
			newEntry("4000", domain.Credit, 12345),
		},
	}

	debits, credits, err := txn.Totals(money.USD)
	if err != nil {
		t.Fatal(err)
	}
	if debits.Amount() != 12345 {
		t.Fatalf("debits = %s; want USD 123.45", debits)
	}
	if credits.Amount() != 12345 {
		t.Fatalf("credits = %s; want USD 123.45", credits)
	}
}

func TestTransactionTotalsOfNoEntriesAreZero(t *testing.T) {
	var txn domain.Transaction

	debits, credits, err := txn.Totals(money.USD)
	if err != nil {
		t.Fatal(err)
	}
	if !debits.IsZero() || !credits.IsZero() {
		t.Fatalf("totals = %s, %s; want zero", debits, credits)
	}
	if debits.Currency() != money.USD {
		t.Fatalf("currency = %q; want USD", debits.Currency())
	}
}

func TestTransactionTotalsRejectMixedCurrencies(t *testing.T) {
	txn := domain.Transaction{
		Entries: []domain.Entry{
			newEntry("1000", domain.Debit, 12345),
			func() domain.Entry {
				e := newEntry("4000", domain.Credit, 12345)
				e.Amount = money.MustNewMinor(12345, money.EUR)
				return e
			}(),
		},
	}

	_, _, err := txn.Totals(money.USD)
	if !errors.Is(err, domain.ErrBadCurrencyMnemonic) {
		t.Fatalf("error = %v; want ErrBadCurrencyMnemonic", err)
	}
	fault, ok := errors.AsType[*domain.Fault](err)
	if !ok {
		t.Fatalf("error %v does not carry a fault", err)
	}
	if fault.Field() != "entries[1].amount" {
		t.Fatalf("Field() = %q; want entries[1].amount", fault.Field())
	}
	if fault.BadValue() != "EUR" {
		t.Fatalf("BadValue() = %q; want EUR", fault.BadValue())
	}
}

func TestTransactionTotalsRejectAnUnregisteredReportingCurrency(t *testing.T) {
	var txn domain.Transaction
	if _, _, err := txn.Totals("XYZ"); !errors.Is(err, domain.ErrBadCurrencyMnemonic) {
		t.Fatalf("error = %v; want ErrBadCurrencyMnemonic", err)
	}
}

func TestSequenceEntries(t *testing.T) {
	type place struct {
		account domain.AccountID
		side    domain.DebitOrCredit
	}
	places := func(entries []domain.Entry) []place {
		got := make([]place, len(entries))
		for i, entry := range entries {
			got[i] = place{account: entry.AccountID, side: entry.DebitOrCredit}
		}
		return got
	}

	tests := []struct {
		name    string
		entries []domain.Entry
		want    []place
	}{
		{
			name: "client order is the audit trail order",
			entries: []domain.Entry{
				newEntry("4000", domain.Credit, 12345),
				newEntry("1000", domain.Debit, 12345),
			},
			want: []place{{"4000", domain.Credit}, {"1000", domain.Debit}},
		},
		{
			name: "a debit precedes a credit on the same account",
			entries: []domain.Entry{
				newEntry("1000", domain.Credit, 500),
				newEntry("1000", domain.Debit, 500),
			},
			want: []place{{"1000", domain.Debit}, {"1000", domain.Credit}},
		},
		{
			name: "entries for other accounts do not move",
			entries: []domain.Entry{
				newEntry("1000", domain.Credit, 500),
				newEntry("4000", domain.Credit, 200),
				newEntry("1000", domain.Debit, 700),
			},
			want: []place{{"1000", domain.Debit}, {"4000", domain.Credit}, {"1000", domain.Credit}},
		},
		{
			name: "order within a side is preserved",
			entries: []domain.Entry{
				newEntry("1000", domain.Credit, 100),
				newEntry("1000", domain.Credit, 200),
				newEntry("1000", domain.Debit, 300),
			},
			want: []place{{"1000", domain.Debit}, {"1000", domain.Credit}, {"1000", domain.Credit}},
		},
		{name: "no entries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.SequenceEntries(tt.entries)
			if len(got) != len(tt.entries) {
				t.Fatalf("SequenceEntries returned %d entries; want %d", len(got), len(tt.entries))
			}
			for i, want := range tt.want {
				if places(got)[i] != want {
					t.Fatalf("entry %d = %+v; want %+v", i, places(got)[i], want)
				}
			}
		})
	}
}

func TestSequenceEntriesKeepsAmountsWithTheirEntries(t *testing.T) {
	entries := []domain.Entry{
		newEntry("1000", domain.Credit, 100),
		newEntry("1000", domain.Debit, 300),
		newEntry("1000", domain.Credit, 200),
	}

	got := domain.SequenceEntries(entries)
	want := []int64{300, 100, 200}
	for i, amount := range want {
		if got[i].Amount.Amount() != amount {
			t.Fatalf("entry %d amount = %d; want %d", i, got[i].Amount.Amount(), amount)
		}
	}

	// The input is left alone, so a caller can report what the client sent.
	if entries[0].Amount.Amount() != 100 || !entries[0].IsCredit() {
		t.Fatal("SequenceEntries modified the entries it was given")
	}
}
