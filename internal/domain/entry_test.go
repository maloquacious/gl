// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/maloquacious/gl/cerrs"
	"github.com/maloquacious/gl/internal/domain"
	"github.com/maloquacious/gl/money"
)

// newEntry returns a postable entry in USD, the currency the entry and
// transaction tests use as the ledger reporting currency.
func newEntry(accountID string, side domain.DebitOrCredit, minor int64) domain.Entry {
	return domain.Entry{
		EntryDate:      domain.NewDate(time.Date(1998, time.December, 21, 9, 30, 0, 0, time.UTC)),
		EntryType:      "JournalEntry",
		AccountID:      domain.AccountID(accountID),
		OriginalAmount: money.MustNewMinor(minor, money.USD),
		Amount:         money.MustNewMinor(minor, money.USD),
		DebitOrCredit:  side,
	}
}

func TestDebitOrCredit(t *testing.T) {
	if !domain.Debit.Valid() || !domain.Credit.Valid() {
		t.Fatal("a side of the account is invalid")
	}
	if !newEntry("1000", domain.Debit, 1).IsDebit() || newEntry("1000", domain.Debit, 1).IsCredit() {
		t.Fatal("a debit entry does not report as one")
	}
	if !newEntry("1000", domain.Credit, 1).IsCredit() || newEntry("1000", domain.Credit, 1).IsDebit() {
		t.Fatal("a credit entry does not report as one")
	}

	tests := []struct {
		name    string
		value   string
		want    domain.DebitOrCredit
		wantErr bool
	}{
		{name: "debit", value: "DEBIT", want: domain.Debit},
		{name: "credit", value: "CREDIT", want: domain.Credit},
		{name: "lowercase", value: "debit", wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "neither", value: "REVERSAL", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseDebitOrCredit(tt.value)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrBadTransaction) {
					t.Fatalf("error = %v; want ErrBadTransaction", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ParseDebitOrCredit(%q) = %q; want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidateEntryTypeInfoList(t *testing.T) {
	tests := []struct {
		name       string
		entryTypes []domain.EntryTypeInfo
		wantBad    []domain.EntryType
	}{
		{
			name: "valid list",
			entryTypes: []domain.EntryTypeInfo{
				{Type: "JournalDebit", Description: "Journal Debit"},
				{Type: "JournalCredit", Description: "Journal Credit"},
			},
		},
		{name: "empty list clears the entry types"},
		{
			name: "missing mnemonic",
			entryTypes: []domain.EntryTypeInfo{
				{Type: "JournalDebit", Description: "Journal Debit"},
				{Type: "", Description: "Journal Credit"},
			},
			wantBad: []domain.EntryType{""},
		},
		{
			name: "missing description",
			entryTypes: []domain.EntryTypeInfo{
				{Type: "JournalDebit", Description: ""},
			},
			wantBad: []domain.EntryType{"JournalDebit"},
		},
		{
			name: "duplicate mnemonic",
			entryTypes: []domain.EntryTypeInfo{
				{Type: "JournalDebit", Description: "Journal Debit"},
				{Type: "JournalDebit", Description: "Journal Debit again"},
			},
			wantBad: []domain.EntryType{"JournalDebit"},
		},
		{
			name: "every bad member is reported, not just the first",
			entryTypes: []domain.EntryTypeInfo{
				{Type: "", Description: "no mnemonic"},
				{Type: "JournalDebit", Description: "Journal Debit"},
				{Type: "JournalCredit", Description: ""},
			},
			wantBad: []domain.EntryType{"", "JournalCredit"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateEntryTypeInfoList(tt.entryTypes)
			if len(tt.wantBad) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, domain.ErrBadEntryTypeInfoList) {
				t.Fatalf("error = %v; want ErrBadEntryTypeInfoList", err)
			}
			fault, ok := errors.AsType[*domain.Fault](err)
			if !ok {
				t.Fatalf("error %v does not carry a fault", err)
			}
			bad := fault.BadValues()
			if len(bad) != len(tt.wantBad) {
				t.Fatalf("BadValues() = %v; want %d values", bad, len(tt.wantBad))
			}
			for i, want := range tt.wantBad {
				if bad[i].Type != want {
					t.Fatalf("BadValues()[%d].Type = %q; want %q", i, bad[i].Type, want)
				}
			}
		})
	}
}

func TestEntryValidate(t *testing.T) {
	tests := []struct {
		name      string
		reporting money.Currency
		mutate    func(*domain.Entry)
		want      cerrs.Error
		wantField string
	}{
		{name: "valid entry", reporting: money.USD},
		{
			name:      "ledger without a reporting currency",
			reporting: "",
			want:      domain.ErrBadCurrencyMnemonic,
		},
		{
			name:      "unset entry date",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.EntryDate = domain.Date{} },
			want:      domain.ErrBadTransaction,
			wantField: "entryDate",
		},
		{
			name:      "empty entry type",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.EntryType = "" },
			want:      domain.ErrBadEntryType,
			wantField: "entryType",
		},
		{
			name:      "empty account id",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.AccountID = "" },
			want:      domain.ErrBadAccountID,
			wantField: "accountId",
		},
		{
			name:      "neither debit nor credit",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.DebitOrCredit = "" },
			want:      domain.ErrBadTransaction,
			wantField: "debitOrCredit",
		},
		{
			name:      "amount in another currency",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.Amount = money.MustNewMinor(100, money.EUR) },
			want:      domain.ErrBadCurrencyMnemonic,
			wantField: "amount",
		},
		{
			name:      "original amount in another currency is fine",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.OriginalAmount = money.MustNewMinor(9000, money.JPY) },
		},
		{
			name:      "original amount with no currency",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.OriginalAmount = money.Money{} },
			want:      domain.ErrBadCurrencyMnemonic,
			wantField: "originalAmount",
		},
		{
			name:      "negative amount",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.Amount = money.MustNewMinor(-100, money.USD) },
			want:      domain.ErrBadTransaction,
			wantField: "amount",
		},
		{
			name:      "negative original amount",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.OriginalAmount = money.MustNewMinor(-100, money.USD) },
			want:      domain.ErrBadTransaction,
			wantField: "originalAmount",
		},
		{
			name:      "zero amount",
			reporting: money.USD,
			mutate:    func(e *domain.Entry) { e.Amount = money.MustNewMinor(0, money.USD) },
		},
		{
			name:      "server allocated identifiers are not required",
			reporting: money.USD,
			mutate: func(e *domain.Entry) {
				e.TransactionID = ""
				e.EntryID = ""
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := newEntry("1000", domain.Debit, 12345)
			if tt.mutate != nil {
				tt.mutate(&entry)
			}
			err := entry.Validate(tt.reporting)
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
