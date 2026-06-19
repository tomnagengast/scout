package scout

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type cliSummarizer struct {
	provider string
	model    string
	config   CLIProviderConfig
}

var defaultCLIProviders = map[string]CLIProviderConfig{
	"codex": {
		Command: "codex",
		Args: []string{
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"--ignore-user-config",
			"--ignore-rules",
			"--sandbox", "read-only",
			"--color", "never",
			"-c", `model_reasoning_effort="none"`,
			"--output-last-message", "{output}",
			"{model_args}",
			"-",
		},
		ModelArg: "--model",
	},
	"claude": {
		Command: "claude",
		Args: []string{
			"-p",
			"--output-format", "text",
			"--permission-mode", "plan",
			"--max-turns", "1",
			"{model_args}",
			"{prompt}",
		},
		ModelArg: "--model",
	},
}

func newSummarizer(cfg Config) (Summarizer, error) {
	providerConfig, err := providerConfigFor(cfg, cfg.Provider)
	if err != nil {
		return nil, err
	}
	return &cliSummarizer{
		provider: cfg.Provider,
		model:    cfg.Model,
		config:   providerConfig,
	}, nil
}

func providerConfigFor(cfg Config, provider string) (CLIProviderConfig, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = defaultProvider
	}

	base, ok := defaultCLIProviders[provider]
	if configured, hasConfig := cfg.Providers[provider]; hasConfig {
		if !ok {
			base = CLIProviderConfig{}
		}
		if configured.Command != "" {
			base.Command = configured.Command
		}
		if configured.Args != nil {
			base.Args = configured.Args
		}
		if configured.ModelArg != "" {
			base.ModelArg = configured.ModelArg
		}
		ok = true
	}
	if !ok || base.Command == "" {
		return CLIProviderConfig{}, fmt.Errorf("unknown summarizer provider %q", provider)
	}
	return base, nil
}

func (s *cliSummarizer) Summarize(ctx context.Context, path, content string, truncated bool) (string, error) {
	prompt := summaryPrompt(path, content, truncated)
	outputPath, cleanup, err := tempOutputPath()
	if err != nil {
		return "", err
	}
	defer cleanup()

	args, opts := expandProviderArgs(s.config.Args, s.config.ModelArg, s.model, outputPath, prompt)
	cmd := exec.CommandContext(ctx, s.config.Command, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "TERM=dumb")
	if opts.UseStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s provider: %w: %s", s.provider, err, compactCommandOutput(stdout.String(), stderr.String()))
	}

	if opts.UseOutputFile {
		data, err := os.ReadFile(outputPath)
		if err != nil {
			return "", fmt.Errorf("%s provider output: %w", s.provider, err)
		}
		return string(data), nil
	}
	return stdout.String(), nil
}

func summaryPrompt(path, content string, truncated bool) string {
	truncatedNote := ""
	if truncated {
		truncatedNote = "\nThe file was truncated; summarize only the visible head without guessing hidden content."
	}
	return fmt.Sprintf(`Write one dense, action-oriented file description for an AI agent building a progressive-disclosure map.

Describe what the file is for and its boundaries. Mention explicit exclusions only when the file makes them clear.
Return exactly one sentence, no markdown, no path prefix, no quotes.

Path: %s%s

File content:
%s`, path, truncatedNote, content)
}

type expandedArgsOptions struct {
	UseStdin      bool
	UseOutputFile bool
}

func expandProviderArgs(args []string, modelArg, model, outputPath, prompt string) ([]string, expandedArgsOptions) {
	opts := expandedArgsOptions{UseStdin: true}
	var expanded []string
	for _, arg := range args {
		switch arg {
		case "{model_args}":
			if model != "" && modelArg != "" {
				expanded = append(expanded, modelArg, model)
			}
		case "{output}":
			expanded = append(expanded, outputPath)
			opts.UseOutputFile = true
		case "{prompt}":
			expanded = append(expanded, prompt)
			opts.UseStdin = false
		default:
			arg = strings.ReplaceAll(arg, "{output}", outputPath)
			arg = strings.ReplaceAll(arg, "{model}", model)
			if strings.Contains(arg, "{prompt}") {
				opts.UseStdin = false
				arg = strings.ReplaceAll(arg, "{prompt}", prompt)
			}
			expanded = append(expanded, arg)
		}
	}
	return expanded, opts
}

func tempOutputPath() (string, func(), error) {
	f, err := os.CreateTemp("", "scout-summary-*")
	if err != nil {
		return "", nil, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func compactCommandOutput(stdout, stderr string) string {
	output := strings.TrimSpace(strings.Join([]string{stderr, stdout}, "\n"))
	if output == "" {
		return "no output"
	}
	const max = 2_000
	if len(output) > max {
		return output[:max] + "...(truncated)"
	}
	return output
}
