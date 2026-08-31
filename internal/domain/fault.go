// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/maloquacious/gl/cerrs"
)

// Fault is a domain error that carries the members of the specification exception
// it stands for.
//
// Every IDL exception declares an error member; nine declare a bad_value,
// BadEntryTypeInfoList declares bad_values, and BadTransactionsInList declares a
// position_in_list. Those arrive at the contract as APIError.message, badValue,
// badValues, and position, so the value that was rejected has to survive the trip
// from the rule that rejected it. A sentinel alone cannot carry it: an error that
// names the account it refused is worth more than one reporting that some account
// was bad.
//
// A Fault unwraps to its sentinel, so errors.Is against the domain error
// constants works whether or not a fault is in the chain.
type Fault struct {
	code        cerrs.Error
	message     string
	badValue    string
	badValues   []EntryTypeInfo
	field       string
	position    int
	hasPosition bool
}

// Faultf returns a Fault for a domain error with a formatted message.
func Faultf(code cerrs.Error, format string, args ...any) *Fault {
	return &Fault{code: code, message: fmt.Sprintf(format, args...)}
}

// WithBadValue records the value that was rejected, from the exception's
// bad_value member. Set it whenever the exception declares one.
func (f *Fault) WithBadValue(value string) *Fault {
	f.badValue = value
	return f
}

// WithBadValues records the entry types that were rejected, from the bad_values
// member of BadEntryTypeInfoList.
func (f *Fault) WithBadValues(values ...EntryTypeInfo) *Fault {
	f.badValues = slices.Clone(values)
	return f
}

// WithField records the request field path the failure belongs to. The field has
// no counterpart in the IDL; it is a REST convenience for locating a failure
// inside a request body.
func (f *Fault) WithField(field string) *Fault {
	f.field = field
	return f
}

// WithPosition records the zero-based list position of a failure, from the
// position_in_list member of BadTransactionsInList.
func (f *Fault) WithPosition(position int) *Fault {
	f.position = position
	f.hasPosition = true
	return f
}

// Error implements the error interface. The text leads with the specification
// code so that a wrapped fault reads the way the sentinels do.
func (f *Fault) Error() string {
	var sb strings.Builder
	sb.WriteString(string(f.code))
	if f.message != "" {
		sb.WriteString(": ")
		sb.WriteString(f.message)
	}
	if f.badValue != "" {
		sb.WriteString(": ")
		sb.WriteString(f.badValue)
	}
	return sb.String()
}

// Unwrap returns the domain error the fault stands for.
func (f *Fault) Unwrap() error { return f.code }

// Code returns the specification error code.
func (f *Fault) Code() string { return string(f.code) }

// Message returns the human-readable detail, from the exception's error member.
func (f *Fault) Message() string { return f.message }

// BadValue returns the rejected value, or an empty string when the exception
// declares none.
func (f *Fault) BadValue() string { return f.badValue }

// BadValues returns the rejected entry types, or nil.
func (f *Fault) BadValues() []EntryTypeInfo { return slices.Clone(f.badValues) }

// Field returns the request field path, or an empty string.
func (f *Fault) Field() string { return f.field }

// Position returns the zero-based list position and whether one was recorded.
func (f *Fault) Position() (int, bool) { return f.position, f.hasPosition }
