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

func TestValidate_Valid(t *testing.T) {
	lsps := map[string]bool{"gopls": true, "pyright": true}
	fmts := map[string]bool{"golines": true, "isort": true, "black": true}

	configs := []*Config{
		{Name: "go", Extensions: []string{"go"}, LSP: "gopls", Formatters: []string{"golines"}},
		{Name: "python", Extensions: []string{"py"}, LSP: "pyright", Formatters: []string{"isort", "black"}, FormatterMode: "chain"},
	}

	if err := Validate(configs, lsps, fmts); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_NoMatchingFields(t *testing.T) {
	configs := []*Config{{Name: "bad", LSP: "gopls"}}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for missing extensions/patterns/language_ids")
	}
}

func TestValidate_UnknownLSP(t *testing.T) {
	configs := []*Config{{Name: "go", Extensions: []string{"go"}, LSP: "unknown"}}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for unknown LSP")
	}
}

func TestValidate_UnknownFormatter(t *testing.T) {
	configs := []*Config{{Name: "go", Extensions: []string{"go"}, LSP: "gopls", Formatters: []string{"unknown"}}}
	err := Validate(configs, map[string]bool{"gopls": true}, map[string]bool{"golines": true})
	if err == nil {
		t.Fatal("expected error for unknown formatter")
	}
}

func TestValidate_DuplicateExtension(t *testing.T) {
	configs := []*Config{
		{Name: "go", Extensions: []string{"go"}, LSP: "gopls"},
		{Name: "golang", Extensions: []string{"go"}, LSP: "gopls"},
	}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate extension")
	}
}

func TestValidate_DuplicateLanguageID(t *testing.T) {
	configs := []*Config{
		{Name: "go", LanguageIDs: []string{"go"}, LSP: "gopls"},
		{Name: "golang", LanguageIDs: []string{"go"}, LSP: "gopls"},
	}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate language_id")
	}
}

func TestValidate_InvalidFormatterMode(t *testing.T) {
	configs := []*Config{{Name: "go", Extensions: []string{"go"}, LSP: "gopls", FormatterMode: "invalid"}}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for invalid formatter_mode")
	}
}

func TestValidate_InvalidLSPFormat(t *testing.T) {
	configs := []*Config{{Name: "go", Extensions: []string{"go"}, LSP: "gopls", LSPFormat: "invalid"}}
	err := Validate(configs, map[string]bool{"gopls": true}, nil)
	if err == nil {
		t.Fatal("expected error for invalid lsp_format")
	}
}

func TestValidate_EmptyLSP(t *testing.T) {
	configs := []*Config{{Name: "go", Extensions: []string{"go"}, Formatters: []string{"golines"}}}
	err := Validate(configs, nil, map[string]bool{"golines": true})
	if err != nil {
		t.Errorf("LSP should be optional, got: %v", err)
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
