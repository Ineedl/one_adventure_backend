package server_manager

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	servermanagerpb "one_adventure_rpc/proto/server_manager"
)

func TestServerInfoGetEmptyBranches(t *testing.T) {
	service := &service{}
	for _, requestType := range []int32{0, 1, 2} {
		response, err := service.ServerInfoGet(context.Background(), &servermanagerpb.ServerInfoGetReq{Type: requestType})
		if err != nil {
			t.Fatalf("ServerInfoGet(type=%d) error = %v", requestType, err)
		}
		if response == nil || response.Servers == nil || len(response.Servers) != 0 {
			t.Fatalf("ServerInfoGet(type=%d) response = %#v, want a non-nil empty servers list", requestType, response)
		}
	}
}

func TestServerInfoGetRejectsInvalidType(t *testing.T) {
	service := &service{}
	_, err := service.ServerInfoGet(context.Background(), &servermanagerpb.ServerInfoGetReq{Type: 3})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ServerInfoGet() code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
