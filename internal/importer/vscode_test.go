package importer

import "testing"

func TestUriToPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"file:///home/bal/my%20project", "/home/bal/my project"},
		{"file:///home/bal/foo#bar", "/home/bal/foo#bar"},
		{"/plain/path", "/plain/path"},
		{"file:///home/bal/a%23b", "/home/bal/a#b"},
	}
	for _, c := range cases {
		got := uriToPath(c.in)
		if got != c.want {
			t.Errorf("uriToPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
