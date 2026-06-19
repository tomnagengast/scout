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

func (fakeSummarizer) SummarizeDir(_ context.Context, path, content string) (string, error) {
	return "Summarizes directory " + path + " with " + strings.TrimSpace(content) + ".", nil
}

type rollupSummarizer struct{}

func (rollupSummarizer) Summarize(_ context.Context, path, _ string, _ bool) (string, error) {
	return "summary for " + path + ".", nil
}

func (rollupSummarizer) SummarizeDir(_ context.Context, path, content string) (string, error) {
	return "directory " + path + " has " + strings.TrimSpace(content) + ".", nil
}

type failingSummarizer struct{}

func (failingSummarizer) Summarize(context.Context, string, string, bool) (string, error) {
	panic("cache was not used")
}

type fileOnlySummarizer struct {
	called bool
}

func (s *fileOnlySummarizer) Summarize(context.Context, string, string, bool) (string, error) {
	s.called = true
	return "unexpected file summary", nil
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
