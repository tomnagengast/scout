package scout

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSummarizer struct{}

func (fakeSummarizer) Summarize(_ context.Context, path, content string, truncated bool) (string, error) {
	suffix := ""
	if truncated {
		suffix = " truncated"
	}
	return "Summarizes " + path + " with " + strings.TrimSpace(content) + suffix + ".", nil
}

type failingSummarizer struct{}

func (failingSummarizer) Summarize(context.Context, string, string, bool) (string, error) {
	panic("cache was not used")
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
