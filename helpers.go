package appkit

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Trimmed string func to attach to a db scan
// for DB2 char fields coming from db...
// handles null strings

type DB2TrimmedString string

func (t *DB2TrimmedString) Scan(src any) error {
	if src == nil {
		*t = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		*t = DB2TrimmedString(strings.TrimSpace(v))
	case []byte:
		*t = DB2TrimmedString(strings.TrimSpace(string(v)))
	default:
		return fmt.Errorf("DB2TrimmedString: unsupported type %T", src)
	}
	return nil
}

type DB2FloatAsString float64

func (f *DB2FloatAsString) Scan(src any) error {
	if src == nil {
		*f = 0
		return nil
	}
	switch v := src.(type) {
	case float64:
		*f = DB2FloatAsString(v)
	case int64:
		*f = DB2FloatAsString(v)
	case []byte:
		val, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		if err != nil {
			return fmt.Errorf("FloatAsString: cannot parse %s", v)
		}
		*f = DB2FloatAsString(val)
	default:
		return fmt.Errorf("FloatAsString: unsupported type %T", src)
	}
	return nil
}

func (f DB2FloatAsString) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatFloat(float64(f), 'f', 0, 64) + `"`), nil
}

// Trimmed float func to attach to a db scan
// for DB2 decimals coming from db...
type DB2TrimmedFloat64 float64

func (t *DB2TrimmedFloat64) Scan(src any) error {
	if src == nil {
		*t = 0
		return nil
	}
	switch v := src.(type) {
	case float64:
		*t = DB2TrimmedFloat64(v)
	case int64:
		*t = DB2TrimmedFloat64(v)
	case []byte:
		val, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		if err != nil {
			return fmt.Errorf("DB2TrimmedFloat64: cannot parse %s", v)
		}
		*t = DB2TrimmedFloat64(val)
	default:
		return fmt.Errorf("DB2TrimmedFloat64: unsupported type %T", src)
	}
	return nil
}

// Trimmed int func to attach to a db scan
// for DB2 integer coming from db...
type DB2TrimmedInt64 int64

func (t *DB2TrimmedInt64) Scan(src any) error {
	if src == nil {
		*t = 0
		return nil
	}
	switch v := src.(type) {
	case int64:
		*t = DB2TrimmedInt64(v)
	case float64:
		*t = DB2TrimmedInt64(int64(v)) // DB2 might store as a decimal...  truncate to int
	case []byte:
		val, err := strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
		if err != nil {
			return fmt.Errorf("DB2TrimmedInt64: cannot parse %s", v)
		}
		*t = DB2TrimmedInt64(val)
	default:
		return fmt.Errorf("DB2TrimmedInt64: unsupported type %T", src)
	}
	return nil
}

// Date func to attach to a db scan
// for DB2 date fields stored in an integer...
type DB2Date string

func (d *DB2Date) Scan(src any) error {
	if src == nil {
		*d = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		*d = DB2Date(parseDB2Date(strings.TrimSpace(v)))
	case []byte:
		*d = DB2Date(parseDB2Date(strings.TrimSpace(string(v))))
	case int64:
		*d = DB2Date(parseDB2Date(strconv.FormatInt(v, 10)))
	case float64:
		*d = DB2Date(parseDB2Date(strconv.FormatFloat(v, 'f', 0, 64)))
	default:
		return fmt.Errorf("DB2Date: unsupported type %T", src)
	}
	return nil
}

func parseDB2Date(s string) string {
	t, err := time.Parse("20060102", s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02")
}
