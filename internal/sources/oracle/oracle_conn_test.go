// Copyright © 2026, Oracle and/or its affiliates.

package oracle

import (
	"net/url"
	"testing"
)

func TestBuildGoOraConnectionString_EncodesProxyUsername(t *testing.T) {
	dsn := buildGoOraConnectionString("app[user]", "p@ss", "dbhost:1521/XEPDB1", "")

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("failed to parse generated DSN: %v", err)
	}

	if got, want := parsed.User.Username(), "app[user]"; got != want {
		t.Fatalf("unexpected username: got %q, want %q", got, want)
	}
	password, ok := parsed.User.Password()
	if !ok {
		t.Fatalf("expected password in generated DSN")
	}
	if got, want := password, "p@ss"; got != want {
		t.Fatalf("unexpected password: got %q, want %q", got, want)
	}
	if got, want := parsed.Host, "dbhost:1521"; got != want {
		t.Fatalf("unexpected host: got %q, want %q", got, want)
	}
	if got, want := parsed.Path, "/XEPDB1"; got != want {
		t.Fatalf("unexpected path: got %q, want %q", got, want)
	}
}

func TestBuildGoOraConnectionString_WithWalletOptions(t *testing.T) {
	dsn := buildGoOraConnectionString("app[user]", "pass", "dbhost:1521/XEPDB1", "/path with space/wallet")

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("failed to parse generated DSN: %v", err)
	}

	if got, want := parsed.Query().Get("ssl"), "true"; got != want {
		t.Fatalf("unexpected ssl query param: got %q, want %q", got, want)
	}
	if got, want := parsed.Query().Get("wallet"), "/path with space/wallet"; got != want {
		t.Fatalf("unexpected wallet query param: got %q, want %q", got, want)
	}
}
