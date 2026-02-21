package formatter

import (
	"testing"

	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/config/filetype"
)

func TestRouterMatch_Filetype(t *testing.T) {
	filetypes := []*filetype.Config{
		{Name: "go", Extensions: []string{"go"}, Formatters: []string{"golines"}},
		{Name: "python", Extensions: []string{"py"}, Formatters: []string{"isort", "black"}, FormatterMode: "chain"},
	}
	formatters := map[string]*config.Formatter{
		"golines": {Name: "golines", Flake: "nixpkgs#golines"},
		"isort":   {Name: "isort", Flake: "nixpkgs#isort"},
		"black":   {Name: "black", Flake: "nixpkgs#black"},
	}

	router, err := NewRouter(filetypes, formatters)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		filePath string
		wantFmt  int
		wantMode string
	}{
		{"go file", "/src/main.go", 1, "chain"},
		{"python file", "/src/main.py", 2, "chain"},
		{"unknown file", "/src/readme.md", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.Match(tt.filePath)
			if tt.wantFmt == 0 {
				if result != nil {
					t.Errorf("expected nil result for %s", tt.filePath)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected result for %s", tt.filePath)
			}
			if len(result.Formatters) != tt.wantFmt {
				t.Errorf("formatters count = %d, want %d", len(result.Formatters), tt.wantFmt)
			}
			if result.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", result.Mode, tt.wantMode)
			}
		})
	}
}
