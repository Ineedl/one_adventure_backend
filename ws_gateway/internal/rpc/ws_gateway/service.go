package ws_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	servermanagerpb "one_adventure_rpc/proto/ws_gateway"
	"one_adventure_servicekit/api-contract/gateway_server_discovery"
	"ws_gateway/internal/dao"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type service struct {
	servermanagerpb.UnimplementedWsGatewayServiceServer
}

type registrationInfo struct {
	Address    string `json:"address"`
	WSPort     int    `json:"ws_port"`
	InstanceID string `json:"instance_id"`
}

func newServerManagerService() servermanagerpb.WsGatewayServiceServer {
	return &service{}
}

func (s *service) ServerInfoGet(ctx context.Context, request *servermanagerpb.ServerInfoGetReq) (*servermanagerpb.ServerInfoGetResp, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	switch request.GetType() {
	case 0:
		return s.allAvailableServers(ctx)
	case 1:
		return s.unplayedServers(ctx)
	case 2:
		return s.playedServers(ctx)
	default:
		return nil, status.Error(codes.InvalidArgument, "type must be 0, 1 or 2")
	}
}

func (s *service) ChannelInfoGet(ctx context.Context, request *servermanagerpb.ChannelInfoGetReq) (*servermanagerpb.ChannelInfoGetResp, error) {
	if request == nil || request.GetServerId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}
	var rows []struct {
		Id            uint64
		Index         int
		Name          string
		ServerId      uint64
		WsConnections int64
	}
	if err := dao.ChannelInfo.Ctx(ctx).Fields("id,index,name,server_id,ws_connections").Where("server_id", request.GetServerId()).OrderAsc("index").Scan(&rows); err != nil {
		return nil, err
	}
	resp := &servermanagerpb.ChannelInfoGetResp{}
	for _, r := range rows {
		resp.Channels = append(resp.Channels, &servermanagerpb.ChannelInfo{Id: r.Id, Index: int32(r.Index), Name: r.Name, ServerId: r.ServerId, WsConnections: r.WsConnections})
	}
	return resp, nil
}

// allAvailableServers returns all available logical servers.
func (*service) allAvailableServers(ctx context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return listServers(ctx, 0)
}

// unplayedServers returns logical servers on which the user has no character.
func (*service) unplayedServers(ctx context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return listServers(ctx, 1)
}

// playedServers returns logical servers on which the user has a character.
func (*service) playedServers(ctx context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return listServers(ctx, 2)
}

func listServers(ctx context.Context, filter int32) (*servermanagerpb.ServerInfoGetResp, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var rows []struct {
		Id   uint64
		Name string
	}
	if err := dao.ServerInfo.Ctx(queryCtx).Fields("id,name").OrderAsc("id").Scan(&rows); err != nil {
		return nil, err
	}
	endpoints := []string{"etcd:2379"}
	if value := strings.TrimSpace(os.Getenv("ETCD_ENDPOINTS")); value != "" {
		endpoints = strings.Split(value, ",")
	}
	cli, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}
	defer cli.Close()
	resp, err := cli.Get(queryCtx, gateway_server_discovery.ServerKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	occupied := map[uint64]registrationInfo{}
	for _, kv := range resp.Kvs {
		var info registrationInfo
		_ = json.Unmarshal(kv.Value, &info)
		if id, ok := gateway_server_discovery.ServerIDFromKey(string(kv.Key)); ok {
			occupied[id] = info
		}
	}
	out := map[uint64]*servermanagerpb.ServerInfo{}
	for _, r := range rows {
		s := out[r.Id]
		if s == nil {
			s = &servermanagerpb.ServerInfo{Id: r.Id, Name: r.Name}
			out[r.Id] = s
			info, ok := occupied[r.Id]
			s.Occupied = ok
			s.Online = ok
			if ok {
				s.InstanceId = info.InstanceID
				s.Address = info.Address
				s.WsPort = int32(info.WSPort)
			}
		}
	}
	for id, s := range out {
		remove := (filter == 1 && s.Occupied) || (filter == 2 && !s.Occupied)
		if remove {
			delete(out, id)
		}
	}
	result := &servermanagerpb.ServerInfoGetResp{}
	ids := make([]uint64, 0, len(out))
	for id := range out {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		result.Servers = append(result.Servers, out[id])
	}
	return result, nil
}
