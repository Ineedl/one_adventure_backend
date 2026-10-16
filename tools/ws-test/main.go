package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type connectRequest struct {
	ConnectionID string            `json:"connection_id,omitempty"`
	URL          string            `json:"url"`
	Token        string            `json:"token"`
	ServerID     int64             `json:"server_id"`
	ServerName   string            `json:"server_name,omitempty"`
	ChannelID    int64             `json:"channel_id"`
	ChannelIndex int               `json:"channel_index"`
	ChannelName  string            `json:"channel_name,omitempty"`
	DeviceID     string            `json:"device_id,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type sendRequest struct {
	ConnectionID string          `json:"connection_id"`
	Type         string          `json:"type"`
	Body         json.RawMessage `json:"body"`
	Timeout      string          `json:"timeout,omitempty"`
}

type wsRequest struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

type wsResponse struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	TraceID   string          `json:"trace_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data"`
}

type client struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	pending sync.Map // map[string]chan wsResponse
}

func main() {
	clients := sync.Map{}
	http.HandleFunc("POST /connect", func(w http.ResponseWriter, r *http.Request) {
		var req connectRequest
		if err := decodeJSON(r, &req); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		if req.URL == "" || req.Token == "" || req.ServerID <= 0 || req.ChannelID <= 0 {
			fail(w, http.StatusBadRequest, errors.New("url, token, server_id and channel_id are required"))
			return
		}
		endpoint, err := addConnectParams(req)
		if err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		headers := http.Header{"Authorization": {"Bearer " + req.Token}}
		conn, response, err := websocket.DefaultDialer.Dial(endpoint, headers)
		if err != nil {
			detail := err.Error()
			if response != nil {
				defer response.Body.Close()
				body, _ := io.ReadAll(response.Body)
				detail = fmt.Sprintf("%s (status=%s body=%s)", detail, response.Status, strings.TrimSpace(string(body)))
			}
			fail(w, http.StatusBadGateway, errors.New(detail))
			return
		}
		id := req.ConnectionID
		if id == "" {
			id = randomID()
		}
		c := &client{conn: conn}
		clients.Store(id, c)
		go c.readLoop(&clients, id)
		writeJSON(w, http.StatusCreated, map[string]any{"connection_id": id, "websocket_connection_id": response.Header.Get("X-WebSocket-Connection-ID")})
	})
	http.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		var req sendRequest
		if err := decodeJSON(r, &req); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		if req.ConnectionID == "" || !strings.HasSuffix(req.Type, "_REQ") || !json.Valid(req.Body) {
			fail(w, http.StatusBadRequest, errors.New("connection_id, a *_REQ type, and a valid JSON body are required"))
			return
		}
		value, ok := clients.Load(req.ConnectionID)
		if !ok {
			fail(w, http.StatusNotFound, errors.New("connection not found"))
			return
		}
		timeout := 10 * time.Second
		if req.Timeout != "" {
			if timeout, _ = time.ParseDuration(req.Timeout); timeout <= 0 {
				fail(w, http.StatusBadRequest, errors.New("invalid timeout"))
				return
			}
		}
		response, err := value.(*client).send(req.Type, req.Body, timeout)
		if err != nil {
			fail(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	log.Printf("ws test server listening on http://127.0.0.1:18081")
	log.Fatal(http.ListenAndServe("127.0.0.1:18081", nil))
}

func addConnectParams(req connectRequest) (string, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return "", err
	}
	params := map[string]any{"user_id": 0, "device_id": req.DeviceID, "platform": req.Platform, "server_info": map[string]any{"server_id": req.ServerID, "server_name": req.ServerName}, "channel_info": map[string]any{"channel_id": req.ChannelID, "channel_index": req.ChannelIndex, "channel_name": req.ChannelName}, "metadata": req.Metadata}
	encoded, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("connect_params", string(encoded))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *client) readLoop(clients *sync.Map, id string) {
	defer func() {
		clients.Delete(id)
		_ = c.conn.Close()
		c.pending.Range(func(_, value any) bool { close(value.(chan wsResponse)); return true })
	}()
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var response wsResponse
		if json.Unmarshal(payload, &response) == nil && response.SessionID != "" {
			if value, ok := c.pending.LoadAndDelete(response.SessionID); ok {
				value.(chan wsResponse) <- response
			}
		}
	}
}

func (c *client) send(typ string, body json.RawMessage, timeout time.Duration) (wsResponse, error) {
	sessionID := randomID()
	result := make(chan wsResponse, 1)
	c.pending.Store(sessionID, result)
	defer c.pending.Delete(sessionID)
	c.writeMu.Lock()
	err := c.conn.WriteJSON(wsRequest{Type: typ, SessionID: sessionID, Data: body})
	c.writeMu.Unlock()
	if err != nil {
		return wsResponse{}, err
	}
	select {
	case response, ok := <-result:
		if !ok {
			return wsResponse{}, errors.New("websocket connection closed")
		}
		return response, nil
	case <-time.After(timeout):
		return wsResponse{}, errors.New("websocket response timeout")
	}
}

func randomID() string {
	var bytes [16]byte
	_, _ = rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:])
}
func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
