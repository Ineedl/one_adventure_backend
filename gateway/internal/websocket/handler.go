package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	gorillaws "github.com/gorilla/websocket"
)

// Handler upgrades HTTP requests and handles WebSocket messages.
type Handler struct {
	upgrader       gorillaws.Upgrader
	maxMessageSize int64
	manager        *Manager
}

func newHandler(cfg WsConfig, manager *Manager) *Handler {
	return &Handler{
		upgrader: gorillaws.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
		},
		maxMessageSize: cfg.MaxMessageSize,
		manager:        manager,
	}
}

// ServeHTTP upgrades and registers a managed WebSocket connection. The
// upgrader keeps Gorilla's default same-origin validation for browser clients.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connectionID := newIdentifier()
	responseHeader := http.Header{}
	responseHeader.Set("X-WebSocket-Connection-ID", connectionID)
	conn, err := h.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		g.Log().Warningf(r.Context(), "websocket upgrade failed: %v", err)
		return
	}
	connection := newConnection(r.Context(), connectionID, conn, h.manager.remove)
	h.manager.add(connection)
	defer connection.Close()

	conn.SetReadLimit(h.maxMessageSize)
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			var closeErr *gorillaws.CloseError
			if !errors.As(err, &closeErr) || (closeErr.Code != gorillaws.CloseNormalClosure && closeErr.Code != gorillaws.CloseGoingAway) {
				g.Log().Warningf(r.Context(), "websocket read failed: %v", err)
			}
			return
		}
		if messageType != gorillaws.TextMessage && messageType != gorillaws.BinaryMessage {
			continue
		}

		request, response, err := decodeMessage(payload)
		if err != nil {
			g.Log().Warningf(r.Context(), "invalid websocket message: %v", err)
			continue
		}
		if response != nil {
			if !connection.dispatchResponse(response) {
				g.Log().Debugf(r.Context(), "websocket response has no pending session: %s", response.SessionID)
			}
			continue
		}

		// A handler may itself send a request and wait for its response, so it
		// must not block this connection's only reader.
		go h.handleRequest(connection, request)
	}
}

func (h *Handler) handleRequest(connection *Connection, request *Request) {
	responseType, err := ResponseType(request.Type)
	if err != nil {
		return
	}

	handler := h.manager.requestHandler()
	response, handlerErr := handler(connection.Context(), connection, request)
	if response == nil {
		response = &Response{}
	}
	response.Type = responseType
	response.SessionID = request.SessionID
	if handlerErr != nil {
		response.Code = http.StatusInternalServerError
		response.Data, _ = json.Marshal(map[string]string{"error": handlerErr.Error()})
	}

	if err = connection.SendResponse(*response); err != nil && !errors.Is(err, ErrConnectionClosed) {
		g.Log().Warningf(context.Background(), "websocket response write failed: %v", err)
	}
}
