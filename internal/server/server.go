package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/amarbel-llc/go-lib-mcp/jsonrpc"
	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/control"
	"github.com/amarbel-llc/lux/internal/lsp"
	"github.com/amarbel-llc/lux/internal/subprocess"
)

type Server struct {
	cfg         *config.Config
	pool        *subprocess.Pool
	router      *Router
	clientConn  *jsonrpc.Conn
	controlSrv  *control.Server
	initParams  *lsp.InitializeParams
	projectRoot string
	initialized bool
	mu          sync.RWMutex
	done        chan struct{}
}

func New(cfg *config.Config) (*Server, error) {
	router, err := NewRouter(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating router: %w", err)
	}

	s := &Server{
		cfg:    cfg,
		router: router,
		done:   make(chan struct{}),
	}

	executor := subprocess.NewNixExecutor()
	s.pool = subprocess.NewPool(executor, serverNotificationHandler(s))

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

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := NewHandler(s)
	s.clientConn = jsonrpc.NewConn(os.Stdin, os.Stdout, handler.Handle)

	controlSrv, err := control.NewServer(s.cfg.SocketPath(), s.pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not start control socket: %v\n", err)
	} else {
		s.controlSrv = controlSrv
		go s.controlSrv.Run(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.clientConn.Run(ctx)
	}()

	select {
	case err := <-errCh:
		s.shutdown()
		return err
	case <-ctx.Done():
		s.shutdown()
		return ctx.Err()
	case <-s.done:
		return nil
	}
}

func (s *Server) shutdown() {
	s.pool.StopAll()

	if s.controlSrv != nil {
		s.controlSrv.Close()
	}
}

func (s *Server) Close() {
	close(s.done)
}

func (s *Server) Pool() *subprocess.Pool {
	return s.pool
}

func (s *Server) Router() *Router {
	return s.router
}

func (s *Server) reloadPool(cfg *config.Config) error {
	s.cfg = cfg

	// Re-register all LSPs with updated config
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

	return nil
}
