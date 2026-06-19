package scout

import (
	"strings"
	"testing"
)

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

func TestRenderJSONIncludesType(t *testing.T) {
	got, err := renderEntries([]Entry{{
		Type:        entryTypeDir,
		Path:        "docs",
		Name:        "docs",
		Description: "Documents Scout.",
	}}, "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"type": "dir"`) {
		t.Fatalf("json missing type:\n%s", got)
	}
}
