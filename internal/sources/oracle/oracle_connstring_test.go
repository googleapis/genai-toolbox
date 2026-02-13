package oracle

import "testing"

func TestBuildGoOraConnString_EncodesCredentialsAndWallet(t *testing.T) {
	t.Parallel()

	got := buildGoOraConnString("user[client]", "pa:ss@word", "dbhost:1521/XEPDB1", "/tmp/my wallet")

	want := "oracle://user%5Bclient%5D:pa%3Ass%40word@dbhost:1521/XEPDB1?ssl=true&wallet=%2Ftmp%2Fmy+wallet"
	if got != want {
		t.Fatalf("buildGoOraConnString() = %q, want %q", got, want)
	}
}

func TestBuildGoOraConnString_NoWallet(t *testing.T) {
	t.Parallel()

	got := buildGoOraConnString("scott", "tiger", "dbhost:1521/ORCL", "")

	want := "oracle://scott:tiger@dbhost:1521/ORCL"
	if got != want {
		t.Fatalf("buildGoOraConnString() = %q, want %q", got, want)
	}
}

func TestBuildGoOraConnString_DoesNotDoubleEncodePercentEncodedUser(t *testing.T) {
	t.Parallel()

	got := buildGoOraConnString("app_user%5BCLIENT_A%5D", "secret", "dbhost:1521/ORCL", "")

	want := "oracle://app_user%5BCLIENT_A%5D:secret@dbhost:1521/ORCL"
	if got != want {
		t.Fatalf("buildGoOraConnString() = %q, want %q", got, want)
	}
}

func TestBuildGoOraConnString_UsesTrimmedWalletLocation(t *testing.T) {
	t.Parallel()

	got := buildGoOraConnString("scott", "tiger", "dbhost:1521/ORCL", "  /tmp/wallet  ")

	want := "oracle://scott:tiger@dbhost:1521/ORCL?ssl=true&wallet=%2Ftmp%2Fwallet"
	if got != want {
		t.Fatalf("buildGoOraConnString() = %q, want %q", got, want)
	}
}
