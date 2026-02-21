package formatter

import (
	"path/filepath"
	"strings"

	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/pkg/filematch"
)

type Router struct {
	matchers   *filematch.MatcherSet
	formatters map[string]*config.Formatter
}

func NewRouter(cfg *config.FormatterConfig) (*Router, error) {
	matchers := filematch.NewMatcherSet()
	formatters := make(map[string]*config.Formatter)

	// TODO(task-7): Rewrite to accept []*filetype.Config for routing.
	// Fields were removed from config.Formatter; routing now lives in filetype configs.

	for i := range cfg.Formatters {
		f := &cfg.Formatters[i]
		if f.Disabled {
			continue
		}
		formatters[f.Name] = f
	}

	return &Router{
		matchers:   matchers,
		formatters: formatters,
	}, nil
}

func (r *Router) Match(filePath string) *config.Formatter {
	ext := strings.ToLower(filepath.Ext(filePath))
	name := r.matchers.Match(filePath, ext, "")
	if name == "" {
		return nil
	}
	return r.formatters[name]
}
