package appkit

import (
	"encoding/json"
	"testing"
	"time"
)

// The point of these: every DB2 scan type must accept the whole set of Go
// values github.com/alexbrainman/odbc can produce for a column, and the values
// that worked before v0.2.1 must still produce exactly what they did then.

// scanned runs Scan and returns the value as JSON, which is what the API
// actually emits and therefore what must not drift.
func scanned(t *testing.T, dst interface {
	Scan(any) error
}, src any) string {
	t.Helper()
	if err := dst.Scan(src); err != nil {
		t.Fatalf("Scan(%#v): unexpected error: %v", src, err)
	}
	b, err := json.Marshal(dst)
	if err != nil {
		t.Fatalf("Marshal after Scan(%#v): %v", src, err)
	}
	return string(b)
}

func TestDB2TrimmedString(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `""`},
		{"  ACME TIRE   ", `"ACME TIRE"`},
		{[]byte("  001 "), `"001"`},
		{int32(7), `"7"`}, // DB2 INTEGER -- the v0.2.1 fix
		{int64(9007199254740993), `"9007199254740993"`},
		{float64(841541488), `"841541488"`}, // DECIMAL(x,0), no decimal point
		{float64(3.5), `"3.5"`},
		{true, `"true"`},
	}
	for _, c := range cases {
		var got DB2TrimmedString
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2TrimmedString.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}
}

// DB2AnyString is an alias, so it must behave identically -- not merely
// similarly.
func TestDB2AnyStringIsAlias(t *testing.T) {
	var a DB2AnyString
	var b DB2TrimmedString
	if scanned(t, &a, int32(42)) != scanned(t, &b, int32(42)) {
		t.Error("DB2AnyString and DB2TrimmedString disagree; the alias is broken")
	}
}

func TestDB2FloatAsString(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `"0"`},
		{float64(841541488), `"841541488"`}, // unchanged from prec-0 behavior
		{int64(12345), `"12345"`},
		{int32(7), `"7"`}, // the v0.2.1 fix
		{[]byte(" 900 "), `"900"`},
		{"901", `"901"`},
		{float64(3.5), `"3.5"`}, // prec 0 used to round this to "4"
	}
	for _, c := range cases {
		var got DB2FloatAsString
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2FloatAsString.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}
}

func TestDB2TrimmedFloat64(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `0`},
		{float64(12.25), `12.25`},
		{int64(4), `4`},
		{int32(4), `4`}, // the v0.2.1 fix
		{[]byte(" 8.5 "), `8.5`},
	}
	for _, c := range cases {
		var got DB2TrimmedFloat64
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2TrimmedFloat64.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}
}

func TestDB2TrimmedInt64(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `0`},
		{int64(4), `4`},
		{int32(4), `4`},     // the v0.2.1 fix
		{float64(8.9), `8`}, // truncates toward zero, does not round
		{[]byte(" 12 "), `12`},
		{"13", `13`},
		// text and int64 paths must stay exact past 2^53 rather than
		// round-tripping through float64
		{int64(9007199254740993), `9007199254740993`},
		{[]byte("9007199254740993"), `9007199254740993`},
	}
	for _, c := range cases {
		var got DB2TrimmedInt64
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2TrimmedInt64.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}
}

func TestDB2Bool(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `false`},
		{int32(1), `true`}, // what CASE ... THEN 1 ELSE 0 actually returns
		{int32(0), `false`},
		{int64(1), `true`},
		{float64(1), `true`},
		{float64(0), `false`},
		{true, `true`},
		{"Y", `true`},
		{[]byte(" n "), `false`},
		{"TRUE", `true`},
		{"", `false`},
	}
	for _, c := range cases {
		var got DB2Bool
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2Bool.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}

	var bad DB2Bool
	if err := bad.Scan("maybe"); err == nil {
		t.Error("DB2Bool.Scan(\"maybe\"): want error, got nil")
	}
}

func TestDB2Date(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{nil, `""`},
		{"20240115", `"2024-01-15"`},
		{[]byte(" 20240115 "), `"2024-01-15"`},
		{int64(20240115), `"2024-01-15"`},
		{float64(20240115), `"2024-01-15"`},
		{int32(20240115), `"2024-01-15"`}, // the v0.2.1 fix
		{time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC), `"2024-01-15"`},
		{"not a date", `"not a date"`}, // unparseable passes through
	}
	for _, c := range cases {
		var got DB2Date
		if s := scanned(t, &got, c.src); s != c.want {
			t.Errorf("DB2Date.Scan(%#v) = %s, want %s", c.src, s, c.want)
		}
	}
}
