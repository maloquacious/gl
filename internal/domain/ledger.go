// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"strconv"

	"github.com/maloquacious/gl/money"
)

// ChartKind selects the chart of accounts a new ledger starts with. The IDL
// declares it as an unsigned short with three constants; the contract spells the
// same three as names, which is what String and ParseChartKind translate between.
type ChartKind uint16

const (
	// EmptyChart starts the ledger with no accounts.
	EmptyChart ChartKind = 1

	// DefaultChart starts the ledger with the facility's default chart.
	DefaultChart ChartKind = 2

	// ExistingChart copies the chart of another ledger, named separately.
	ExistingChart ChartKind = 3
)

// chartKindNames maps each kind to the name the contract uses.
var chartKindNames = map[ChartKind]string{
	EmptyChart:    "EMPTY_CHART",
	DefaultChart:  "DEFAULT_CHART",
	ExistingChart: "EXISTING_CHART",
}

// Valid reports whether the kind is one the specification declares.
func (k ChartKind) Valid() bool {
	_, ok := chartKindNames[k]
	return ok
}

// String returns the contract name for the kind.
func (k ChartKind) String() string {
	if name, ok := chartKindNames[k]; ok {
		return name
	}
	return "ChartKind(" + strconv.FormatUint(uint64(k), 10) + ")"
}

// RequiresSourceLedger reports whether the kind needs a ledger to copy the chart
// of accounts from.
func (k ChartKind) RequiresSourceLedger() bool { return k == ExistingChart }

// ParseChartKind converts a contract name to a chart kind.
func ParseChartKind(value string) (ChartKind, error) {
	for kind, name := range chartKindNames {
		if name == value {
			return kind, nil
		}
	}
	return 0, Faultf(ErrBadChartKind, "chart kind is not one of EMPTY_CHART, DEFAULT_CHART, or EXISTING_CHART").WithBadValue(value)
}

// Ledger is a ledger and its metadata.
//
// Currency is the reporting currency every entry amount is denominated in. It is
// empty until it is set, because create_ledger_chart_of_accounts takes no
// currency; a ledger without one cannot accept a posting.
type Ledger struct {
	Name      LedgerName
	ChartKind ChartKind
	Currency  money.Currency
}

// IsPostable reports whether the ledger has a reporting currency and can accept
// postings.
func (l Ledger) IsPostable() bool { return l.Currency != "" }
