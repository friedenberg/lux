package formatter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/subprocess"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/output"
)

type Result struct {
	Formatted string
	Stderr    string
	Changed   bool
}

func ResolveExecutable(ctx context.Context, f *config.Formatter, executor subprocess.Executor) (string, error) {
	if f.Flake != "" {
		return executor.Build(ctx, f.Flake, f.Binary)
	}
	return config.ExpandEnvVars(f.Path), nil
}

func Format(ctx context.Context, f *config.Formatter, filePath string, content []byte, executor subprocess.Executor) (*Result, error) {
	binPath, err := ResolveExecutable(ctx, f, executor)
	if err != nil {
		return nil, fmt.Errorf("resolving formatter %s: %w", f.Name, err)
	}

	args := SubstituteArgs(f.Args, filePath)

	mode := f.EffectiveMode()
	switch mode {
	case config.FormatterModeStdin:
		return formatStdin(ctx, binPath, args, f.Env, content)
	case config.FormatterModeFilepath:
		return formatFilepath(ctx, binPath, args, f.Env, filePath, content)
	default:
		return nil, fmt.Errorf("unknown formatter mode: %s", mode)
	}
}

func buildCmd(ctx context.Context, binPath string, args []string, env map[string]string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binPath, args...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd
}

func formatStdin(ctx context.Context, binPath string, args []string, env map[string]string, content []byte) (*Result, error) {
	cmd := buildCmd(ctx, binPath, args, env)

	cmd.Stdin = bytes.NewReader(content)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		limited := output.LimitStderr(stderr.String())
		return nil, fmt.Errorf("formatter %s failed: %w\nstderr: %s", binPath, err, limited.Content)
	}

	formatted := stdout.String()
	limited := output.LimitStderr(stderr.String())
	return &Result{
		Formatted: formatted,
		Stderr:    limited.Content,
		Changed:   formatted != string(content),
	}, nil
}

func formatFilepath(ctx context.Context, binPath string, args []string, env map[string]string, filePath string, content []byte) (*Result, error) {
	tmpFile, err := os.CreateTemp("", "lux-fmt-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	fileArgs := substituteFilepathArgs(args, tmpPath)

	cmd := buildCmd(ctx, binPath, fileArgs, env)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		limited := output.LimitStderr(stderr.String())
		return nil, fmt.Errorf("formatter %s failed: %w\nstderr: %s", binPath, err, limited.Content)
	}

	formatted, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("reading formatted file: %w", err)
	}

	limited := output.LimitStderr(stderr.String())
	return &Result{
		Formatted: string(formatted),
		Stderr:    limited.Content,
		Changed:   string(formatted) != string(content),
	}, nil
}

func substituteFilepathArgs(args []string, filePath string) []string {
	result := make([]string, len(args))
	hasPlaceholder := false
	for i, arg := range args {
		if strings.Contains(arg, "{file}") {
			hasPlaceholder = true
		}
		result[i] = strings.ReplaceAll(arg, "{file}", filePath)
	}
	if !hasPlaceholder {
		result = append(result, filePath)
	}
	return result
}

func SubstituteArgs(args []string, filePath string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = strings.ReplaceAll(arg, "{file}", filePath)
	}
	return result
}

// FormatReader formats content from a reader, useful for piping.
func FormatReader(ctx context.Context, f *config.Formatter, filePath string, reader io.Reader, executor subprocess.Executor) (*Result, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return Format(ctx, f, filePath, content, executor)
}
