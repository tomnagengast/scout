package scout

import "testing"

func TestRenderListPadsPaths(t *testing.T) {
	got := string(renderList([]Entry{
		{Path: "a.md", Description: "Alpha."},
		{Path: "docs/b.md", Description: "Bravo."},
	}))
	want := "a.md       Alpha.\ndocs/b.md  Bravo.\n"
	if got != want {
		t.Fatalf("renderList mismatch\nwant: %q\n got: %q", want, got)
	}
}
