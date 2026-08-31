// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"

	"github.com/maloquacious/gl/internal/domain"
)

func TestCapabilities(t *testing.T) {
	// One capability per GL service the IDL reaches through a Profile.
	want := []domain.Capability{
		domain.CapabilityBookKeeping,
		domain.CapabilityRetrieval,
		domain.CapabilityIntegrity,
		domain.CapabilityLedgerLifecycle,
		domain.CapabilityFacilityLifecycle,
	}
	got := domain.Capabilities()
	if len(got) != len(want) {
		t.Fatalf("Capabilities() = %v; want %d capabilities", got, len(want))
	}
	for i, capability := range want {
		if got[i] != capability {
			t.Fatalf("Capabilities()[%d] = %q; want %q", i, got[i], capability)
		}
		if !capability.Valid() {
			t.Fatalf("%q.Valid() = false", capability)
		}
		parsed, err := domain.ParseCapability(string(capability))
		if err != nil {
			t.Fatal(err)
		}
		if parsed != capability {
			t.Fatalf("ParseCapability(%q) = %q", capability, parsed)
		}
	}

	got[0] = "mutated"
	if domain.Capabilities()[0] != domain.CapabilityBookKeeping {
		t.Fatal("Capabilities() leaked its backing array")
	}
}

func TestParseCapabilityRejectsUnknownNames(t *testing.T) {
	for _, value := range []string{"", "bookkeeping", "admin"} {
		t.Run("parse "+value, func(t *testing.T) {
			got, err := domain.ParseCapability(value)
			if !errors.Is(err, domain.ErrPermissionDenied) {
				t.Fatalf("error = %v; want ErrPermissionDenied", err)
			}
			if got != "" {
				t.Fatalf("Capability = %q; want empty on error", got)
			}
		})
	}
}
