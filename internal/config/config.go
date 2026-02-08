package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Socket string `toml:"socket"`
	LSPs   []LSP  `toml:"lsp"`
}

type LSP struct {
	Name        string   `toml:"name"`
	Flake       string   `toml:"flake"`
	Extensions  []string `toml:"extensions"`
	Patterns    []string `toml:"patterns"`
	LanguageIDs []string `toml:"language_ids"`
	Args        []string `toml:"args"`
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lux")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "lux")
	}
	return filepath.Join(home, ".config", "lux")
}

func dataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "lux")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".local", "share", "lux")
	}
	return filepath.Join(home, ".local", "share", "lux")
}

func runtimeDir() string {
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return xdg
	}
	return os.TempDir()
}

func ConfigPath() string {
	return filepath.Join(configDir(), "lsps.toml")
}

func DataDir() string {
	return dataDir()
}

func CapabilitiesDir() string {
	return filepath.Join(dataDir(), "capabilities")
}

func (c *Config) SocketPath() string {
	if c.Socket != "" {
		return c.Socket
	}
	return filepath.Join(runtimeDir(), "lux.sock")
}

func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	names := make(map[string]bool)
	for i, lsp := range c.LSPs {
		if lsp.Name == "" {
			return fmt.Errorf("lsp[%d]: name is required", i)
		}
		if lsp.Flake == "" {
			return fmt.Errorf("lsp[%d] (%s): flake is required", i, lsp.Name)
		}
		if names[lsp.Name] {
			return fmt.Errorf("lsp[%d]: duplicate name %q", i, lsp.Name)
		}
		names[lsp.Name] = true

		if len(lsp.Extensions) == 0 && len(lsp.Patterns) == 0 && len(lsp.LanguageIDs) == 0 {
			return fmt.Errorf("lsp[%d] (%s): at least one of extensions, patterns, or language_ids is required", i, lsp.Name)
		}
	}
	return nil
}

func (c *Config) FindLSP(name string) *LSP {
	for i := range c.LSPs {
		if c.LSPs[i].Name == name {
			return &c.LSPs[i]
		}
	}
	return nil
}

func Save(cfg *Config) error {
	path := ConfigPath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil
}

func AddLSP(lsp LSP) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	for i, existing := range cfg.LSPs {
		if existing.Name == lsp.Name {
			cfg.LSPs[i] = lsp
			return Save(cfg)
		}
	}

	cfg.LSPs = append(cfg.LSPs, lsp)
	return Save(cfg)
}
