package cli

import "testing"

func TestShortestPath(t *testing.T) {
	cases := []struct {
		path, cwd, want string
	}{
		{"/home/user/proj/sub", "/home/user/proj", "sub"},
		{"/home/user/proj", "/home/user/proj", "."},
	}
	for _, c := range cases {
		if got := shortestPath(c.path, c.cwd); got != c.want {
			t.Errorf("shortestPath(%q, %q) = %q, want %q", c.path, c.cwd, got, c.want)
		}
	}
}

func TestShortSHA(t *testing.T) {
	if got := shortSHA("abcdef0123456789"); got != "abcdef0" {
		t.Errorf("shortSHA(long) = %q, want %q", got, "abcdef0")
	}
	if got := shortSHA("abc"); got != "abc" {
		t.Errorf("shortSHA(short) = %q, want %q", got, "abc")
	}
}
