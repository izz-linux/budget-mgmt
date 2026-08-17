package config

import (
	"net/url"
	"testing"
)

// TestDatabaseURL_EscapesSpecialCharacters pins the DSN-building bug: the URL
// was assembled with fmt.Sprintf, so a password containing a URL-significant
// character silently corrupted the connection string (everything after an "@"
// was read as the host, a "?" started the query string, and so on).
func TestDatabaseURL_EscapesSpecialCharacters(t *testing.T) {
	passwords := []string{
		"p@ssw0rd",
		"a/b",
		"what?now",
		"colon:pass",
		"hash#tag",
		"sp ace",
		"plain",
	}

	for _, pw := range passwords {
		cfg := &Config{
			DBHost:     "db.internal",
			DBPort:     5432,
			DBName:     "budgetapp",
			DBUser:     "budget",
			DBPassword: pw,
			DBSSLMode:  "require",
		}

		raw := cfg.DatabaseURL()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("password %q produced an unparseable DSN %q: %v", pw, raw, err)
		}

		if u.Scheme != "postgres" {
			t.Errorf("password %q: expected scheme postgres, got %q", pw, u.Scheme)
		}
		if u.Host != "db.internal:5432" {
			t.Errorf("password %q: expected host db.internal:5432, got %q", pw, u.Host)
		}
		if u.Path != "/budgetapp" {
			t.Errorf("password %q: expected path /budgetapp, got %q", pw, u.Path)
		}
		if u.User.Username() != "budget" {
			t.Errorf("password %q: expected user budget, got %q", pw, u.User.Username())
		}
		gotPW, set := u.User.Password()
		if !set || gotPW != pw {
			t.Errorf("password %q did not round-trip; got %q (set=%v)", pw, gotPW, set)
		}
		if got := u.Query().Get("sslmode"); got != "require" {
			t.Errorf("password %q: expected sslmode=require, got %q", pw, got)
		}
	}
}

// TestDatabaseURL_IPv6Host confirms the host is bracketed correctly, which
// net.JoinHostPort handles and naive concatenation does not.
func TestDatabaseURL_IPv6Host(t *testing.T) {
	cfg := &Config{
		DBHost:     "::1",
		DBPort:     5432,
		DBName:     "budgetapp",
		DBUser:     "budget",
		DBPassword: "secret",
		DBSSLMode:  "disable",
	}

	u, err := url.Parse(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("unparseable DSN: %v", err)
	}
	if u.Hostname() != "::1" {
		t.Errorf("expected hostname ::1, got %q", u.Hostname())
	}
	if u.Port() != "5432" {
		t.Errorf("expected port 5432, got %q", u.Port())
	}
}
