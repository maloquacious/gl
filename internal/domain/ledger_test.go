// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"

	"github.com/maloquacious/gl/internal/domain"
	"github.com/maloquacious/gl/money"
)

func TestChartKind(t *testing.T) {
	tests := []struct {
		name        string
		kind        domain.ChartKind
		value       uint16
		text        string
		needsSource bool
	}{
		{name: "empty", kind: domain.EmptyChart, value: 1, text: "EMPTY_CHART"},
		{name: "default", kind: domain.DefaultChart, value: 2, text: "DEFAULT_CHART"},
		{name: "existing", kind: domain.ExistingChart, value: 3, text: "EXISTING_CHART", needsSource: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The IDL numbers these constants, and the contract names them.
			if uint16(tt.kind) != tt.value {
				t.Fatalf("value = %d; want %d", uint16(tt.kind), tt.value)
			}
			if !tt.kind.Valid() {
				t.Fatal("Valid() = false")
			}
			if got := tt.kind.String(); got != tt.text {
				t.Fatalf("String() = %q; want %q", got, tt.text)
			}
			if got := tt.kind.RequiresSourceLedger(); got != tt.needsSource {
				t.Fatalf("RequiresSourceLedger() = %v; want %v", got, tt.needsSource)
			}
			parsed, err := domain.ParseChartKind(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if parsed != tt.kind {
				t.Fatalf("ParseChartKind(%q) = %v; want %v", tt.text, parsed, tt.kind)
			}
		})
	}
}

func TestChartKindRejectsUnknownValues(t *testing.T) {
	var unset domain.ChartKind
	if unset.Valid() {
		t.Fatal("the zero ChartKind is valid; want invalid")
	}
	if got := domain.ChartKind(9).String(); got != "ChartKind(9)" {
		t.Fatalf("String() = %q", got)
	}

	tests := []string{"", "empty_chart", "EMPTY", "2"}
	for _, value := range tests {
		t.Run("parse "+value, func(t *testing.T) {
			got, err := domain.ParseChartKind(value)
			if !errors.Is(err, domain.ErrBadChartKind) {
				t.Fatalf("error = %v; want ErrBadChartKind", err)
			}
			if got != 0 {
				t.Fatalf("ChartKind = %v; want the zero value on error", got)
			}
			fault, ok := errors.AsType[*domain.Fault](err)
			if !ok || fault.BadValue() != value {
				t.Fatalf("fault = %v; want one carrying %q", err, value)
			}
		})
	}
}

func TestLedgerIsPostableOnlyWithAReportingCurrency(t *testing.T) {
	// create_ledger_chart_of_accounts takes no currency, so a new ledger has none
	// and cannot accept a posting until set_ledger_currency supplies one.
	fresh := domain.Ledger{Name: "general", ChartKind: domain.EmptyChart}
	if fresh.IsPostable() {
		t.Fatal("a ledger with no reporting currency reports as postable")
	}

	fresh.Currency = money.USD
	if !fresh.IsPostable() {
		t.Fatal("a ledger with a reporting currency reports as not postable")
	}
}
