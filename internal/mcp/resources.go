package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/friedenberg/lux/internal/config"
	"github.com/friedenberg/lux/internal/lsp"
	"github.com/friedenberg/lux/internal/subprocess"
	"github.com/friedenberg/lux/pkg/filematch"
)

type ResourceRegistry struct {
	pool    *subprocess.Pool
	bridge  *Bridge
	config  *config.Config
	cwd     string
	matcher *filematch.MatcherSet
}

func NewResourceRegistry(pool *subprocess.Pool, bridge *Bridge, cfg *config.Config) *ResourceRegistry {
	cwd, _ := os.Getwd()

	matcher := filematch.NewMatcherSet()
	for _, l := range cfg.LSPs {
		matcher.Add(l.Name, l.Extensions, l.Patterns, l.LanguageIDs)
	}

	return &ResourceRegistry{
		pool:    pool,
		bridge:  bridge,
		config:  cfg,
		cwd:     cwd,
		matcher: matcher,
	}
}

func (r *ResourceRegistry) List() []Resource {
	return []Resource{
		{
			URI:         "lux://status",
			Name:        "LSP Status",
			Description: "Current status of configured language servers including which are running",
			MimeType:    "application/json",
		},
		{
			URI:         "lux://languages",
			Name:        "Supported Languages",
			Description: "Languages supported by lux with their file extensions and patterns",
			MimeType:    "application/json",
		},
		{
			URI:         "lux://files",
			Name:        "Project Files",
			Description: "Files in the current directory that match configured LSP extensions/patterns",
			MimeType:    "application/json",
		},
	}
}

func (r *ResourceRegistry) ListTemplates() []ResourceTemplate {
	return []ResourceTemplate{
		{
			URITemplate: "lux://symbols/{uri}",
			Name:        "File Symbols",
			Description: "All symbols (functions, types, constants, etc.) in a file as reported by the LSP. Use file:// URI encoding for the path (e.g., lux://symbols/file:///path/to/file.go)",
			MimeType:    "application/json",
		},
	}
}

func (r *ResourceRegistry) Read(ctx context.Context, uri string) (*ResourceReadResult, error) {
	switch uri {
	case "lux://status":
		return r.readStatus()
	case "lux://languages":
		return r.readLanguages()
	case "lux://files":
		return r.readFiles()
	default:
		if strings.HasPrefix(uri, "lux://symbols/") {
			fileURI := strings.TrimPrefix(uri, "lux://symbols/")
			return r.readSymbols(ctx, uri, fileURI)
		}
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

type statusResponse struct {
	ConfiguredLSPs      []lspStatus `json:"configured_lsps"`
	SupportedExtensions []string    `json:"supported_extensions"`
	SupportedLanguages  []string    `json:"supported_languages"`
}

type lspStatus struct {
	Name       string   `json:"name"`
	Flake      string   `json:"flake"`
	Extensions []string `json:"extensions,omitempty"`
	Patterns   []string `json:"patterns,omitempty"`
	State      string   `json:"state"`
}

func (r *ResourceRegistry) readStatus() (*ResourceReadResult, error) {
	statuses := r.pool.Status()
	statusMap := make(map[string]string)
	for _, s := range statuses {
		statusMap[s.Name] = s.State
	}

	var lsps []lspStatus
	extSet := make(map[string]bool)
	langSet := make(map[string]bool)

	for _, l := range r.config.LSPs {
		state := statusMap[l.Name]
		if state == "" {
			state = "idle"
		}
		lsps = append(lsps, lspStatus{
			Name:       l.Name,
			Flake:      l.Flake,
			Extensions: l.Extensions,
			Patterns:   l.Patterns,
			State:      state,
		})

		for _, ext := range l.Extensions {
			extSet[ext] = true
		}
		for _, lang := range l.LanguageIDs {
			langSet[lang] = true
		}
	}

	var extensions, languages []string
	for ext := range extSet {
		extensions = append(extensions, ext)
	}
	for lang := range langSet {
		languages = append(languages, lang)
	}
	sort.Strings(extensions)
	sort.Strings(languages)

	resp := statusResponse{
		ConfiguredLSPs:      lsps,
		SupportedExtensions: extensions,
		SupportedLanguages:  languages,
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      "lux://status",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

type languagesResponse map[string]languageInfo

type languageInfo struct {
	LSP        string   `json:"lsp"`
	Extensions []string `json:"extensions,omitempty"`
	Patterns   []string `json:"patterns,omitempty"`
}

func (r *ResourceRegistry) readLanguages() (*ResourceReadResult, error) {
	resp := make(languagesResponse)

	for _, l := range r.config.LSPs {
		for _, langID := range l.LanguageIDs {
			resp[langID] = languageInfo{
				LSP:        l.Name,
				Extensions: l.Extensions,
				Patterns:   l.Patterns,
			}
		}
		if len(l.LanguageIDs) == 0 && len(l.Extensions) > 0 {
			resp[l.Name] = languageInfo{
				LSP:        l.Name,
				Extensions: l.Extensions,
				Patterns:   l.Patterns,
			}
		}
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      "lux://languages",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

type filesResponse struct {
	Root  string     `json:"root"`
	Files []string   `json:"files"`
	Stats filesStats `json:"stats"`
}

type filesStats struct {
	TotalFiles  int            `json:"total_files"`
	ByExtension map[string]int `json:"by_extension"`
}

func (r *ResourceRegistry) readFiles() (*ResourceReadResult, error) {
	var files []string
	byExt := make(map[string]int)

	err := filepath.Walk(r.cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		relPath, _ := filepath.Rel(r.cwd, path)

		if r.matcher.Match(relPath, ext, "") != "" {
			files = append(files, relPath)
			byExt[ext]++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	resp := filesResponse{
		Root:  r.cwd,
		Files: files,
		Stats: filesStats{
			TotalFiles:  len(files),
			ByExtension: byExt,
		},
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      "lux://files",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

type symbolsResponse struct {
	URI     string   `json:"uri"`
	Symbols []Symbol `json:"symbols"`
}

func (r *ResourceRegistry) readSymbols(ctx context.Context, resourceURI, fileURI string) (*ResourceReadResult, error) {
	symbols, err := r.bridge.DocumentSymbolsRaw(ctx, lsp.DocumentURI(fileURI))
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	resp := symbolsResponse{
		URI:     fileURI,
		Symbols: symbols,
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, err
	}

	return &ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      resourceURI,
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}
