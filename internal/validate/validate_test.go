package validate

import "testing"

func TestBranchNameValid(t *testing.T) {
	valid := []string{
		"main",
		"feature/login",
		"release/v1.2.3",
		"user/manuel/wip",
		"fix-123",
		"a",
	}
	for _, name := range valid {
		if err := BranchName(name); err != nil {
			t.Errorf("BranchName(%q) = %v, want nil", name, err)
		}
	}
}

func TestBranchNameInvalid(t *testing.T) {
	invalid := []string{
		"",           // empty
		"-x",         // leading dash
		"@",          // bare @
		"foo bar",    // space
		"foo~1",      // tilde
		"foo^",       // caret
		"foo:bar",    // colon
		"foo?",       // question mark
		"foo*",       // glob
		"foo[bar",    // open bracket
		"foo\\bar",   // backslash
		"foo..bar",   // double dot
		"foo@{u}",    // reflog syntax
		"foo//bar",   // double slash
		"/foo",       // leading slash
		"foo/",       // trailing slash
		"foo.",       // trailing dot
		".hidden",    // component starts with dot
		"foo/.bar",   // later component starts with dot
		"foo.lock",   // component ends with .lock
		"a/b.lock/c", // interior component ends with .lock
		"foo\tbar",   // control char
	}
	for _, name := range invalid {
		if err := BranchName(name); err == nil {
			t.Errorf("BranchName(%q) = nil, want error", name)
		}
	}
}
