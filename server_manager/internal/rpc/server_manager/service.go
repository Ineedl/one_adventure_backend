package server_manager

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	servermanagerpb "one_adventure_rpc/proto/server_manager"
)

type service struct {
	servermanagerpb.UnimplementedServerManagerServiceServer
}

func newServerManagerService() servermanagerpb.ServerManagerServiceServer {
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

// allAvailableServers returns all available logical servers.
func (*service) allAvailableServers(context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return &servermanagerpb.ServerInfoGetResp{Servers: []*servermanagerpb.ServerInfo{}}, nil
}

// unplayedServers returns logical servers on which the user has no character.
func (*service) unplayedServers(context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return &servermanagerpb.ServerInfoGetResp{Servers: []*servermanagerpb.ServerInfo{}}, nil
}

// playedServers returns logical servers on which the user has a character.
func (*service) playedServers(context.Context) (*servermanagerpb.ServerInfoGetResp, error) {
	return &servermanagerpb.ServerInfoGetResp{Servers: []*servermanagerpb.ServerInfo{}}, nil
}
