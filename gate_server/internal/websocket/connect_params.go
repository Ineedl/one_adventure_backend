package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	contract "one_adventure_servicekit/api-contract/websocket"
)

const ConnectParamsQueryKey = "connect_params"

var ErrConnectParamsRequired = errors.New("websocket connect_params is required")

// WsConnectParams contains application-defined parameters supplied during the
// WebSocket handshake. Add strongly typed fields here as the connection
// protocol evolves; Metadata is reserved for forward-compatible values.

type WsConnectParams = contract.WsConnectParams
type ConnectChannelInfo = contract.ConnectChannelInfo
type ConnectServerInfo = contract.ConnectServerInfo

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
