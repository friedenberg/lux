package filetype

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFiletype(t *testing.T) {
	dir := t.TempDir()
	content := `
extensions = ["go"]
patterns = ["go.mod", "go.sum"]
language_ids = ["go"]
lsp = "gopls"
formatters = ["golines"]
formatter_mode = "chain"
lsp_format = "fallback"
`
	if err := os.WriteFile(filepath.Join(dir, "go.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	configs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	cfg := configs[0]
	if cfg.Name != "go" {
		t.Errorf("name = %q, want %q", cfg.Name, "go")
	}
	if len(cfg.Extensions) != 1 || cfg.Extensions[0] != "go" {
		t.Errorf("extensions = %v, want [go]", cfg.Extensions)
	}
	if cfg.LSP != "gopls" {
		t.Errorf("lsp = %q, want %q", cfg.LSP, "gopls")
	}
	if len(cfg.Formatters) != 1 || cfg.Formatters[0] != "golines" {
		t.Errorf("formatters = %v, want [golines]", cfg.Formatters)
	}
	if cfg.FormatterMode != "chain" {
		t.Errorf("formatter_mode = %q, want %q", cfg.FormatterMode, "chain")
	}
	if cfg.LSPFormat != "fallback" {
		t.Errorf("lsp_format = %q, want %q", cfg.LSPFormat, "fallback")
	}
}

func TestLoadDir_Empty(t *testing.T) {
	dir := t.TempDir()
	configs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected 0 configs, got %d", len(configs))
	}
}

func TestLoadDir_NonExistent(t *testing.T) {
	configs, err := LoadDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadDir should not error on missing dir: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected 0 configs, got %d", len(configs))
	}
}

func TestLoadDir_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	goContent := `
extensions = ["go"]
lsp = "gopls"
`
	pyContent := `
extensions = ["py"]
lsp = "pyright"
formatters = ["isort", "black"]
`
	os.WriteFile(filepath.Join(dir, "go.toml"), []byte(goContent), 0644)
	os.WriteFile(filepath.Join(dir, "python.toml"), []byte(pyContent), 0644)

	configs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}

	// Should be sorted alphabetically by filename
	if configs[0].Name != "go" {
		t.Errorf("first config name = %q, want %q", configs[0].Name, "go")
	}
	if configs[1].Name != "python" {
		t.Errorf("second config name = %q, want %q", configs[1].Name, "python")
	}
}
