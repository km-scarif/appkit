package appkit

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DB2 scan types.
//
// These attach to model struct fields so a DB2/IBM i column lands in Go as the
// shape the API wants: CHAR columns get their padding trimmed, DECIMAL(x,0)
// "numbers" that are really identifiers come out as strings, dates stored as
// YYYYMMDD integers come out as YYYY-MM-DD.
//
// The ODBC driver (github.com/alexbrainman/odbc) decides the concrete Go type
// per column, and the full set it can produce -- see BaseColumn.Value in its
// column.go -- is:
//
//	SQL_C_BIT            -> bool
//	SQL_C_LONG           -> int32      (DB2 INTEGER)
//	SQL_C_SBIGINT        -> int64      (DB2 BIGINT)
//	SQL_C_DOUBLE         -> float64    (DB2 DECIMAL / NUMERIC, incl. DECIMAL(x,0))
//	SQL_C_CHAR           -> []byte
//	SQL_C_WCHAR          -> string
//	SQL_C_TYPE_TIMESTAMP -> time.Time
//
// int32 is the easy one to miss: it is not one of database/sql's driver.Value
// types, but this driver returns it anyway. Every type below accepts the whole
// set that makes sense for it, so a column typed DECIMAL(5,0) in one table and
// INTEGER in another still scans into the same field type.
//
// Precision note: DB2 DECIMAL arrives as float64, which holds integers exactly
// only up to 2^53 (about 9.0e15, 16 digits). A DECIMAL(17,0) or wider used as
// an identifier is already lossy by the time Go sees it -- scan those as
// DB2TrimmedString instead, which keeps the driver's own text.

// ---------------------------------------------------------------------------
// normalizers -- one per target kind, shared by the types below
// ---------------------------------------------------------------------------

// db2Text renders any driver value as a trimmed string.
func db2Text(src any) (string, bool) {
	switch v := src.(type) {
	case string:
		return strings.TrimSpace(v), true
	case []byte:
		return strings.TrimSpace(string(v)), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		// -1 keeps whatever precision the value actually has: an integral
		// DECIMAL(x,0) renders with no decimal point, a fractional one keeps
		// its digits instead of being rounded away.
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(v), true
	case time.Time:
		return v.Format(time.RFC3339), true
	}
	return "", false
}

// db2Number renders any numeric-ish driver value as a float64.
func db2Number(src any) (float64, bool) {
	switch v := src.(type) {
	case float64:
		return v, true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case []byte:
		f, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// DB2TrimmedString -- CHAR/VARCHAR without the padding
// ---------------------------------------------------------------------------

// DB2TrimmedString is a string with the DB2 CHAR padding trimmed off. It also
// accepts numeric columns, so it is the right choice for an identifier that is
// CHAR in one table and DECIMAL(x,0) in another.
type DB2TrimmedString string

// DB2AnyString is an alias for DB2TrimmedString, kept as a self-documenting
// name for fields where the column's DB2 type is unknown or inconsistent and
// you just want the text. Same type, same behavior -- not a second copy.
type DB2AnyString = DB2TrimmedString

func (t *DB2TrimmedString) Scan(src any) error {
	if src == nil {
		*t = ""
		return nil
	}
	s, ok := db2Text(src)
	if !ok {
		return fmt.Errorf("DB2TrimmedString: unsupported type %T", src)
	}
	*t = DB2TrimmedString(s)
	return nil
}

// ---------------------------------------------------------------------------
// DB2FloatAsString -- numeric column that is really an identifier
// ---------------------------------------------------------------------------

// DB2FloatAsString scans a numeric column but marshals to a JSON *string*, for
// the DECIMAL(x,0) columns that are identifiers rather than quantities
// (invoice numbers, customer numbers) and should not be arithmetic in JSON.
type DB2FloatAsString float64

func (f *DB2FloatAsString) Scan(src any) error {
	if src == nil {
		*f = 0
		return nil
	}
	n, ok := db2Number(src)
	if !ok {
		return fmt.Errorf("DB2FloatAsString: unsupported type %T", src)
	}
	*f = DB2FloatAsString(n)
	return nil
}

func (f DB2FloatAsString) MarshalJSON() ([]byte, error) {
	// -1, not 0: an integral value renders identically either way, but 0 would
	// round a fractional value away ("3.5" -> "4") instead of keeping it.
	return []byte(`"` + strconv.FormatFloat(float64(f), 'f', -1, 64) + `"`), nil
}

// ---------------------------------------------------------------------------
// DB2TrimmedFloat64 -- numeric column that really is a number
// ---------------------------------------------------------------------------

// DB2TrimmedFloat64 is a float64 for DB2 decimals. Marshals as a JSON number.
type DB2TrimmedFloat64 float64

func (t *DB2TrimmedFloat64) Scan(src any) error {
	if src == nil {
		*t = 0
		return nil
	}
	n, ok := db2Number(src)
	if !ok {
		return fmt.Errorf("DB2TrimmedFloat64: unsupported type %T", src)
	}
	*t = DB2TrimmedFloat64(n)
	return nil
}

// ---------------------------------------------------------------------------
// DB2TrimmedInt64 -- integer column
// ---------------------------------------------------------------------------

// DB2TrimmedInt64 is an int64 for DB2 integer columns. A decimal source is
// truncated toward zero, not rounded.
type DB2TrimmedInt64 int64

func (t *DB2TrimmedInt64) Scan(src any) error {
	if src == nil {
		*t = 0
		return nil
	}
	// int64 and whole-number text go straight across, so a BIGINT beyond 2^53
	// keeps every digit instead of round-tripping through float64.
	switch v := src.(type) {
	case int64:
		*t = DB2TrimmedInt64(v)
		return nil
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			*t = DB2TrimmedInt64(i)
			return nil
		}
	case []byte:
		if i, err := strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64); err == nil {
			*t = DB2TrimmedInt64(i)
			return nil
		}
	}
	n, ok := db2Number(src)
	if !ok {
		return fmt.Errorf("DB2TrimmedInt64: unsupported type %T", src)
	}
	*t = DB2TrimmedInt64(int64(n)) // DB2 might store as a decimal... truncate to int
	return nil
}

// ---------------------------------------------------------------------------
// DB2Bool -- flag column
// ---------------------------------------------------------------------------

// DB2Bool maps a DB2 flag column to a real bool: numeric 0/1 (however the
// driver types it), or the usual character flags Y/N, T/F, 1/0. Note a plain
// Go bool field cannot do this job -- the stdlib's driver.Bool handles int64
// and string but not int32 or float64, which is what DB2 actually produces for
// a CASE ... THEN 1 ELSE 0.
type DB2Bool bool

func (b *DB2Bool) Scan(src any) error {
	if src == nil {
		*b = false
		return nil
	}
	switch v := src.(type) {
	case bool:
		*b = DB2Bool(v)
		return nil
	case int32:
		*b = DB2Bool(v != 0)
		return nil
	case int64:
		*b = DB2Bool(v != 0)
		return nil
	case float64:
		*b = DB2Bool(v != 0)
		return nil
	case string:
		return b.scanText(v)
	case []byte:
		return b.scanText(string(v))
	}
	return fmt.Errorf("DB2Bool: unsupported type %T", src)
}

func (b *DB2Bool) scanText(s string) error {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "", "0", "N", "F", "FALSE":
		*b = false
	case "1", "Y", "T", "TRUE":
		*b = true
	default:
		return fmt.Errorf("DB2Bool: cannot parse %q", s)
	}
	return nil
}

// ---------------------------------------------------------------------------
// DB2Date -- YYYYMMDD stored as an integer or a char
// ---------------------------------------------------------------------------

// DB2Date renders a DB2 date stored as YYYYMMDD (in any column type) as
// YYYY-MM-DD. A value that doesn't parse is passed through unchanged.
type DB2Date string

func (d *DB2Date) Scan(src any) error {
	if src == nil {
		*d = ""
		return nil
	}
	if v, ok := src.(time.Time); ok {
		*d = DB2Date(v.Format("2006-01-02"))
		return nil
	}
	s, ok := db2Text(src)
	if !ok {
		return fmt.Errorf("DB2Date: unsupported type %T", src)
	}
	*d = DB2Date(parseDB2Date(s))
	return nil
}

func parseDB2Date(s string) string {
	t, err := time.Parse("20060102", s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02")
}
