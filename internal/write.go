package scout

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	scoutStart = "<!-- scout:start -->"
	scoutEnd   = "<!-- scout:end -->"
)

func writeIndex(path, format string, rendered []byte, entries []Entry) error {
	if format == "skill" {
		return writeSkillFrontmatter(path, entries)
	}
	if format == "list" {
		rendered = renderManagedList(entries)
	}
	return writeManagedBlock(path, rendered)
}

func writeManagedBlock(path string, rendered []byte) error {
	body, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	block := []byte(scoutStart + "\n" + strings.TrimRight(string(rendered), "\n") + "\n" + scoutEnd + "\n")
	if len(body) == 0 {
		return os.WriteFile(path, block, 0o644)
	}
	start := bytes.Index(body, []byte(scoutStart))
	end := bytes.Index(body, []byte(scoutEnd))
	if (start >= 0) != (end >= 0) || (start >= 0 && end < start) {
		return fmt.Errorf("%s: malformed scout managed block", path)
	}
	var out []byte
	if start >= 0 && end >= start {
		end += len(scoutEnd)
		out = append(out, body[:start]...)
		out = append(out, block...)
		out = append(out, bytes.TrimLeft(body[end:], "\r\n")...)
	} else {
		out = append(bytes.TrimRight(body, "\r\n"), '\n', '\n')
		out = append(out, block...)
	}
	return os.WriteFile(path, out, 0o644)
}

func writeSkillFrontmatter(path string, entries []Entry) error {
	if len(entries) != 1 {
		return errors.New("--format skill --write requires exactly one input file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n", yamlScalar(entries[0].Name), yamlScalar(entries[0].Description))
	body = stripLeadingFrontmatter(body)
	out := append([]byte(frontmatter), body...)
	return os.WriteFile(path, out, 0o644)
}

var frontmatterRE = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n{0,2}`)

func stripLeadingFrontmatter(body []byte) []byte {
	return frontmatterRE.ReplaceAll(body, nil)
}
