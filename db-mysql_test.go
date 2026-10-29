package appkit

import (
	"strings"
	"testing"
)

func TestRedactDSN(t *testing.T) {
	got := redactDSN("someuser:s3cret@tcp(db.example.com:3306)/somedb?parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27")
	if strings.Contains(got, "s3cret") || !strings.HasPrefix(got, "someuser:****@tcp(db.example.com:3306)/somedb?") {
		t.Fatalf("password not redacted: %s", got)
	}
	if got := redactDSN("not a dsn"); got != "(unparseable DSN)" {
		t.Fatalf("unparseable DSN not suppressed: %s", got)
	}
}
