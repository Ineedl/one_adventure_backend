package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const ConnectParamsQueryKey = "connect_params"

var ErrConnectParamsRequired = errors.New("websocket connect_params is required")

// WsConnectParams contains application-defined parameters supplied during the
// WebSocket handshake. Add strongly typed fields here as the connection
// protocol evolves; Metadata is reserved for forward-compatible values.

type ChannelInfo struct {
	ChannelId    int64  `json:"channel_id"`
	ChannelIndex int    `json:"channel_index"`
	ChannelName  string `json:"channel_name"`
}

type ServerInfo struct {
	ServerId   int64  `json:"server_id"`
	ServerName string `json:"server_name"`
}

type WsConnectParams struct {
	UserID      uint64            `json:"user_id,omitempty"`
	DeviceID    string            `json:"device_id,omitempty"`
	Platform    string            `json:"platform,omitempty"`
	ServerInfo  ServerInfo        `json:"server_name,omitempty"`
	ChannelInfo ChannelInfo       `json:"channel_info,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func parseConnectParams(request *http.Request) (WsConnectParams, error) {
	raw := request.URL.Query().Get(ConnectParamsQueryKey)
	if raw == "" {
		return WsConnectParams{}, ErrConnectParamsRequired
	}
	var params WsConnectParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return WsConnectParams{}, fmt.Errorf("decode websocket connect_params: %w", err)
	}
	return params, nil
}
