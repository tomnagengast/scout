package main

import (
	"context"
	"strings"
	"testing"
)

func TestExpandProviderArgs(t *testing.T) {
	args, opts := expandProviderArgs(
		[]string{"exec", "{model_args}", "{prompt}", "{output}"},
		"--model",
		"fast-model",
		"/tmp/out",
		"prompt text",
	)
	got := strings.Join(args, "\x00")
	want := strings.Join([]string{"exec", "--model", "fast-model", "prompt text", "/tmp/out"}, "\x00")
	if got != want {
		t.Fatalf("args mismatch\nwant: %q\n got: %q", want, got)
	}
	if opts.UseStdin {
		t.Fatal("prompt placeholder should disable stdin")
	}
	if !opts.UseOutputFile {
		t.Fatal("output placeholder should enable output file")
	}
}

func TestCLISummarizerReadsStdout(t *testing.T) {
	summarizer := &cliSummarizer{
		provider: "test",
		config: CLIProviderConfig{
			Command: "sh",
			Args:    []string{"-c", "cat >/dev/null; printf 'Stdout summary.'"},
		},
	}
	got, err := summarizer.Summarize(context.Background(), "note.md", "hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "Stdout summary." {
		t.Fatalf("summary mismatch: %q", got)
	}
}

func TestCLISummarizerReadsOutputFile(t *testing.T) {
	summarizer := &cliSummarizer{
		provider: "test",
		config: CLIProviderConfig{
			Command: "sh",
			Args:    []string{"-c", "cat >/dev/null; printf 'File summary.' > \"$1\"", "sh", "{output}"},
		},
	}
	got, err := summarizer.Summarize(context.Background(), "note.md", "hello", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "File summary." {
		t.Fatalf("summary mismatch: %q", got)
	}
}
