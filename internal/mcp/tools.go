package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/friedenberg/lux/internal/lsp"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error)

type ToolRegistry struct {
	tools    []Tool
	handlers map[string]ToolHandler
	bridge   *Bridge
}

func NewToolRegistry(bridge *Bridge) *ToolRegistry {
	r := &ToolRegistry{
		handlers: make(map[string]ToolHandler),
		bridge:   bridge,
	}
	r.registerBuiltinTools()
	return r
}

func (r *ToolRegistry) List() []Tool {
	return r.tools
}

func (r *ToolRegistry) Call(ctx context.Context, name string, args json.RawMessage) (*ToolCallResult, error) {
	handler, ok := r.handlers[name]
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown tool: %s", name)), nil
	}
	return handler(ctx, args)
}

func (r *ToolRegistry) register(name, description string, schema json.RawMessage, handler ToolHandler) {
	r.tools = append(r.tools, Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	})
	r.handlers[name] = handler
}

func (r *ToolRegistry) registerBuiltinTools() {
	r.register("lsp_hover", "Get type information, documentation, and signatures for a symbol. PREFER this over reading source files when you need to understand what a function/type does, its parameters, return types, or documentation. Unlike grep/read which show raw text, hover provides semantically-parsed information. Use cases: understanding function signatures, checking type definitions, reading docstrings.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleHover)

	r.register("lsp_definition", "Jump to the definition of any symbol (function, type, variable). PREFER this over grep when you know a symbol name and need its implementation. Uses semantic analysis to find the actual definition, not just string matches. Handles cross-file navigation, interface implementations, import sources.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleDefinition)

	r.register("lsp_references", "Find ALL usages of a symbol throughout the codebase. PREFER this over grep for finding where functions/types/variables are used - it understands scope and semantics, finding actual references not just string matches. Critical for impact analysis before refactoring, understanding how functions are called, tracing data flow.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"},
				"include_declaration": {"type": "boolean", "description": "Include the declaration in results", "default": true}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleReferences)

	r.register("lsp_completion", "Get context-aware code completions at a cursor position. Shows valid symbols, methods, and fields available in scope. Use when exploring available methods on a type, discovering struct fields, finding imported symbols, understanding API surfaces.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"}
			},
			"required": ["uri", "line", "character"]
		}`),
		r.handleCompletion)

	r.register("lsp_format", "Get formatting edits for a document according to language-standard style. Returns text edits needed to properly format the file. Note: returns edits but does not apply them - use Edit tool to apply changes.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"}
			},
			"required": ["uri"]
		}`),
		r.handleFormat)

	r.register("lsp_document_symbols", "Get a structured outline of all symbols in a file. PREFER this over reading entire files when you need to understand file structure. Returns hierarchical symbols: function/method names, type definitions, nested structures, top-level constants. Faster than parsing file content manually.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"}
			},
			"required": ["uri"]
		}`),
		r.handleDocumentSymbols)

	r.register("lsp_code_action", "Get suggested fixes, refactorings, and improvements for code at a range. Language servers suggest quick fixes for errors, refactoring operations (extract function, inline variable), import organization, code generation (implement interface).",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"start_line": {"type": "integer", "description": "0-indexed start line"},
				"start_character": {"type": "integer", "description": "0-indexed start character"},
				"end_line": {"type": "integer", "description": "0-indexed end line"},
				"end_character": {"type": "integer", "description": "0-indexed end character"}
			},
			"required": ["uri", "start_line", "start_character", "end_line", "end_character"]
		}`),
		r.handleCodeAction)

	r.register("lsp_rename", "Rename a symbol across the entire codebase with semantic accuracy. PREFER this over find-and-replace: only renames actual references, handles scoping correctly, updates imports appropriately. Returns workspace edit showing all changes.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"},
				"line": {"type": "integer", "description": "0-indexed line number"},
				"character": {"type": "integer", "description": "0-indexed character offset"},
				"new_name": {"type": "string", "description": "New name for the symbol"}
			},
			"required": ["uri", "line", "character", "new_name"]
		}`),
		r.handleRename)

	r.register("lsp_workspace_symbols", "Search for symbols (functions, types, constants) across the entire workspace by name pattern. PREFER this over grep when searching for symbol definitions by name. Use when you know a function/type name but not its file location.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Symbol name pattern to search for"},
				"uri": {"type": "string", "description": "Any file URI in the workspace (used to identify which LSP to query)"}
			},
			"required": ["query", "uri"]
		}`),
		r.handleWorkspaceSymbols)

	r.register("lsp_diagnostics", "Get compiler/linter diagnostics (errors, warnings, hints) for a file. Use to understand issues before making edits or to verify changes are correct.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"uri": {"type": "string", "description": "File URI (e.g., file:///path/to/file.go)"}
			},
			"required": ["uri"]
		}`),
		r.handleDiagnostics)
}

type positionArgs struct {
	URI       string `json:"uri"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

type referencesArgs struct {
	positionArgs
	IncludeDeclaration bool `json:"include_declaration"`
}

type formatArgs struct {
	URI string `json:"uri"`
}

type codeActionArgs struct {
	URI            string `json:"uri"`
	StartLine      int    `json:"start_line"`
	StartCharacter int    `json:"start_character"`
	EndLine        int    `json:"end_line"`
	EndCharacter   int    `json:"end_character"`
}

type renameArgs struct {
	positionArgs
	NewName string `json:"new_name"`
}

type workspaceSymbolsArgs struct {
	Query string `json:"query"`
	URI   string `json:"uri"`
}

type diagnosticsArgs struct {
	URI string `json:"uri"`
}

func (r *ToolRegistry) handleHover(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Hover(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleDefinition(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Definition(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleReferences(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a referencesArgs
	a.IncludeDeclaration = true // default
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.References(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character, a.IncludeDeclaration)
}

func (r *ToolRegistry) handleCompletion(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a positionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Completion(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character)
}

func (r *ToolRegistry) handleFormat(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a formatArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Format(ctx, lsp.DocumentURI(a.URI))
}

func (r *ToolRegistry) handleDocumentSymbols(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a formatArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.DocumentSymbols(ctx, lsp.DocumentURI(a.URI))
}

func (r *ToolRegistry) handleCodeAction(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a codeActionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.CodeAction(ctx, lsp.DocumentURI(a.URI),
		a.StartLine, a.StartCharacter, a.EndLine, a.EndCharacter)
}

func (r *ToolRegistry) handleRename(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a renameArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Rename(ctx, lsp.DocumentURI(a.URI), a.Line, a.Character, a.NewName)
}

func (r *ToolRegistry) handleWorkspaceSymbols(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a workspaceSymbolsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.WorkspaceSymbols(ctx, lsp.DocumentURI(a.URI), a.Query)
}

func (r *ToolRegistry) handleDiagnostics(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a diagnosticsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	return r.bridge.Diagnostics(ctx, lsp.DocumentURI(a.URI))
}
