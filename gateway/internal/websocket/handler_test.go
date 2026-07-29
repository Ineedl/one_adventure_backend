package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

func TestHandlerManagesConnectionAndHandlesClientRequest(t *testing.T) {
	manager := NewManager()
	handler := newHandler(testConfig(), manager)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, response, err := gorillaws.DefaultDialer.Dial(websocketURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	connectionID := response.Header.Get("X-WebSocket-Connection-ID")
	if connectionID == "" {
		t.Fatal("connection ID response header is empty")
	}
	waitFor(t, func() bool { return manager.Len() == 1 })
	if connection, exists := manager.Get(connectionID); !exists || connection.ID() != connectionID {
		t.Fatalf("Manager.Get(%q) = %#v, %v", connectionID, connection, exists)
	}

	if err = client.WriteJSON(Request{
		Type:      "PING_REQ",
		SessionID: "client-session",
		Data:      json.RawMessage(`{"message":"hello"}`),
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got Response
	if err = client.ReadJSON(&got); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if got.Type != "PING_RESP" || got.SessionID != "client-session" || got.Code != 0 || string(got.Data) != `{"message":"hello"}` {
		t.Fatalf("response = %#v", got)
	}

	if err = client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	waitFor(t, func() bool { return manager.Len() == 0 })
}

func TestManagerSendRequestMatchesResponseBySessionID(t *testing.T) {
	manager := NewManager()
	handler := newHandler(testConfig(), manager)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, response, err := gorillaws.DefaultDialer.Dial(websocketURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	connectionID := response.Header.Get("X-WebSocket-Connection-ID")
	waitFor(t, func() bool { return manager.Len() == 1 })

	type requestResult struct {
		response *Response
		err      error
	}
	resultCh := make(chan requestResult, 1)
	go func() {
		response, requestErr := manager.SendRequest(context.Background(), connectionID, Request{
			Type:      "SERVER_PUSH_REQ",
			SessionID: "server-session",
			Data:      json.RawMessage(`{"id":7}`),
		})
		resultCh <- requestResult{response: response, err: requestErr}
	}()

	var request Request
	if err = client.ReadJSON(&request); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if request.Type != "SERVER_PUSH_REQ" || request.SessionID != "server-session" {
		t.Fatalf("request = %#v", request)
	}

	// A valid response with another session must not complete this request.
	if err = client.WriteJSON(Response{Type: "SERVER_PUSH_RESP", SessionID: "another-session", Data: json.RawMessage(`null`)}); err != nil {
		t.Fatalf("WriteJSON(wrong session) error = %v", err)
	}
	select {
	case result := <-resultCh:
		t.Fatalf("request completed for a different session: %#v", result)
	case <-time.After(30 * time.Millisecond):
	}

	if err = client.WriteJSON(Response{
		Type:      "SERVER_PUSH_RESP",
		SessionID: "server-session",
		Code:      0,
		Data:      json.RawMessage(`{"accepted":true}`),
	}); err != nil {
		t.Fatalf("WriteJSON(response) error = %v", err)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("SendRequest() error = %v", result.err)
		}
		if result.response.SessionID != "server-session" || string(result.response.Data) != `{"accepted":true}` {
			t.Fatalf("SendRequest() response = %#v", result.response)
		}
	case <-time.After(time.Second):
		t.Fatal("SendRequest() did not receive matching response")
	}
}

func testConfig() WsConfig {
	return WsConfig{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		MaxMessageSize:  65536,
	}
}

func websocketURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
