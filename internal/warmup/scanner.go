package warmup

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/pkg/filematch"
)

const (
	maxDepth = 3
	maxFiles = 5000
)

var skipDirs = map[string]bool{
	".git":        true,
	".hg":         true,
	"node_modules": true,
	"vendor":      true,
	"__pycache__": true,
	".direnv":     true,
	"result":      true,
}

type ScanResult struct {
	LSPNames map[string]bool
}

type Scanner struct {
	cfg *config.Config
}

func NewScanner(cfg *config.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

func (s *Scanner) ScanDirectories(dirs []string) ScanResult {
	result := ScanResult{LSPNames: make(map[string]bool)}
	total := len(s.cfg.LSPs)
	if total == 0 {
		return result
	}

	// TODO(task-6): Rewrite to use filetype configs for matching.
	// Fields were removed from config.LSP; routing now lives in filetype configs.
	matchers := make(map[string]*filematch.Matcher, total)

	fileCount := 0
	for _, dir := range dirs {
		if len(result.LSPNames) >= total {
			break
		}
		baseDepth := strings.Count(filepath.Clean(dir), string(filepath.Separator))

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}

			if d.IsDir() {
				if skipDirs[d.Name()] {
					return fs.SkipDir
				}
				depth := strings.Count(filepath.Clean(path), string(filepath.Separator)) - baseDepth
				if depth > maxDepth {
					return fs.SkipDir
				}
				return nil
			}

			fileCount++
			if fileCount > maxFiles {
				return fs.SkipAll
			}

			ext := filepath.Ext(path)
			for name, m := range matchers {
				if result.LSPNames[name] {
					continue
				}
				if m.MatchesExtension(ext) || m.MatchesPattern(path) {
					result.LSPNames[name] = true
					if len(result.LSPNames) >= total {
						return fs.SkipAll
					}
				}
			}

			return nil
		})
	}

	return result
}

func (s *Scanner) AllLSPNames() []string {
	names := make([]string, 0, len(s.cfg.LSPs))
	for _, l := range s.cfg.LSPs {
		names = append(names, l.Name)
	}
	return names
}
