package websocket

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrConnectionNotFound = errors.New("websocket connection not found")

// RequestHandler handles a request received from a client. The returned
// response's type and session_id are normalized to match the request.
type RequestHandler func(context.Context, *Connection, *Request) (*Response, error)

// Manager owns all live WebSocket connections.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	handler     RequestHandler
}

// NewManager creates an empty connection manager.
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		handler:     echoRequestHandler,
	}
}

// SetRequestHandler replaces the handler for requests received from clients.
// Passing nil restores the default echo-data handler.
func (m *Manager) SetRequestHandler(handler RequestHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if handler == nil {
		handler = echoRequestHandler
	}
	m.handler = handler
}

// Get looks up a live connection by ID.
func (m *Manager) Get(connectionID string) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connection, exists := m.connections[connectionID]
	return connection, exists
}

// Connections returns a snapshot of all live connections.
func (m *Manager) Connections() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connections := make([]*Connection, 0, len(m.connections))
	for _, connection := range m.connections {
		connections = append(connections, connection)
	}
	return connections
}

// Len returns the number of live connections.
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// SendRequest sends a request through one connection and waits for its
// response.
func (m *Manager) SendRequest(ctx context.Context, connectionID string, request Request) (*Response, error) {
	connection, exists := m.Get(connectionID)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrConnectionNotFound, connectionID)
	}
	return connection.SendRequest(ctx, request)
}

// Close closes and removes one connection.
func (m *Manager) Close(connectionID string) error {
	connection, exists := m.Get(connectionID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrConnectionNotFound, connectionID)
	}
	return connection.Close()
}

// CloseAll closes every managed connection.
func (m *Manager) CloseAll() error {
	connections := m.Connections()
	var errs []error
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) add(connection *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[connection.ID()] = connection
}

func (m *Manager) remove(connection *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current := m.connections[connection.ID()]; current == connection {
		delete(m.connections, connection.ID())
	}
}

func (m *Manager) requestHandler() RequestHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handler
}

func echoRequestHandler(_ context.Context, _ *Connection, request *Request) (*Response, error) {
	return &Response{Data: request.Data}, nil
}
