// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maloquacious/gl/internal/domain"
)

func TestFaultCarriesExceptionMembers(t *testing.T) {
	badValues := []domain.EntryTypeInfo{{Type: "", Description: "no mnemonic"}}
	fault := domain.Faultf(domain.ErrBadEntryTypeInfoList, "%d entry types are unusable", 1).
		WithBadValue("JournalDebit").
		WithBadValues(badValues...).
		WithField("entryTypes").
		WithPosition(3)

	if got := fault.Code(); got != "BadEntryTypeInfoList" {
		t.Fatalf("Code() = %q; want BadEntryTypeInfoList", got)
	}
	if got := fault.Message(); got != "1 entry types are unusable" {
		t.Fatalf("Message() = %q", got)
	}
	if got := fault.BadValue(); got != "JournalDebit" {
		t.Fatalf("BadValue() = %q; want JournalDebit", got)
	}
	if got := fault.BadValues(); len(got) != 1 || got[0].Description != "no mnemonic" {
		t.Fatalf("BadValues() = %v", got)
	}
	if got := fault.Field(); got != "entryTypes" {
		t.Fatalf("Field() = %q; want entryTypes", got)
	}
	if position, ok := fault.Position(); !ok || position != 3 {
		t.Fatalf("Position() = %d, %v; want 3, true", position, ok)
	}
}

func TestFaultWithoutOptionalMembers(t *testing.T) {
	fault := domain.Faultf(domain.ErrPermissionDenied, "principal lacks bookKeeping")

	if got := fault.BadValue(); got != "" {
		t.Fatalf("BadValue() = %q; want empty", got)
	}
	if got := fault.BadValues(); got != nil {
		t.Fatalf("BadValues() = %v; want nil", got)
	}
	if position, ok := fault.Position(); ok {
		t.Fatalf("Position() = %d, true; want no position", position)
	}
	if got := fault.Error(); got != "PermissionDenied: principal lacks bookKeeping" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestFaultMatchesItsSentinel(t *testing.T) {
	fault := domain.Faultf(domain.ErrBadAccountID, "no such account").WithBadValue("4000")

	if !errors.Is(fault, domain.ErrBadAccountID) {
		t.Fatal("errors.Is(fault, ErrBadAccountID) = false; want true")
	}
	if errors.Is(fault, domain.ErrBadTransID) {
		t.Fatal("errors.Is(fault, ErrBadTransID) = true; want false")
	}
	if !errors.Is(fmt.Errorf("get account: %w", fault), domain.ErrBadAccountID) {
		t.Fatal("wrapped fault did not match its sentinel")
	}
	if got, ok := errors.AsType[*domain.Fault](fmt.Errorf("get account: %w", fault)); !ok || got.BadValue() != "4000" {
		t.Fatalf("errors.AsType recovered %v, %v", got, ok)
	}
	if got := fault.Error(); got != "BadAccountId: no such account: 4000" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestFaultBadValuesAreCopied(t *testing.T) {
	values := []domain.EntryTypeInfo{{Type: "JournalDebit", Description: "debit"}}
	fault := domain.Faultf(domain.ErrBadEntryTypeInfoList, "unusable").WithBadValues(values...)

	values[0].Type = "mutated"
	if got := fault.BadValues(); got[0].Type != "JournalDebit" {
		t.Fatalf("BadValues()[0].Type = %q; want JournalDebit", got[0].Type)
	}

	returned := fault.BadValues()
	returned[0].Type = "mutated"
	if got := fault.BadValues(); got[0].Type != "JournalDebit" {
		t.Fatalf("BadValues() leaked its backing array")
	}
}
