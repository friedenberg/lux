package mcp

import (
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
)

const codeExplorationPrompt = `When exploring an unfamiliar codebase, use lux LSP tools strategically:

1. START WITH STRUCTURE: Use document_symbols on key files to understand the file's API surface - what functions, types, and constants are defined.

2. UNDERSTAND TYPES: Use hover on type names to see their full definition and documentation without navigating away.

3. FOLLOW DEFINITIONS: Use definition to jump to where symbols are defined. This is faster and more accurate than grep for finding implementations.

4. TRACE USAGE: Use references to understand how a function or type is used throughout the codebase. Essential before modifying public APIs.

5. EXPLORE APIS: Use completion to discover available methods and fields on types you're working with.

WHEN TO USE LSP vs GREP:
- Use references instead of grep when searching for symbol usage
- Use definition instead of grep when finding where something is defined
- Use document_symbols instead of reading files to understand structure
- Use grep when searching for string literals, comments, or patterns

SUPPORTED FILES:
Read the lux://languages resource to see which file types have LSP support.
Read the lux://files resource to see all supported files in the current project.`

const refactoringGuidePrompt = `For safe, comprehensive refactoring using lux:

RENAMING:
1. First use references to see all usages of the symbol
2. Review the references to understand the impact
3. Use rename to rename across all files semantically
4. The rename operation handles scoping - it won't rename unrelated symbols

BEFORE MODIFYING A FUNCTION:
1. Use references to find all callers
2. Use hover on call sites to understand expected behavior
3. Check if function is part of an interface with definition

FINDING IMPLEMENTATIONS:
- Use definition on interface methods to find implementations
- Use references on types to find where they're instantiated

CODE ACTIONS:
Use code_action to discover automated fixes and refactorings available at a specific code location. LSPs can suggest:
- Extracting code to functions
- Inlining variables
- Organizing imports
- Implementing interface methods

DIAGNOSTICS:
Use diagnostics to check for errors and warnings before and after making changes.`

type PromptRegistry struct {
	prompts map[string]promptDef
}

type promptDef struct {
	prompt  protocol.Prompt
	content string
}

func NewPromptRegistry() *PromptRegistry {
	r := &PromptRegistry{
		prompts: make(map[string]promptDef),
	}

	r.prompts["code-exploration"] = promptDef{
		prompt: protocol.Prompt{
			Name:        "code-exploration",
			Description: "Best practices for exploring and understanding code using LSP tools",
		},
		content: codeExplorationPrompt,
	}

	r.prompts["refactoring-guide"] = promptDef{
		prompt: protocol.Prompt{
			Name:        "refactoring-guide",
			Description: "How to safely refactor code using LSP-assisted tools",
		},
		content: refactoringGuidePrompt,
	}

	return r
}

func (r *PromptRegistry) List() []protocol.Prompt {
	var result []protocol.Prompt
	for _, p := range r.prompts {
		result = append(result, p.prompt)
	}
	return result
}

func (r *PromptRegistry) Get(name string, args map[string]string) (*protocol.PromptGetResult, error) {
	def, ok := r.prompts[name]
	if !ok {
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}

	return &protocol.PromptGetResult{
		Description: def.prompt.Description,
		Messages: []protocol.PromptMessage{
			{
				Role:    "user",
				Content: protocol.TextContent(def.content),
			},
		},
	}, nil
}
