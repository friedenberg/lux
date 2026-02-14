package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/amarbel-llc/go-lib-mcp/jsonrpc"
	"github.com/amarbel-llc/go-lib-mcp/transport"
	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/formatter"
	"github.com/amarbel-llc/lux/internal/server"
	"github.com/amarbel-llc/lux/internal/subprocess"
)

type Server struct {
	cfg       *config.Config
	transport transport.Transport
	handler   *Handler
	pool      *subprocess.Pool
	router    *server.Router
	bridge    *Bridge
	tools     *ToolRegistry
	resources *ResourceRegistry
	prompts   *PromptRegistry
	done      chan struct{}
	wg        sync.WaitGroup
}

func New(cfg *config.Config, t transport.Transport) (*Server, error) {
	router, err := server.NewRouter(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating router: %w", err)
	}

	s := &Server{
		cfg:       cfg,
		transport: t,
		router:    router,
		done:      make(chan struct{}),
	}

	executor := subprocess.NewNixExecutor()
	s.pool = subprocess.NewPool(executor, s.lspNotificationHandler())

	for _, l := range cfg.LSPs {
		// Convert config.CapabilityOverride to subprocess.CapabilityOverride
		var capOverrides *subprocess.CapabilityOverride
		if l.Capabilities != nil {
			capOverrides = &subprocess.CapabilityOverride{
				Disable: l.Capabilities.Disable,
				Enable:  l.Capabilities.Enable,
			}
		}
		s.pool.Register(l.Name, l.Flake, l.Binary, l.Args, l.Env, l.InitOptions, capOverrides)
	}

	var fmtRouter *formatter.Router
	fmtCfg, err := config.LoadMergedFormatters()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load formatter config: %v\n", err)
	} else {
		fmtRouter, err = formatter.NewRouter(fmtCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create formatter router: %v\n", err)
			fmtRouter = nil
		}
	}

	s.bridge = NewBridge(s.pool, s.router, fmtRouter, executor)
	s.tools = NewToolRegistry(s.bridge)
	s.resources = NewResourceRegistry(s.pool, s.bridge, cfg)
	s.prompts = NewPromptRegistry()
	s.handler = NewHandler(s)
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			s.gracefulShutdown()
			return ctx.Err()
		case <-s.done:
			s.gracefulShutdown()
			return nil
		default:
		}

		msg, err := s.transport.Read()
		if err != nil {
			// EOF signals graceful shutdown from client
			if err == io.EOF {
				s.gracefulShutdown()
				return nil
			}
			s.gracefulShutdown()
			return fmt.Errorf("reading message: %w", err)
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleMessage(ctx, msg)
		}()
	}
}

func (s *Server) handleMessage(ctx context.Context, msg *jsonrpc.Message) {
	resp, err := s.handler.Handle(ctx, msg)
	if err != nil {
		if msg.IsRequest() {
			errResp, _ := jsonrpc.NewErrorResponse(*msg.ID, jsonrpc.InternalError, err.Error(), nil)
			s.transport.Write(errResp)
		}
		return
	}

	if resp != nil {
		s.transport.Write(resp)
	}
}

func (s *Server) gracefulShutdown() {
	// Wait for all in-flight requests to complete
	s.wg.Wait()
	s.pool.StopAll()
	s.transport.Close()
}

func (s *Server) Close() {
	close(s.done)
}

func (s *Server) lspNotificationHandler() jsonrpc.Handler {
	return func(ctx context.Context, msg *jsonrpc.Message) (*jsonrpc.Message, error) {
		// For now, we ignore notifications from LSP servers
		// In the future, we could forward diagnostics as MCP resource updates
		return nil, nil
	}
}
