package mcp

import (
	"context"
	"encoding/json"

	"github.com/friedenberg/lux/internal/jsonrpc"
)

type Handler struct {
	server      *Server
	initialized bool
}

func NewHandler(s *Server) *Handler {
	return &Handler{server: s}
}

func (h *Handler) Handle(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	switch msg.Method {
	case MethodInitialize:
		return h.handleInitialize(ctx, msg)
	case MethodInitialized:
		return nil, nil
	case MethodPing:
		return h.handlePing(ctx, msg)
	case MethodToolsList:
		return h.handleToolsList(ctx, msg)
	case MethodToolsCall:
		return h.handleToolsCall(ctx, msg)
	case MethodResourcesList:
		return h.handleResourcesList(ctx, msg)
	case MethodResourcesRead:
		return h.handleResourcesRead(ctx, msg)
	case MethodResourcesTemplates:
		return h.handleResourcesTemplates(ctx, msg)
	case MethodPromptsList:
		return h.handlePromptsList(ctx, msg)
	case MethodPromptsGet:
		return h.handlePromptsGet(ctx, msg)
	default:
		if msg.IsRequest() {
			return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.MethodNotFound,
				"method not found: "+msg.Method, nil)
		}
		return nil, nil
	}
}

func (h *Handler) handleInitialize(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	h.initialized = true

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools:     &ToolsCapability{},
			Resources: &ResourcesCapability{},
			Prompts:   &PromptsCapability{},
		},
		ServerInfo: Implementation{
			Name:    "lux",
			Version: "0.1.0",
		},
	}

	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePing(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	return jsonrpc.NewResponse(*msg.ID, PingResult{})
}

func (h *Handler) handleToolsList(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := ToolsListResult{
		Tools: h.server.tools.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleToolsCall(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params ToolCallParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	result, err := h.server.tools.Call(ctx, params.Name, params.Arguments)
	if err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InternalError, err.Error(), nil)
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleResourcesList(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := ResourcesListResult{
		Resources: h.server.resources.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleResourcesRead(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params ResourceReadParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	result, err := h.server.resources.Read(ctx, params.URI)
	if err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, err.Error(), nil)
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleResourcesTemplates(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := ResourceTemplatesListResult{
		ResourceTemplates: h.server.resources.ListTemplates(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePromptsList(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := PromptsListResult{
		Prompts: h.server.prompts.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePromptsGet(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params PromptGetParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	result, err := h.server.prompts.Get(params.Name, params.Arguments)
	if err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, err.Error(), nil)
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}
