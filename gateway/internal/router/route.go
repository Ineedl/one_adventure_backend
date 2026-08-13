package router

import (
	"context"
	"net/http"
	itempb "one_adventure_rpc/proto/item"

	userpb "one_adventure_rpc/proto/user"

	"google.golang.org/grpc"
)

type RouteKey struct {
	Service string
	Version string
	Path    string
}

type Route struct {
	Method     string
	NewRequest func() any
	Invoke     func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error)
}

type RouteTable map[RouteKey]Route

func DefaultRouteTable() RouteTable {
	return RouteTable{
		{Service: "user", Version: "v1", Path: "login"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &userpb.LoginReq{} },
			Invoke: func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error) {
				return userpb.NewUserServiceClient(connection).Login(ctx, request.(*userpb.LoginReq))
			},
		},
		{Service: "user", Version: "v1", Path: "refresh-token"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &userpb.RefreshTokenReq{} },
			Invoke: func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error) {
				return userpb.NewUserServiceClient(connection).RefreshToken(ctx, request.(*userpb.RefreshTokenReq))
			},
		},
		{Service: "item", Version: "v1", Path: "inventory-conf"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &itempb.InventoryConfGetReq{} },
			Invoke: func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error) {
				return itempb.NewItemServiceClient(connection).InventoryConfGet(ctx, request.(*itempb.InventoryConfGetReq))
			},
		},
	}
}
