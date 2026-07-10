package cli

import "testing"

func TestFullyStaged(t *testing.T) {
	cases := map[string]bool{
		"M.": true, "A.": true, "D.": true,
		"MM": false, ".M": false, "??": false, "UU": false,
	}
	for code, want := range cases {
		if got := fullyStaged(code); got != want {
			t.Errorf("fullyStaged(%q) = %v, want %v", code, got, want)
		}
	}
}
