package formatter

import (
	"testing"

	"github.com/amarbel-llc/lux/internal/config"
)

func TestRouterMatch(t *testing.T) {
	// TODO(task-7): Routing fields moved to filetype configs.
	// Router currently returns no matches. Full rewrite in Task 7.
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "gofumpt", Flake: "nixpkgs#gofumpt"},
			{Name: "prettier", Path: "/usr/bin/prettier"},
			{Name: "nixfmt", Flake: "nixpkgs#nixfmt"},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	// Without filetype configs wired in, all matches return nil
	f := router.Match("/src/main.go")
	if f != nil {
		t.Errorf("Match() should return nil until filetype routing is wired, got %s", f.Name)
	}
}

func TestRouterDisabledFormatters(t *testing.T) {
	// TODO(task-7): Full routing test after filetype config rewrite.
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "gofumpt", Flake: "nixpkgs#gofumpt"},
			{Name: "prettier", Path: "/usr/bin/prettier", Disabled: true},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	// Disabled formatters should not be in the formatters map
	f := router.Match("/src/app.js")
	if f != nil {
		t.Errorf("Match() should return nil for disabled formatter, got %s", f.Name)
	}
}

func TestRouterWithPatterns(t *testing.T) {
	// TODO(task-7): Pattern matching test after filetype config rewrite.
	cfg := &config.FormatterConfig{
		Formatters: []config.Formatter{
			{Name: "prettier", Path: "/usr/bin/prettier"},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter() error: %v", err)
	}

	// Without filetype configs, no pattern matching occurs
	f := router.Match("/src/webpack.config.js")
	if f != nil {
		t.Errorf("Match() should return nil until filetype routing is wired, got %s", f.Name)
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
