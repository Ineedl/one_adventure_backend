package websocket

type ConnectChannelInfo struct {
	ChannelId    int64  `json:"channel_id"`
	ChannelIndex int    `json:"channel_index"`
	ChannelName  string `json:"channel_name"`
}
type ConnectServerInfo struct {
	ServerId   int64  `json:"server_id"`
	ServerName string `json:"server_name"`
}
type WsConnectParams struct {
	UserID      uint64             `json:"user_id,omitempty"`
	DeviceID    string             `json:"device_id,omitempty"`
	Platform    string             `json:"platform,omitempty"`
	ServerInfo  ConnectServerInfo  `json:"server_name,omitempty"`
	ChannelInfo ConnectChannelInfo `json:"channel_info,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}
