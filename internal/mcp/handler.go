package mcp

import (
	"context"
	"encoding/json"
	"os"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/jsonrpc"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/lux/internal/warmup"
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
	case protocol.MethodInitialize:
		return h.handleInitialize(ctx, msg)
	case protocol.MethodInitialized:
		return nil, nil
	case protocol.MethodPing:
		return h.handlePing(ctx, msg)
	case protocol.MethodToolsList:
		return h.handleToolsList(ctx, msg)
	case protocol.MethodToolsCall:
		return h.handleToolsCall(ctx, msg)
	case protocol.MethodResourcesList:
		return h.handleResourcesList(ctx, msg)
	case protocol.MethodResourcesRead:
		return h.handleResourcesRead(ctx, msg)
	case protocol.MethodResourcesTemplates:
		return h.handleResourcesTemplates(ctx, msg)
	case protocol.MethodPromptsList:
		return h.handlePromptsList(ctx, msg)
	case protocol.MethodPromptsGet:
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
	var params protocol.InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	h.initialized = true

	go func() {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		warmup.StartAllLSPs(context.Background(), h.server.pool, h.server.cfg,
			warmup.SynthesizeInitParams(cwd))
	}()

	result := protocol.InitializeResult{
		ProtocolVersion: protocol.ProtocolVersion,
		Capabilities: protocol.ServerCapabilities{
			Tools:     &protocol.ToolsCapability{},
			Resources: &protocol.ResourcesCapability{Subscribe: true},
			Prompts:   &protocol.PromptsCapability{},
		},
		ServerInfo: protocol.Implementation{
			Name:    "lux",
			Version: "0.1.0",
		},
	}

	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePing(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	return jsonrpc.NewResponse(*msg.ID, protocol.PingResult{})
}

func (h *Handler) handleToolsList(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := protocol.ToolsListResult{
		Tools: h.server.tools.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleToolsCall(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params protocol.ToolCallParams
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
	result := protocol.ResourcesListResult{
		Resources: h.server.resources.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handleResourcesRead(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params protocol.ResourceReadParams
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
	result := protocol.ResourceTemplatesListResult{
		ResourceTemplates: h.server.resources.ListTemplates(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePromptsList(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	result := protocol.PromptsListResult{
		Prompts: h.server.prompts.List(),
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}

func (h *Handler) handlePromptsGet(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
	var params protocol.PromptGetParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, "invalid params", nil)
	}

	result, err := h.server.prompts.Get(params.Name, params.Arguments)
	if err != nil {
		return jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InvalidParams, err.Error(), nil)
	}
	return jsonrpc.NewResponse(*msg.ID, result)
}
