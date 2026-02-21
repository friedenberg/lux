package server

import (
	"encoding/json"
	"sync"

	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/lsp"
	"github.com/amarbel-llc/lux/pkg/filematch"
)

type Router struct {
	matchers    *filematch.MatcherSet
	languageMap map[lsp.DocumentURI]string
	mu          sync.RWMutex
}

func NewRouter(cfg *config.Config) (*Router, error) {
	matchers := filematch.NewMatcherSet()

	// TODO(task-6): Rewrite to accept []*filetype.Config for routing.
	// Fields were removed from config.LSP; routing now lives in filetype configs.

	_ = cfg // suppress unused warning until full rewrite

	return &Router{
		matchers:    matchers,
		languageMap: make(map[lsp.DocumentURI]string),
	}, nil
}

func (r *Router) Route(method string, params json.RawMessage) string {
	var paramsMap map[string]any
	if err := json.Unmarshal(params, &paramsMap); err != nil {
		return ""
	}

	uri := lsp.ExtractURI(method, paramsMap)
	if uri == "" {
		return ""
	}

	if method == lsp.MethodTextDocumentDidOpen {
		langID := lsp.ExtractLanguageID(paramsMap)
		if langID != "" {
			r.mu.Lock()
			r.languageMap[uri] = langID
			r.mu.Unlock()
		}
	}

	if method == lsp.MethodTextDocumentDidClose {
		r.mu.Lock()
		delete(r.languageMap, uri)
		r.mu.Unlock()
	}

	r.mu.RLock()
	langID := r.languageMap[uri]
	r.mu.RUnlock()

	path := uri.Path()
	ext := uri.Extension()

	return r.matchers.Match(path, ext, langID)
}

func (r *Router) RouteByURI(uri lsp.DocumentURI) string {
	r.mu.RLock()
	langID := r.languageMap[uri]
	r.mu.RUnlock()

	path := uri.Path()
	ext := uri.Extension()

	return r.matchers.Match(path, ext, langID)
}

func (r *Router) RouteByExtension(ext string) string {
	return r.matchers.MatchByExtension(ext)
}

func (r *Router) RouteByLanguageID(langID string) string {
	return r.matchers.MatchByLanguageID(langID)
}

func (r *Router) SetLanguageID(uri lsp.DocumentURI, langID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.languageMap[uri] = langID
}

func (r *Router) GetLanguageID(uri lsp.DocumentURI) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.languageMap[uri]
}
