package scout

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func renderEntries(entries []Entry, format string) ([]byte, error) {
	switch format {
	case "list":
		return renderList(entries), nil
	case "skill":
		return renderSkill(entries), nil
	case "json":
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func renderList(entries []Entry) []byte {
	width := 0
	for _, entry := range entries {
		if len(entry.Path) > width {
			width = len(entry.Path)
		}
	}
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%-*s  %s\n", width, entry.Path, entry.Description)
	}
	return []byte(b.String())
}

func renderSkill(entries []Entry) []byte {
	var b strings.Builder
	for i, entry := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\n---\nname: %s\ndescription: %s\n---\n", entry.Path, yamlScalar(entry.Name), yamlScalar(entry.Description))
	}
	return []byte(b.String())
}

func renderManagedList(entries []Entry) []byte {
	width := 0
	for _, entry := range entries {
		if len(entry.Path) > width {
			width = len(entry.Path)
		}
	}
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "- %-*s  %s\n", width, entry.Path, entry.Description)
	}
	return []byte(b.String())
}

func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	plain := true
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' || r == '.' || r == '/' || r == ',' || r == '(' || r == ')') {
			plain = false
			break
		}
	}
	if plain && !strings.HasPrefix(s, " ") && !strings.HasSuffix(s, " ") {
		return s
	}
	return strconv.Quote(s)
}
