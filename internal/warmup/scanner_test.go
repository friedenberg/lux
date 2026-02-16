package warmup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarbel-llc/lux/internal/config"
)

func TestScanner_ScanDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create files matching different LSPs
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "lib.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)

	cfg := &config.Config{
		LSPs: []config.LSP{
			{Name: "gopls", Flake: "nixpkgs#gopls", Extensions: []string{".go"}},
			{Name: "pyright", Flake: "nixpkgs#pyright", Extensions: []string{".py"}},
			{Name: "rust-analyzer", Flake: "nixpkgs#rust-analyzer", Extensions: []string{".rs"}},
		},
	}

	scanner := NewScanner(cfg)
	result := scanner.ScanDirectories([]string{dir})

	if !result.LSPNames["gopls"] {
		t.Error("expected gopls to be found")
	}
	if !result.LSPNames["pyright"] {
		t.Error("expected pyright to be found")
	}
	if result.LSPNames["rust-analyzer"] {
		t.Error("expected rust-analyzer to NOT be found")
	}
}

func TestScanner_ScanDirectories_SkipsDirs(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "index.js"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "config.py"), []byte(""), 0644)

	cfg := &config.Config{
		LSPs: []config.LSP{
			{Name: "tsserver", Flake: "nixpkgs#tsserver", Extensions: []string{".js"}},
			{Name: "pyright", Flake: "nixpkgs#pyright", Extensions: []string{".py"}},
		},
	}

	scanner := NewScanner(cfg)
	result := scanner.ScanDirectories([]string{dir})

	if result.LSPNames["tsserver"] {
		t.Error("expected tsserver to NOT be found (file in node_modules)")
	}
	if result.LSPNames["pyright"] {
		t.Error("expected pyright to NOT be found (file in .git)")
	}
}

func TestScanner_ScanDirectories_MaxDepth(t *testing.T) {
	dir := t.TempDir()

	// Create file at depth 4 (beyond max depth of 3)
	deep := filepath.Join(dir, "a", "b", "c", "d")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "main.go"), []byte("package main"), 0644)

	// Create file at depth 3 (within max depth)
	shallow := filepath.Join(dir, "a", "b", "c")
	os.WriteFile(filepath.Join(shallow, "lib.py"), []byte(""), 0644)

	cfg := &config.Config{
		LSPs: []config.LSP{
			{Name: "gopls", Flake: "nixpkgs#gopls", Extensions: []string{".go"}},
			{Name: "pyright", Flake: "nixpkgs#pyright", Extensions: []string{".py"}},
		},
	}

	scanner := NewScanner(cfg)
	result := scanner.ScanDirectories([]string{dir})

	if result.LSPNames["gopls"] {
		t.Error("expected gopls to NOT be found (file beyond max depth)")
	}
	if !result.LSPNames["pyright"] {
		t.Error("expected pyright to be found (file within max depth)")
	}
}

func TestScanner_ScanDirectories_ShortCircuits(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.py"), []byte(""), 0644)

	cfg := &config.Config{
		LSPs: []config.LSP{
			{Name: "gopls", Flake: "nixpkgs#gopls", Extensions: []string{".go"}},
			{Name: "pyright", Flake: "nixpkgs#pyright", Extensions: []string{".py"}},
		},
	}

	scanner := NewScanner(cfg)
	result := scanner.ScanDirectories([]string{dir})

	if len(result.LSPNames) != 2 {
		t.Errorf("expected 2 LSPs found, got %d", len(result.LSPNames))
	}
}

func TestScanner_ScanDirectories_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	scanner := NewScanner(cfg)
	result := scanner.ScanDirectories([]string{dir})

	if len(result.LSPNames) != 0 {
		t.Errorf("expected 0 LSPs found, got %d", len(result.LSPNames))
	}
}

func TestScanner_AllLSPNames(t *testing.T) {
	cfg := &config.Config{
		LSPs: []config.LSP{
			{Name: "gopls", Flake: "f"},
			{Name: "pyright", Flake: "f"},
		},
	}

	scanner := NewScanner(cfg)
	names := scanner.AllLSPNames()

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["gopls"] || !found["pyright"] {
		t.Error("expected gopls and pyright in names")
	}
}
