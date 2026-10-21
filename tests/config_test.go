package tests

import (
	"net/url"
	"strings"
	"testing"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
)

func TestDatabaseDSNPreservesTimezoneAndCredentials(t *testing.T) {
	t.Setenv("DB_HOST", "db.example.test")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "test user")
	t.Setenv("DB_PASSWORD", "p@ss/word")
	t.Setenv("DB_NAME", "test database")
	t.Setenv("DB_SSLROOTCERT", `C:\certs\root.crt`)

	dsn, err := config.DatabaseDSN()
	if err != nil {
		t.Fatalf("DatabaseDSN() error = %v", err)
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Query().Get("TimeZone") != "Asia/Jakarta" {
		t.Fatalf("TimeZone = %q, want Asia/Jakarta", parsed.Query().Get("TimeZone"))
	}
	if !strings.Contains(parsed.RawQuery, "TimeZone=Asia/Jakarta") {
		t.Fatalf("raw query = %q, want unescaped Asia/Jakarta timezone", parsed.RawQuery)
	}
	if parsed.Query().Get("sslrootcert") != `C:\certs\root.crt` {
		t.Fatalf("sslrootcert = %q, want configured CA path", parsed.Query().Get("sslrootcert"))
	}
	if parsed.User.Username() != "test user" {
		t.Fatalf("username = %q, want escaped username to round-trip", parsed.User.Username())
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss/word" {
		t.Fatal("password did not round-trip through the connection URL")
	}
	if parsed.Path != "/test database" {
		t.Fatalf("database path = %q, want escaped database name to round-trip", parsed.Path)
	}
}

func TestDatabaseDSNRequiresConnectionSettings(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_NAME", "database")

	if _, err := config.DatabaseDSN(); err == nil {
		t.Fatal("DatabaseDSN() error = nil, want missing host error")
	}
}
