package websocket

import (
	"errors"
	"testing"
)

func TestDecodeMessage(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		request, response, err := decodeMessage([]byte(`{"type":"PING_REQ","session_id":"s1","data":{"value":1}}`))
		if err != nil {
			t.Fatalf("decodeMessage() error = %v", err)
		}
		if response != nil {
			t.Fatalf("decodeMessage() response = %#v, want nil", response)
		}
		if request.Type != "PING_REQ" || request.SessionID != "s1" || string(request.Data) != `{"value":1}` {
			t.Fatalf("decodeMessage() request = %#v", request)
		}
	})

	t.Run("response", func(t *testing.T) {
		request, response, err := decodeMessage([]byte(`{"type":"PING_RESP","session_id":"s1","code":0,"data":[1,2]}`))
		if err != nil {
			t.Fatalf("decodeMessage() error = %v", err)
		}
		if request != nil {
			t.Fatalf("decodeMessage() request = %#v, want nil", request)
		}
		if response.Type != "PING_RESP" || response.SessionID != "s1" || response.Code != 0 || string(response.Data) != `[1,2]` {
			t.Fatalf("decodeMessage() response = %#v", response)
		}
	})

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{name: "invalid json", payload: `{`},
		{name: "unknown suffix", payload: `{"type":"PING","session_id":"s1","data":null}`, wantErr: ErrInvalidMessageType},
		{name: "empty type prefix", payload: `{"type":"_REQ","session_id":"s1","data":null}`, wantErr: ErrInvalidMessageType},
		{name: "missing session", payload: `{"type":"PING_REQ","data":null}`, wantErr: ErrInvalidSessionID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := decodeMessage([]byte(test.payload))
			if err == nil {
				t.Fatal("decodeMessage() error = nil")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("decodeMessage() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestResponseType(t *testing.T) {
	got, err := ResponseType("LOGIN_REQ")
	if err != nil {
		t.Fatalf("ResponseType() error = %v", err)
	}
	if got != "LOGIN_RESP" {
		t.Fatalf("ResponseType() = %q, want LOGIN_RESP", got)
	}
	if _, err = ResponseType("LOGIN"); !errors.Is(err, ErrInvalidMessageType) {
		t.Fatalf("ResponseType() error = %v, want ErrInvalidMessageType", err)
	}
}
