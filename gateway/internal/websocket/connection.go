package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	gorillaws "github.com/gorilla/websocket"
)

var (
	ErrConnectionClosed = errors.New("websocket connection is closed")
	ErrSessionPending   = errors.New("websocket session_id is already pending")
)

type responseResult struct {
	response *Response
	err      error
}

// Connection is one managed WebSocket connection. It is safe for concurrent
// calls to SendRequest and SendResponse.
type Connection struct {
	id     string
	conn   *gorillaws.Conn
	ctx    context.Context
	cancel context.CancelFunc

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan responseResult

	closeOnce sync.Once
	closed    chan struct{}
	onClose   func(*Connection)
}

func newConnection(parent context.Context, id string, conn *gorillaws.Conn, onClose func(*Connection)) *Connection {
	ctx, cancel := context.WithCancel(parent)
	return &Connection{
		id:      id,
		conn:    conn,
		ctx:     ctx,
		cancel:  cancel,
		pending: make(map[string]chan responseResult),
		closed:  make(chan struct{}),
		onClose: onClose,
	}
}

// ID returns the server-generated identifier of the connection.
func (c *Connection) ID() string {
	return c.id
}

// Context is canceled when the connection closes.
func (c *Connection) Context() context.Context {
	return c.ctx
}

// SendRequest writes a request and waits for a response carrying the same
// session_id. A session_id is generated when request.SessionID is empty.
func (c *Connection) SendRequest(ctx context.Context, request Request) (*Response, error) {
	if !IsRequestType(request.Type) {
		return nil, fmt.Errorf("%w: request type %q must end in %s", ErrInvalidMessageType, request.Type, RequestSuffix)
	}
	if request.SessionID == "" {
		request.SessionID = newIdentifier()
	}

	resultCh := make(chan responseResult, 1)
	if err := c.addPending(request.SessionID, resultCh); err != nil {
		return nil, err
	}
	defer c.removePending(request.SessionID, resultCh)

	if err := c.writeJSON(request); err != nil {
		return nil, err
	}

	select {
	case result := <-resultCh:
		return result.response, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, ErrConnectionClosed
	}
}

// SendResponse writes a response to the connection.
func (c *Connection) SendResponse(response Response) error {
	if err := response.validate(); err != nil {
		return err
	}
	return c.writeJSON(response)
}

func (c *Connection) writeJSON(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	select {
	case <-c.closed:
		return ErrConnectionClosed
	default:
	}

	if err := c.conn.WriteJSON(value); err != nil {
		_ = c.Close()
		return fmt.Errorf("write websocket message: %w", err)
	}
	return nil
}

func (c *Connection) addPending(sessionID string, resultCh chan responseResult) error {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	select {
	case <-c.closed:
		return ErrConnectionClosed
	default:
	}
	if _, exists := c.pending[sessionID]; exists {
		return fmt.Errorf("%w: %s", ErrSessionPending, sessionID)
	}
	c.pending[sessionID] = resultCh
	return nil
}

func (c *Connection) removePending(sessionID string, resultCh chan responseResult) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	if current, exists := c.pending[sessionID]; exists && current == resultCh {
		delete(c.pending, sessionID)
	}
}

func (c *Connection) dispatchResponse(response *Response) bool {
	c.pendingMu.Lock()
	resultCh, exists := c.pending[response.SessionID]
	if exists {
		delete(c.pending, response.SessionID)
	}
	c.pendingMu.Unlock()

	if exists {
		resultCh <- responseResult{response: response}
	}
	return exists
}

// Close closes the socket, cancels all pending requests, and unregisters the
// connection from its manager. It is safe to call more than once.
func (c *Connection) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		close(c.closed)
		c.cancel()
		closeErr = c.conn.Close()

		c.pendingMu.Lock()
		pending := c.pending
		c.pending = make(map[string]chan responseResult)
		c.pendingMu.Unlock()
		for _, resultCh := range pending {
			resultCh <- responseResult{err: ErrConnectionClosed}
		}

		if c.onClose != nil {
			c.onClose(c)
		}
	})
	return closeErr
}

var fallbackIdentifier atomic.Uint64

func newIdentifier() string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err == nil {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("fallback-%d", fallbackIdentifier.Add(1))
}
