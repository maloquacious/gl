// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maloquacious/gl/cerrs"
	"github.com/maloquacious/gl/internal/domain"
)

// errorCodes is the ErrorCode enum from api/openapi.yaml, paired with the domain
// error that has to report it. The contract and this package drift apart quietly
// otherwise: a handler can only report a code some error carries.
var errorCodes = map[string]cerrs.Error{
	"BadDate":               domain.ErrBadDate,
	"BadChartKind":          domain.ErrBadChartKind,
	"BadTransaction":        domain.ErrBadTransaction,
	"BadTransactionsInList": domain.ErrBadTransactionsInList,
	"BadEntryType":          domain.ErrBadEntryType,
	"BadEntryTypeInfoList":  domain.ErrBadEntryTypeInfoList,
	"BadCurrencyMnemonic":   domain.ErrBadCurrencyMnemonic,
	"BadAccountId":          domain.ErrBadAccountID,
	"BadTransId":            domain.ErrBadTransID,
	"CannotRemove":          domain.ErrCannotRemove,
	"PermissionDenied":      domain.ErrPermissionDenied,
	"UnknownLedger":         domain.ErrUnknownLedger,
	"BadIntegritySelection": domain.ErrBadIntegritySelection,
	"BadAccountName":        domain.ErrBadAccountName,
}

func TestErrorCodesMatchTheContract(t *testing.T) {
	for code, err := range errorCodes {
		t.Run(code, func(t *testing.T) {
			if got := err.Error(); got != code {
				t.Fatalf("Error() = %q; want %q", got, code)
			}
			if got := domain.Code(err); got != code {
				t.Fatalf("Code() = %q; want %q", got, code)
			}
			wrapped := fmt.Errorf("posting failed: %w", err)
			if got := domain.Code(wrapped); got != code {
				t.Fatalf("Code(wrapped) = %q; want %q", got, code)
			}
			if got := domain.Code(domain.Faultf(err, "detail")); got != code {
				t.Fatalf("Code(fault) = %q; want %q", got, code)
			}
		})
	}
}

func TestCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil error", err: nil, want: ""},
		{name: "foreign error", err: errors.New("disk full"), want: ""},
		{name: "sentinel", err: domain.ErrCannotRemove, want: "CannotRemove"},
		{name: "wrapped sentinel", err: fmt.Errorf("%w: account 100", domain.ErrCannotRemove), want: "CannotRemove"},
		{name: "fault", err: domain.Faultf(domain.ErrBadTransID, "no such transaction"), want: "BadTransId"},
		{name: "wrapped fault", err: fmt.Errorf("retrieval: %w", domain.Faultf(domain.ErrBadTransID, "no such transaction")), want: "BadTransId"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.Code(tt.err); got != tt.want {
				t.Fatalf("Code() = %q; want %q", got, tt.want)
			}
		})
	}
}
