package cli

import "testing"

func TestShortSHA(t *testing.T) {
	if got := shortSHA("abcdef0123456789"); got != "abcdef0" {
		t.Errorf("shortSHA(long) = %q, want %q", got, "abcdef0")
	}
	if got := shortSHA("abc"); got != "abc" {
		t.Errorf("shortSHA(short) = %q, want %q", got, "abc")
	}
}
