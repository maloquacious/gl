// Copyright (c) 2026 Michael D Henderson.

package domain

import (
	"encoding/json"
	"time"
)

// Date is a date that may be left unset.
//
// The specification pairs a DTime with an is_set flag because retrieval needs a
// bound that means "no bound" rather than a sentinel instant (section 2.1.12).
// The zero Date is the unset one, so a Date field left alone is already the open
// bound. Over the wire an unset Date is an omitted member, not a sentinel.
type Date struct {
	t   time.Time
	set bool
}

// NewDate returns a set Date.
func NewDate(t time.Time) Date { return Date{t: t, set: true} }

// ParseDate parses an RFC 3339 timestamp, the contract's date-time format.
func ParseDate(value string) (Date, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return Date{}, Faultf(ErrBadDate, "date must be an RFC 3339 timestamp").WithBadValue(value)
	}
	return NewDate(t), nil
}

// IsSet reports whether the date carries an instant.
func (d Date) IsSet() bool { return d.set }

// IsZero reports whether the date is unset. It also lets encoding/json omit an
// unset Date under the omitzero tag option.
func (d Date) IsZero() bool { return !d.set }

// Time returns the instant, or the zero time.Time when the date is unset.
func (d Date) Time() time.Time { return d.t }

// Equal reports whether two dates are both unset or describe the same instant.
func (d Date) Equal(other Date) bool {
	if d.set != other.set {
		return false
	}
	return !d.set || d.t.Equal(other.t)
}

// Before reports whether a set date precedes another set date. An unset date has
// no position in time, so it precedes nothing.
func (d Date) Before(other Date) bool {
	return d.set && other.set && d.t.Before(other.t)
}

// After reports whether a set date follows another set date. An unset date has no
// position in time, so it follows nothing.
func (d Date) After(other Date) bool {
	return d.set && other.set && d.t.After(other.t)
}

// String returns the RFC 3339 form, or an empty string when the date is unset.
func (d Date) String() string {
	if !d.set {
		return ""
	}
	return d.t.Format(time.RFC3339)
}

// MarshalJSON encodes a set date as an RFC 3339 string and an unset date as null.
func (d Date) MarshalJSON() ([]byte, error) {
	if !d.set {
		return []byte("null"), nil
	}
	return json.Marshal(d.t.Format(time.RFC3339))
}

// UnmarshalJSON decodes an RFC 3339 string, treating null as an unset date.
func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = Date{}
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseDate(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// DateRange bounds a retrieval by date. Both bounds are inclusive, per
// specification section 2.1.12, and either may be left unset: an unset start
// means "from the beginning of the ledger" and an unset end means "through the
// most recent posting." The zero DateRange is therefore unbounded.
type DateRange struct {
	start Date
	end   Date
}

// NewDateRange returns an inclusive date range, rejecting a start that follows
// the end.
func NewDateRange(start, end Date) (DateRange, error) {
	if start.After(end) {
		return DateRange{}, Faultf(ErrBadDate, "start date %s is after end date %s", start, end).WithBadValue(start.String())
	}
	return DateRange{start: start, end: end}, nil
}

// Start returns the inclusive lower bound.
func (r DateRange) Start() Date { return r.start }

// End returns the inclusive upper bound.
func (r DateRange) End() Date { return r.end }

// IsUnbounded reports whether neither bound is set.
func (r DateRange) IsUnbounded() bool { return !r.start.IsSet() && !r.end.IsSet() }

// Contains reports whether d falls within the range. Both bounds are inclusive,
// and an unset bound does not constrain. An unset date has no position in time
// and so is never contained.
func (r DateRange) Contains(d Date) bool {
	if !d.IsSet() {
		return false
	}
	if r.start.IsSet() && d.Before(r.start) {
		return false
	}
	if r.end.IsSet() && d.After(r.end) {
		return false
	}
	return true
}

// String returns a readable form of the bounds, writing an unset bound as a dash.
func (r DateRange) String() string {
	bound := func(d Date) string {
		if !d.IsSet() {
			return "-"
		}
		return d.String()
	}
	return bound(r.start) + "/" + bound(r.end)
}
