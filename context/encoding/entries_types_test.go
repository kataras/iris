package encoding

import (
	"net/netip"
	"testing"
	"time"

	"uuid"
)

const testCanonicalUUID = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

func TestEntryUUID(t *testing.T) {
	want := uuid.MustParse(testCanonicalUUID)

	typed := Entry[any]{Key: "id", Value: want}
	if got := typed.UUID(); got != want {
		t.Fatalf("typed: expected %s, got %s", want, got)
	}
	if got, ok := typed.GetUUID(); !ok || got != want {
		t.Fatalf("typed GetUUID: expected (%s, true), got (%s, %v)", want, got, ok)
	}
	if got := typed.MustUUID(); got != want {
		t.Fatalf("typed MustUUID: expected %s, got %s", want, got)
	}

	// Every spelling uuid.Parse accepts lands on the same value.
	for _, s := range []string{
		testCanonicalUUID,
		"{" + testCanonicalUUID + "}",
		"urn:uuid:" + testCanonicalUUID,
		"A6CE7330-D584-4675-AF9E-2C2D2C40DE07",
		"a6ce7330d5844675af9e2c2d2c40de07",
	} {
		text := Entry[any]{Key: "id", Value: s}
		if got := text.UUID(); got != want {
			t.Fatalf("string %q: expected %s, got %s", s, want, got)
		}
		if _, ok := text.GetUUID(); ok {
			t.Fatalf("string %q: GetUUID must report false for a string value", s)
		}
	}

	for _, v := range []any{"not-a-uuid", "", 42, nil} {
		e := Entry[any]{Key: "id", Value: v}
		if got := e.UUID(); got != (uuid.UUID{}) {
			t.Fatalf("value %#v: expected the zero UUID, got %s", v, got)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustUUID must panic when the value is not a uuid.UUID")
		}
	}()
	e := Entry[any]{Key: "id", Value: testCanonicalUUID}
	e.MustUUID()
}

func TestEntryDuration(t *testing.T) {
	typed := Entry[any]{Key: "ttl", Value: 5 * time.Minute}
	if got := typed.Duration(); got != 5*time.Minute {
		t.Fatalf("typed: expected 5m, got %s", got)
	}
	if got, ok := typed.GetDuration(); !ok || got != 5*time.Minute {
		t.Fatalf("typed GetDuration: expected (5m, true), got (%s, %v)", got, ok)
	}
	if got := typed.MustDuration(); got != 5*time.Minute {
		t.Fatalf("typed MustDuration: expected 5m, got %s", got)
	}

	text := Entry[any]{Key: "ttl", Value: "1h30m"}
	if got := text.Duration(); got != 90*time.Minute {
		t.Fatalf("string: expected 1h30m, got %s", got)
	}
	if _, ok := text.GetDuration(); ok {
		t.Fatal("string: GetDuration must report false for a string value")
	}

	// A bare number carries no unit, so it is not a duration.
	for _, v := range []any{"soon", "", int64(5000), 5, nil} {
		e := Entry[any]{Key: "ttl", Value: v}
		if got := e.Duration(); got != 0 {
			t.Fatalf("value %#v: expected 0, got %s", v, got)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustDuration must panic when the value is not a time.Duration")
		}
	}()
	e := Entry[any]{Key: "ttl", Value: "1h"}
	e.MustDuration()
}

func TestEntryTime(t *testing.T) {
	want := time.Date(2026, time.October, 10, 8, 0, 0, 0, time.UTC)

	typed := Entry[any]{Key: "since", Value: want}
	if got := typed.Time(); !got.Equal(want) {
		t.Fatalf("typed: expected %s, got %s", want, got)
	}
	if got, ok := typed.GetTime(); !ok || !got.Equal(want) {
		t.Fatalf("typed GetTime: expected (%s, true), got (%s, %v)", want, got, ok)
	}
	if got := typed.MustTime(); !got.Equal(want) {
		t.Fatalf("typed MustTime: expected %s, got %s", want, got)
	}

	// RFC 3339, with and without fractional seconds, in any zone.
	for s, exp := range map[string]time.Time{
		"2026-10-10T08:00:00Z":        want,
		"2026-10-10T08:00:00.250Z":    want.Add(250 * time.Millisecond),
		"2026-10-10T10:00:00+02:00":   want,
		"2026-10-10T08:00:00.000001Z": want.Add(time.Microsecond),
	} {
		text := Entry[any]{Key: "since", Value: s}
		if got := text.Time(); !got.Equal(exp) {
			t.Fatalf("string %q: expected %s, got %s", s, exp, got)
		}
		if _, ok := text.GetTime(); ok {
			t.Fatalf("string %q: GetTime must report false for a string value", s)
		}
	}

	// The calendar-day layouts belong to GetDate, not GetTime.
	for _, v := range []any{"2026/10/10", "2026-10-10", "yesterday", "", int64(1760083200), nil} {
		e := Entry[any]{Key: "since", Value: v}
		if got := e.Time(); !got.IsZero() {
			t.Fatalf("value %#v: expected the zero time, got %s", v, got)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustTime must panic when the value is not a time.Time")
		}
	}()
	e := Entry[any]{Key: "since", Value: "2026-10-10T08:00:00Z"}
	e.MustTime()
}

func TestEntryIP(t *testing.T) {
	want := netip.MustParseAddr("10.0.0.1")

	typed := Entry[any]{Key: "addr", Value: want}
	if got := typed.IP(); got != want {
		t.Fatalf("typed: expected %s, got %s", want, got)
	}
	if got, ok := typed.GetIP(); !ok || got != want {
		t.Fatalf("typed GetIP: expected (%s, true), got (%s, %v)", want, got, ok)
	}
	if got := typed.MustIP(); got != want {
		t.Fatalf("typed MustIP: expected %s, got %s", want, got)
	}

	for _, s := range []string{"10.0.0.1", "::1", "2001:db8::1", "fe80::1%eth0"} {
		text := Entry[any]{Key: "addr", Value: s}
		if got, exp := text.IP(), netip.MustParseAddr(s); got != exp {
			t.Fatalf("string %q: expected %s, got %s", s, exp, got)
		}
		if _, ok := text.GetIP(); ok {
			t.Fatalf("string %q: GetIP must report false for a string value", s)
		}
	}

	// An address with a port, a host name, or a number is not an IP address.
	for _, v := range []any{"10.0.0.1:8080", "localhost", "", 42, nil} {
		e := Entry[any]{Key: "addr", Value: v}
		if got := e.IP(); got.IsValid() {
			t.Fatalf("value %#v: expected an invalid (zero) address, got %s", v, got)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustIP must panic when the value is not a netip.Addr")
		}
	}()
	e := Entry[any]{Key: "addr", Value: "10.0.0.1"}
	e.MustIP()
}

// The Entries getters read through the Entry accessors and answer the zero
// value for a missing key, like GetDate does.
func TestEntriesTypedGetters(t *testing.T) {
	id := uuid.MustParse(testCanonicalUUID)
	since := time.Date(2026, time.October, 10, 8, 0, 0, 0, time.UTC)
	addr := netip.MustParseAddr("10.0.0.1")

	entries := Entries[any]{}
	entries.Add("id", id)
	entries.Add("id_text", testCanonicalUUID)
	entries.Add("ttl", "30s")
	entries.Add("since", "2026-10-10T08:00:00Z")
	entries.Add("addr", "10.0.0.1")
	entries.Add("bad", "garbage")

	if got := entries.GetUUID("id"); got != id {
		t.Errorf("GetUUID(id) = %s, expected %s", got, id)
	}
	if got := entries.GetUUID("id_text"); got != id {
		t.Errorf("GetUUID(id_text) = %s, expected %s", got, id)
	}
	if got := entries.GetUUID("bad"); got != (uuid.UUID{}) {
		t.Errorf("GetUUID(bad) = %s, expected the zero UUID", got)
	}
	if got := entries.GetUUID("missing"); got != (uuid.UUID{}) {
		t.Errorf("GetUUID(missing) = %s, expected the zero UUID", got)
	}

	if got := entries.GetDuration("ttl"); got != 30*time.Second {
		t.Errorf("GetDuration(ttl) = %s, expected 30s", got)
	}
	if got := entries.GetDuration("bad"); got != 0 {
		t.Errorf("GetDuration(bad) = %s, expected 0", got)
	}
	if got := entries.GetDuration("missing"); got != 0 {
		t.Errorf("GetDuration(missing) = %s, expected 0", got)
	}

	if got := entries.GetTime("since"); !got.Equal(since) {
		t.Errorf("GetTime(since) = %s, expected %s", got, since)
	}
	if got := entries.GetTime("bad"); !got.IsZero() {
		t.Errorf("GetTime(bad) = %s, expected the zero time", got)
	}
	if got := entries.GetTime("missing"); !got.IsZero() {
		t.Errorf("GetTime(missing) = %s, expected the zero time", got)
	}

	if got := entries.GetIP("addr"); got != addr {
		t.Errorf("GetIP(addr) = %s, expected %s", got, addr)
	}
	if got := entries.GetIP("bad"); got.IsValid() {
		t.Errorf("GetIP(bad) = %s, expected an invalid address", got)
	}
	if got := entries.GetIP("missing"); got.IsValid() {
		t.Errorf("GetIP(missing) = %s, expected an invalid address", got)
	}
}

// ctx.Query(), ctx.Form() and ctx.PostForm() are StringSliceEntries; the same
// getters must reach them through the embedded Entries[string].
func TestStringSliceEntriesTypedGetters(t *testing.T) {
	q := NewStringSliceEntries(map[string][]string{
		"id":    {testCanonicalUUID},
		"ttl":   {"30s"},
		"since": {"2026-10-10T08:00:00Z"},
		"addr":  {"::1"},
	}, ",")

	if got, want := q.GetUUID("id"), uuid.MustParse(testCanonicalUUID); got != want {
		t.Errorf("GetUUID = %s, expected %s", got, want)
	}
	if got := q.GetDuration("ttl"); got != 30*time.Second {
		t.Errorf("GetDuration = %s, expected 30s", got)
	}
	if got, want := q.GetTime("since"), time.Date(2026, time.October, 10, 8, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("GetTime = %s, expected %s", got, want)
	}
	if got, want := q.GetIP("addr"), netip.MustParseAddr("::1"); got != want {
		t.Errorf("GetIP = %s, expected %s", got, want)
	}
	if got := q.GetUUID("missing"); got != (uuid.UUID{}) {
		t.Errorf("GetUUID(missing) = %s, expected the zero UUID", got)
	}
}
