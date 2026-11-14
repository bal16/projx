package util

import "testing"

func TestBaseName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/a/b/c", "c"},
		{"/a/b/c/", "c"},
		{"c", "c"},
		{"/", "/"},
		{"", ""},
	}
	for _, c := range cases {
		got := BaseName(c.in)
		if got != c.want {
			t.Errorf("BaseName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
