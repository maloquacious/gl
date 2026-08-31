// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"

	"github.com/maloquacious/gl/cerrs"
	"github.com/maloquacious/gl/internal/domain"
	"github.com/maloquacious/gl/money"
)

func TestIdentifierConstructorsAcceptUsableValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		new   func(string) (string, error)
	}{
		{name: "ledger name", value: "general", new: wrapNew(domain.NewLedgerName)},
		{name: "account id", value: "1000-CASH", new: wrapNew(domain.NewAccountID)},
		{name: "transaction id", value: "TXN-000001", new: wrapNew(domain.NewTransactionID)},
		{name: "entry id", value: "ENT-000001", new: wrapNew(domain.NewEntryID)},
		{name: "entry type", value: "JournalDebit", new: wrapNew(domain.NewEntryType)},
		{name: "period id", value: "1998-Q4", new: wrapNew(domain.NewPeriodID)},
		{name: "name with inner spaces", value: "Accounts Receivable", new: wrapNew(domain.NewAccountID)},
		{name: "name with non-ASCII", value: "Kundefordringer-Ø", new: wrapNew(domain.NewLedgerName)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.new(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.value {
				t.Fatalf("value = %q; want %q", got, tt.value)
			}
		})
	}
}

func TestIdentifierConstructorsRejectUnusableValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		new   func(string) (string, error)
		want  cerrs.Error
	}{
		{name: "empty ledger name", value: "", new: wrapNew(domain.NewLedgerName), want: domain.ErrUnknownLedger},
		{name: "empty account id", value: "", new: wrapNew(domain.NewAccountID), want: domain.ErrBadAccountID},
		{name: "blank account id", value: "   ", new: wrapNew(domain.NewAccountID), want: domain.ErrBadAccountID},
		{name: "padded account id", value: " 1000 ", new: wrapNew(domain.NewAccountID), want: domain.ErrBadAccountID},
		{name: "empty transaction id", value: "", new: wrapNew(domain.NewTransactionID), want: domain.ErrBadTransID},
		{name: "empty entry id", value: "", new: wrapNew(domain.NewEntryID), want: domain.ErrBadTransaction},
		{name: "empty entry type", value: "", new: wrapNew(domain.NewEntryType), want: domain.ErrBadEntryType},
		{name: "control character", value: "1000\n", new: wrapNew(domain.NewAccountID), want: domain.ErrBadAccountID},
		{name: "tab in entry type", value: "Journal\tDebit", new: wrapNew(domain.NewEntryType), want: domain.ErrBadEntryType},
		{name: "blank period id", value: " ", new: wrapNew(domain.NewPeriodID), want: domain.ErrBadTransaction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.new(tt.value)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v; want %v", err, tt.want)
			}
			if got != "" {
				t.Fatalf("value = %q; want empty on error", got)
			}
			fault, ok := errors.AsType[*domain.Fault](err)
			if !ok {
				t.Fatalf("error %v does not carry a fault", err)
			}
			if fault.BadValue() != tt.value {
				t.Fatalf("BadValue() = %q; want %q", fault.BadValue(), tt.value)
			}
		})
	}
}

func TestNewPeriodIDAcceptsEmpty(t *testing.T) {
	got, err := domain.NewPeriodID("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("PeriodID = %q; want empty", got)
	}
}

func TestNewCurrency(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    money.Currency
		wantErr bool
	}{
		{name: "registered", value: "USD", want: money.USD},
		{name: "registered with zero scale", value: "JPY", want: money.JPY},
		{name: "unregistered", value: "XYZ", wantErr: true},
		{name: "lowercase", value: "usd", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewCurrency(tt.value)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrBadCurrencyMnemonic) {
					t.Fatalf("error = %v; want ErrBadCurrencyMnemonic", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Currency = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestValidateAccountName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "description", value: "Accounts Receivable"},
		{name: "empty", value: "", wantErr: true},
		{name: "blank", value: "  ", wantErr: true},
		{name: "trailing space", value: "Cash ", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateAccountName(tt.value)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrBadAccountName) {
					t.Fatalf("error = %v; want ErrBadAccountName", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// wrapNew adapts a typed identifier constructor to a common signature so that one
// table can cover every identifier type.
func wrapNew[T ~string](new func(string) (T, error)) func(string) (string, error) {
	return func(value string) (string, error) {
		got, err := new(value)
		return string(got), err
	}
}
