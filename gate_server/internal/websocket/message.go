package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	// RequestSuffix and ResponseSuffix identify the direction of a message.
	RequestSuffix  = "_REQ"
	ResponseSuffix = "_RESP"
)

var (
	ErrInvalidMessageType = errors.New("invalid websocket message type")
	ErrInvalidSessionID   = errors.New("websocket session_id is required")
)

// Request is the wire format of a WebSocket request.
type Request struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	TraceID   string          `json:"trace_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// Response is the wire format of a WebSocket response.
type Response struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	TraceID   string          `json:"trace_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data"`
}

// IsRequestType reports whether typeName represents a request.
func IsRequestType(typeName string) bool {
	return hasNonEmptyPrefix(typeName, RequestSuffix)
}

// IsResponseType reports whether typeName represents a response.
func IsResponseType(typeName string) bool {
	return hasNonEmptyPrefix(typeName, ResponseSuffix)
}

// ResponseType converts, for example, LOGIN_REQ to LOGIN_RESP.
func ResponseType(requestType string) (string, error) {
	if !IsRequestType(requestType) {
		return "", fmt.Errorf("%w: %q does not end in %s", ErrInvalidMessageType, requestType, RequestSuffix)
	}
	return strings.TrimSuffix(requestType, RequestSuffix) + ResponseSuffix, nil
}

func hasNonEmptyPrefix(typeName, suffix string) bool {
	return len(typeName) > len(suffix) && strings.HasSuffix(typeName, suffix)
}

func (r Request) validate() error {
	if !IsRequestType(r.Type) {
		return fmt.Errorf("%w: request type %q must end in %s", ErrInvalidMessageType, r.Type, RequestSuffix)
	}
	if r.SessionID == "" {
		return ErrInvalidSessionID
	}
	return nil
}

func (r Response) validate() error {
	if !IsResponseType(r.Type) {
		return fmt.Errorf("%w: response type %q must end in %s", ErrInvalidMessageType, r.Type, ResponseSuffix)
	}
	if r.SessionID == "" {
		return ErrInvalidSessionID
	}
	return nil
}

type messageEnvelope struct {
	Type string `json:"type"`
}

func decodeMessage(payload []byte) (*Request, *Response, error) {
	var envelope messageEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, nil, fmt.Errorf("decode websocket message: %w", err)
	}

	switch {
	case IsRequestType(envelope.Type):
		var request Request
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, nil, fmt.Errorf("decode websocket request: %w", err)
		}
		if err := request.validate(); err != nil {
			return nil, nil, err
		}
		return &request, nil, nil
	case IsResponseType(envelope.Type):
		var response Response
		if err := json.Unmarshal(payload, &response); err != nil {
			return nil, nil, fmt.Errorf("decode websocket response: %w", err)
		}
		if err := response.validate(); err != nil {
			return nil, nil, err
		}
		return nil, &response, nil
	default:
		return nil, nil, fmt.Errorf("%w: %q must end in %s or %s", ErrInvalidMessageType, envelope.Type, RequestSuffix, ResponseSuffix)
	}
}
