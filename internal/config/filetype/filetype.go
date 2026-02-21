package filetype

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Name          string   `toml:"-"`
	Extensions    []string `toml:"extensions"`
	Patterns      []string `toml:"patterns"`
	LanguageIDs   []string `toml:"language_ids"`
	LSP           string   `toml:"lsp"`
	Formatters    []string `toml:"formatters"`
	FormatterMode string   `toml:"formatter_mode"`
	LSPFormat     string   `toml:"lsp_format"`
}

func LoadDir(dir string) ([]*Config, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading filetype dir %s: %w", dir, err)
	}

	var configs []*Config
	var names []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		names = append(names, entry.Name())
	}

	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var cfg Config
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		cfg.Name = strings.TrimSuffix(name, ".toml")
		configs = append(configs, &cfg)
	}

	return configs, nil
}

func Merge(global, project []*Config) []*Config {
	projectByName := make(map[string]*Config)
	for _, p := range project {
		projectByName[p.Name] = p
	}

	var merged []*Config
	seen := make(map[string]bool)

	for _, g := range global {
		if p, ok := projectByName[g.Name]; ok {
			merged = append(merged, p)
		} else {
			merged = append(merged, g)
		}
		seen[g.Name] = true
	}

	for _, p := range project {
		if !seen[p.Name] {
			merged = append(merged, p)
		}
	}

	return merged
}

func Validate(configs []*Config, lsps, formatters map[string]bool) error {
	seenExts := make(map[string]string)
	seenLangs := make(map[string]string)

	for _, cfg := range configs {
		if len(cfg.Extensions) == 0 && len(cfg.Patterns) == 0 && len(cfg.LanguageIDs) == 0 {
			return fmt.Errorf("filetype/%s.toml: at least one of extensions, patterns, or language_ids is required", cfg.Name)
		}

		if cfg.LSP != "" && !lsps[cfg.LSP] {
			return fmt.Errorf("filetype/%s.toml: lsp %q not found in lsps.toml", cfg.Name, cfg.LSP)
		}

		for _, f := range cfg.Formatters {
			if !formatters[f] {
				return fmt.Errorf("filetype/%s.toml: formatter %q not found in formatters.toml", cfg.Name, f)
			}
		}

		if cfg.FormatterMode != "" && cfg.FormatterMode != "chain" && cfg.FormatterMode != "fallback" {
			return fmt.Errorf("filetype/%s.toml: invalid formatter_mode %q (must be \"chain\" or \"fallback\")", cfg.Name, cfg.FormatterMode)
		}

		if cfg.LSPFormat != "" && cfg.LSPFormat != "never" && cfg.LSPFormat != "fallback" && cfg.LSPFormat != "prefer" {
			return fmt.Errorf("filetype/%s.toml: invalid lsp_format %q (must be \"never\", \"fallback\", or \"prefer\")", cfg.Name, cfg.LSPFormat)
		}

		for _, ext := range cfg.Extensions {
			if other, ok := seenExts[ext]; ok {
				return fmt.Errorf("filetype/%s.toml: extension %q also claimed by filetype/%s.toml", cfg.Name, ext, other)
			}
			seenExts[ext] = cfg.Name
		}

		for _, lang := range cfg.LanguageIDs {
			if other, ok := seenLangs[lang]; ok {
				return fmt.Errorf("filetype/%s.toml: language_id %q also claimed by filetype/%s.toml", cfg.Name, lang, other)
			}
			seenLangs[lang] = cfg.Name
		}
	}

	return nil
}
