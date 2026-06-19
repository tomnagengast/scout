package scout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManagedBlockIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	initial := "# Title\n\nBody.\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	rendered := []byte("docs/a.md  Alpha.\n")
	if err := writeManagedBlock(path, rendered); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeManagedBlock(path, rendered); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("managed write was not idempotent\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if count := strings.Count(string(second), scoutStart); count != 1 {
		t.Fatalf("expected one managed start marker, got %d", count)
	}
}

func TestWriteIndexListUsesManagedMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Title\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Path: "docs/a.md", Description: "Alpha."}}
	if err := writeIndex(path, "list", entries); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "- docs/a.md  Alpha.") {
		t.Fatalf("managed block was not markdown-like:\n%s", got)
	}
}

func TestWriteIndexJSONUsesManagedBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Title\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{
		Type:        entryTypeFile,
		Path:        "docs/a.md",
		Name:        "a",
		Description: "Alpha.",
	}}
	if err := writeIndex(path, "json", entries); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, scoutStart+"\n[\n") || !strings.Contains(text, `"path": "docs/a.md"`) {
		t.Fatalf("managed block did not contain json:\n%s", got)
	}
}

func TestWriteManagedBlockRejectsMalformedMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Title\n\n<!-- scout:start -->\nold\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeManagedBlock(path, []byte("new\n")); err == nil {
		t.Fatal("expected malformed managed block error")
	}
}

func TestWriteIndexSkillReplacesLeadingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gh.md")
	body := "---\nname: old\ndescription: old\n---\n\n# gh\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeIndex(path, "skill", []Entry{{
		Name:        "gh",
		Description: "GitHub CLI for repos.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: gh\ndescription: GitHub CLI for repos.\n---\n\n# gh\n"
	if string(got) != want {
		t.Fatalf("frontmatter mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestWriteSkillFrontmatterReplacesLeadingBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gh.md")
	body := "---\nname: old\ndescription: old\n---\n\n# gh\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeSkillFrontmatter(path, []Entry{{
		Name:        "gh",
		Description: "GitHub CLI for repos.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: gh\ndescription: GitHub CLI for repos.\n---\n\n# gh\n"
	if string(got) != want {
		t.Fatalf("frontmatter mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}
