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

func TestRenderSkillIncludesPathAndFrontmatter(t *testing.T) {
	got := string(renderSkill([]Entry{{
		Path:        "skills/gh.md",
		Name:        "gh",
		Description: "GitHub CLI for repos.",
	}}))
	want := "skills/gh.md\n---\nname: gh\ndescription: GitHub CLI for repos.\n---\n"
	if got != want {
		t.Fatalf("renderSkill mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderJSONIncludesType(t *testing.T) {
	got, err := renderStdout([]Entry{{
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
