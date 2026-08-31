// Copyright (c) 2026 Michael D Henderson.

package domain

import "slices"

// Capability is one of the GL services a principal may use in a ledger.
//
// The IDL reaches each service by asking a Profile for an object reference, and
// each of those operations raises PermissionDenied. Handing out object references
// means nothing over HTTP, so the same information becomes a capability set on
// the session profile, and a missing capability reports the same code the IDL
// operation would have raised.
type Capability string

const (
	CapabilityBookKeeping       Capability = "bookKeeping"
	CapabilityRetrieval         Capability = "retrieval"
	CapabilityIntegrity         Capability = "integrity"
	CapabilityLedgerLifecycle   Capability = "ledgerLifecycle"
	CapabilityFacilityLifecycle Capability = "facilityLifecycle"
)

// capabilities lists every capability in the order the contract declares them.
var capabilities = []Capability{
	CapabilityBookKeeping,
	CapabilityRetrieval,
	CapabilityIntegrity,
	CapabilityLedgerLifecycle,
	CapabilityFacilityLifecycle,
}

// Capabilities returns every capability the facility defines.
func Capabilities() []Capability { return slices.Clone(capabilities) }

// Valid reports whether the capability is one the facility defines.
func (c Capability) Valid() bool { return slices.Contains(capabilities, c) }

// ParseCapability converts a contract name to a capability. An unrecognized name
// grants nothing, so it reports PermissionDenied.
func ParseCapability(value string) (Capability, error) {
	capability := Capability(value)
	if !capability.Valid() {
		return "", Faultf(ErrPermissionDenied, "unknown capability").WithBadValue(value)
	}
	return capability, nil
}
