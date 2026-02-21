package server

import (
	"testing"

	"github.com/amarbel-llc/lux/internal/config/filetype"
	"github.com/amarbel-llc/lux/internal/lsp"
)

func TestRouter_RouteByURI_Extension(t *testing.T) {
	filetypes := []*filetype.Config{
		{Name: "go", Extensions: []string{"go"}, LSP: "gopls"},
		{Name: "python", Extensions: []string{"py"}, LSP: "pyright"},
	}

	router, err := NewRouter(filetypes)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		uri     string
		wantLSP string
	}{
		{"go file", "file:///src/main.go", "gopls"},
		{"python file", "file:///src/main.py", "pyright"},
		{"unknown file", "file:///src/readme.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.RouteByURI(lsp.DocumentURI(tt.uri))
			if got != tt.wantLSP {
				t.Errorf("RouteByURI(%q) = %q, want %q", tt.uri, got, tt.wantLSP)
			}
		})
	}
}

func TestRouter_FiletypeByURI(t *testing.T) {
	filetypes := []*filetype.Config{
		{Name: "go", Extensions: []string{"go"}, LSP: "gopls", Formatters: []string{"golines"}},
	}

	router, err := NewRouter(filetypes)
	if err != nil {
		t.Fatal(err)
	}

	ft := router.FiletypeByURI(lsp.DocumentURI("file:///src/main.go"))
	if ft == nil {
		t.Fatal("expected filetype config for .go file")
	}
	if ft.LSP != "gopls" {
		t.Errorf("lsp = %q, want %q", ft.LSP, "gopls")
	}
	if len(ft.Formatters) != 1 || ft.Formatters[0] != "golines" {
		t.Errorf("formatters = %v, want [golines]", ft.Formatters)
	}
}
