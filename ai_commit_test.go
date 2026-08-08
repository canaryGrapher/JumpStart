package main

import "testing"

func TestCleanSubject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "feat(git): add ai commit messages", "feat(git): add ai commit messages"},
		{"quoted", `"fix(ai): handle empty diff"`, "fix(ai): handle empty diff"},
		{"backticked", "`chore: bump deps`", "chore: bump deps"},
		{"labelled", "Subject: docs: update readme", "docs: update readme"},
		{"body leaked in", "feat: add box\n\n- also did this", "feat: add box"},
		{"padded", "   refactor: split panel   ", "refactor: split panel"},
		{"empty", "   ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := cleanSubject(c.in); got != c.want {
				t.Errorf("cleanSubject(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
