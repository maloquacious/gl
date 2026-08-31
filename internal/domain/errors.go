// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"errors"

	"github.com/maloquacious/gl/cerrs"
)

// The domain errors mirror the OpenAPI ErrorCode enum, which in turn mirrors the
// exception names declared by the LEDG IDL. Each constant's text is the
// specification error code, so an error arriving at the API boundary already
// carries the code the contract has to report; Code recovers it.
//
// The HTTP status is not decided here. The same code appears under more than one
// status - a missing account is 404 BadAccountId and a malformed one is 400
// BadAccountId - and choosing between them is the handler's job.
const (
	ErrBadAccountID          = cerrs.Error("BadAccountId")
	ErrBadAccountName        = cerrs.Error("BadAccountName")
	ErrBadChartKind          = cerrs.Error("BadChartKind")
	ErrBadCurrencyMnemonic   = cerrs.Error("BadCurrencyMnemonic")
	ErrBadDate               = cerrs.Error("BadDate")
	ErrBadEntryType          = cerrs.Error("BadEntryType")
	ErrBadEntryTypeInfoList  = cerrs.Error("BadEntryTypeInfoList")
	ErrBadIntegritySelection = cerrs.Error("BadIntegritySelection")
	ErrBadTransaction        = cerrs.Error("BadTransaction")
	ErrBadTransactionsInList = cerrs.Error("BadTransactionsInList")
	ErrBadTransID            = cerrs.Error("BadTransId")
	ErrCannotRemove          = cerrs.Error("CannotRemove")
	ErrPermissionDenied      = cerrs.Error("PermissionDenied")
	ErrUnknownLedger         = cerrs.Error("UnknownLedger")
)

// errorCodes lists every domain error, so that Code can name one that was wrapped
// with fmt.Errorf rather than carried by a Fault. Keep it complete: an error
// missing from this list reports no code at all at the API boundary.
var errorCodes = []cerrs.Error{
	ErrBadAccountID,
	ErrBadAccountName,
	ErrBadChartKind,
	ErrBadCurrencyMnemonic,
	ErrBadDate,
	ErrBadEntryType,
	ErrBadEntryTypeInfoList,
	ErrBadIntegritySelection,
	ErrBadTransaction,
	ErrBadTransactionsInList,
	ErrBadTransID,
	ErrCannotRemove,
	ErrPermissionDenied,
	ErrUnknownLedger,
}

// Code returns the specification error code carried by err, or an empty string
// when err is not a domain error. An empty result means the failure has no
// specification meaning and belongs in a 500, not that it can be ignored.
func Code(err error) string {
	if err == nil {
		return ""
	}
	if fault, ok := errors.AsType[*Fault](err); ok {
		return fault.Code()
	}
	for _, code := range errorCodes {
		if errors.Is(err, code) {
			return string(code)
		}
	}
	return ""
}
