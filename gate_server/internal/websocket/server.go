// Package websocket provides an independently configured WebSocket server for
// the GoFrame application.
package websocket

import (
	"context"
	"errors"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	tracekit "one_adventure_observability_trace/trace"
)

const serverName = "websocket"

// Server wraps the dedicated GoFrame server instance used for WebSocket
// connections.
type Server struct {
	server  *ghttp.Server
	manager *Manager
}

// New builds a WebSocket server from the application's YAML configuration.
func New(ctx context.Context) (*Server, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	manager := NewManager()
	handler := newHandler(cfg, manager)
	server := g.Server(serverName)
	server.SetPort(cfg.Port)
	server.SetDumpRouterMap(false)
	server.BindHandler(cfg.Path, func(r *ghttp.Request) {
		tracekit.HTTPMiddleware("gate_server", handler).ServeHTTP(r.Response.Writer, r.Request)
	})

	return &Server{server: server, manager: manager}, nil
}

// Manager returns the connection manager used by this server.
func (s *Server) Manager() *Manager {
	return s.manager
}

// Start starts the WebSocket listener without blocking.
func (s *Server) Start() error {
	return s.server.Start()
}

// Shutdown gracefully stops the WebSocket server.
func (s *Server) Shutdown() error {
	connectionsErr := s.manager.CloseAll()
	serverErr := s.server.Shutdown()
	return errors.Join(connectionsErr, serverErr)
}
