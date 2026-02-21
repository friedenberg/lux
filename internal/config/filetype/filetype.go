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
