package formatter

import (
	"testing"

	"github.com/amarbel-llc/lux/internal/config"
)

func TestRouterMatch(t *testing.T) {
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "gofumpt", Flake: "nixpkgs#gofumpt", Extensions: []string{"go"}},
			{Name: "prettier", Path: "/usr/bin/prettier", Extensions: []string{"js", "ts", "json"}},
			{Name: "nixfmt", Flake: "nixpkgs#nixfmt", Extensions: []string{"nix"}},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"go file", "/src/main.go", "gofumpt"},
		{"js file", "/src/app.js", "prettier"},
		{"ts file", "/src/app.ts", "prettier"},
		{"json file", "/src/config.json", "prettier"},
		{"nix file", "/src/flake.nix", "nixfmt"},
		{"unknown file", "/src/readme.md", ""},
		{"no extension", "/src/Makefile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := router.Match(tt.filePath)
			if tt.want == "" {
				if f != nil {
					t.Errorf("Match(%q) = %s, want nil", tt.filePath, f.Name)
				}
				return
			}
			if f == nil {
				t.Fatalf("Match(%q) = nil, want %s", tt.filePath, tt.want)
			}
			if f.Name != tt.want {
				t.Errorf("Match(%q) = %s, want %s", tt.filePath, f.Name, tt.want)
			}
		})
	}
}

func TestRouterDisabledFormatters(t *testing.T) {
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "gofumpt", Flake: "nixpkgs#gofumpt", Extensions: []string{"go"}},
			{Name: "prettier", Path: "/usr/bin/prettier", Extensions: []string{"js"}, Disabled: true},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	f := router.Match("/src/app.js")
	if f != nil {
		t.Errorf("Match() should return nil for disabled formatter, got %s", f.Name)
	}

	f = router.Match("/src/main.go")
	if f == nil {
		t.Fatal("Match() should return gofumpt for .go files")
	}
	if f.Name != "gofumpt" {
		t.Errorf("Match() = %s, want gofumpt", f.Name)
	}
}

func TestRouterWithPatterns(t *testing.T) {
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "prettier", Path: "/usr/bin/prettier", Patterns: []string{"*.config.js", "*.config.ts"}},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	f := router.Match("/src/webpack.config.js")
	if f == nil {
		t.Fatal("Match() should match *.config.js pattern")
	}
	if f.Name != "prettier" {
		t.Errorf("Match() = %s, want prettier", f.Name)
	}
}

func TestRouterEmptyConfig(t *testing.T) {
	cfg := &config.FormatterConfig{}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	f := router.Match("/src/main.go")
	if f != nil {
		t.Errorf("Match() should return nil for empty config, got %s", f.Name)
	}
}
