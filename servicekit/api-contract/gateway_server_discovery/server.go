// Package gateway_server_discovery contains shared service-gateway_server_discovery key contracts.
package gateway_server_discovery

import "strconv"

// ServerKeyPrefix is the etcd prefix used by gate servers to claim logical servers.
const ServerKeyPrefix = "/server/"

func ServerKey(serverID uint64) string { return ServerKeyPrefix + strconv.FormatUint(serverID, 10) }

func ServerIDFromKey(key string) (uint64, bool) {
	if len(key) <= len(ServerKeyPrefix) || key[:len(ServerKeyPrefix)] != ServerKeyPrefix {
		return 0, false
	}
	id, err := strconv.ParseUint(key[len(ServerKeyPrefix):], 10, 64)
	return id, err == nil
}
