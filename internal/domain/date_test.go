// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/maloquacious/gl/internal/domain"
)

func mustParseDate(t *testing.T, value string) domain.Date {
	t.Helper()
	d, err := domain.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDateSetAndUnset(t *testing.T) {
	var unset domain.Date
	if unset.IsSet() {
		t.Fatal("the zero Date is set; want unset")
	}
	if !unset.IsZero() {
		t.Fatal("IsZero() = false for the zero Date")
	}
	if got := unset.String(); got != "" {
		t.Fatalf("String() = %q; want empty", got)
	}
	if !unset.Time().IsZero() {
		t.Fatalf("Time() = %v; want the zero time", unset.Time())
	}

	set := domain.NewDate(time.Date(1998, time.December, 21, 9, 30, 0, 0, time.UTC))
	if !set.IsSet() {
		t.Fatal("IsSet() = false for a date built from an instant")
	}
	if set.IsZero() {
		t.Fatal("IsZero() = true for a set date")
	}
	if got := set.String(); got != "1998-12-21T09:30:00Z" {
		t.Fatalf("String() = %q", got)
	}
}

func TestDateZeroInstantIsStillSet(t *testing.T) {
	// A Date built from the zero instant is set; only an untouched Date is unset.
	set := domain.NewDate(time.Time{})
	if !set.IsSet() || set.IsZero() {
		t.Fatal("a Date built from the zero instant reports as unset")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "utc", value: "1998-12-21T09:30:00Z"},
		{name: "offset", value: "1998-12-21T09:30:00-05:00"},
		{name: "date only", value: "1998-12-21", wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "nonsense", value: "yesterday", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseDate(tt.value)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrBadDate) {
					t.Fatalf("error = %v; want ErrBadDate", err)
				}
				if got.IsSet() {
					t.Fatal("a rejected date came back set")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsSet() {
				t.Fatal("a parsed date came back unset")
			}
		})
	}
}

func TestDateEqualComparesInstants(t *testing.T) {
	utc := mustParseDate(t, "1998-12-21T09:30:00Z")
	sameInstant := mustParseDate(t, "1998-12-21T04:30:00-05:00")
	if !utc.Equal(sameInstant) {
		t.Fatal("dates naming the same instant in different zones are not equal")
	}
	var unset domain.Date
	if utc.Equal(unset) || unset.Equal(utc) {
		t.Fatal("an unset date equals a set one")
	}
	if !unset.Equal(domain.Date{}) {
		t.Fatal("two unset dates are not equal")
	}
}

func TestDateOrderingIgnoresUnsetDates(t *testing.T) {
	early := mustParseDate(t, "1998-01-01T00:00:00Z")
	late := mustParseDate(t, "1998-12-31T00:00:00Z")
	var unset domain.Date

	if !early.Before(late) || !late.After(early) {
		t.Fatal("set dates do not order")
	}
	if early.Before(unset) || early.After(unset) || unset.Before(early) || unset.After(early) {
		t.Fatal("an unset date takes a position in time")
	}
}

func TestDateJSON(t *testing.T) {
	type payload struct {
		VoucherDate domain.Date `json:"voucherDate,omitzero"`
	}

	encoded, err := json.Marshal(payload{VoucherDate: mustParseDate(t, "1998-12-21T09:30:00Z")})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"voucherDate":"1998-12-21T09:30:00Z"}` {
		t.Fatalf("Marshal() = %s", encoded)
	}

	// An unset bound is an omitted member, not a sentinel value.
	encoded, err = json.Marshal(payload{})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{}` {
		t.Fatalf("Marshal() of an unset date = %s; want {}", encoded)
	}

	// An unset Date that is marshalled anyway encodes as null rather than as a
	// sentinel instant.
	encoded, err = json.Marshal(domain.Date{})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "null" {
		t.Fatalf("Marshal() of an unset date = %s; want null", encoded)
	}

	var decoded payload
	if err := json.Unmarshal([]byte(`{"voucherDate":"1998-12-21T09:30:00Z"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.VoucherDate.Equal(mustParseDate(t, "1998-12-21T09:30:00Z")) {
		t.Fatalf("Unmarshal() = %v", decoded.VoucherDate)
	}

	decoded = payload{VoucherDate: mustParseDate(t, "1998-12-21T09:30:00Z")}
	if err := json.Unmarshal([]byte(`{"voucherDate":null}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.VoucherDate.IsSet() {
		t.Fatal("null decoded to a set date")
	}

	if err := json.Unmarshal([]byte(`{"voucherDate":"yesterday"}`), &decoded); !errors.Is(err, domain.ErrBadDate) {
		t.Fatalf("Unmarshal error = %v; want ErrBadDate", err)
	}
}

func TestNewDateRangeRejectsReversedBounds(t *testing.T) {
	start := mustParseDate(t, "1998-12-31T00:00:00Z")
	end := mustParseDate(t, "1998-01-01T00:00:00Z")

	_, err := domain.NewDateRange(start, end)
	if !errors.Is(err, domain.ErrBadDate) {
		t.Fatalf("error = %v; want ErrBadDate", err)
	}
	fault, ok := errors.AsType[*domain.Fault](err)
	if !ok {
		t.Fatalf("error %v does not carry a fault", err)
	}
	if fault.BadValue() != start.String() {
		t.Fatalf("BadValue() = %q; want %q", fault.BadValue(), start)
	}
}

func TestDateRangeContainsBothBounds(t *testing.T) {
	start := mustParseDate(t, "1998-01-01T00:00:00Z")
	end := mustParseDate(t, "1998-12-31T00:00:00Z")

	tests := []struct {
		name  string
		start domain.Date
		end   domain.Date
		date  domain.Date
		want  bool
	}{
		{name: "inside", start: start, end: end, date: mustParseDate(t, "1998-06-15T12:00:00Z"), want: true},
		{name: "on the start bound", start: start, end: end, date: start, want: true},
		{name: "on the end bound", start: start, end: end, date: end, want: true},
		{name: "one second before the start", start: start, end: end, date: mustParseDate(t, "1997-12-31T23:59:59Z")},
		{name: "one second after the end", start: start, end: end, date: mustParseDate(t, "1998-12-31T00:00:01Z")},
		{name: "unset start bounds only above", end: end, date: mustParseDate(t, "1066-10-14T00:00:00Z"), want: true},
		{name: "unset end bounds only below", start: start, date: mustParseDate(t, "2999-01-01T00:00:00Z"), want: true},
		{name: "unbounded contains any set date", date: mustParseDate(t, "1998-06-15T12:00:00Z"), want: true},
		{name: "unbounded does not contain an unset date"},
		{name: "an unset date is never contained", start: start, end: end},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := domain.NewDateRange(tt.start, tt.end)
			if err != nil {
				t.Fatal(err)
			}
			if got := r.Contains(tt.date); got != tt.want {
				t.Fatalf("Contains(%v) = %v; want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestDateRangeZeroValueIsUnbounded(t *testing.T) {
	var r domain.DateRange
	if !r.IsUnbounded() {
		t.Fatal("the zero DateRange is bounded; want unbounded")
	}
	if r.Start().IsSet() || r.End().IsSet() {
		t.Fatal("the zero DateRange carries a bound")
	}

	bounded, err := domain.NewDateRange(mustParseDate(t, "1998-01-01T00:00:00Z"), domain.Date{})
	if err != nil {
		t.Fatal(err)
	}
	if bounded.IsUnbounded() {
		t.Fatal("a range with a start bound reports as unbounded")
	}
	if got := bounded.String(); got != "1998-01-01T00:00:00Z/-" {
		t.Fatalf("String() = %q", got)
	}
}
