package router

import (
	"context"
	"net/http"
	itempb "one_adventure_rpc/proto/item"
	paypb "one_adventure_rpc/proto/pay"
	promotionpb "one_adventure_rpc/proto/promotion"
	servermanagerpb "one_adventure_rpc/proto/ws_gateway"

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
	IsAdmin    bool
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
		{Service: "promotion", Version: "v1", Path: "stock-refresh"}: {
			Method:     http.MethodPost,
			IsAdmin:    true,
			NewRequest: func() any { return &promotionpb.PromotionStockRefreshReq{} },
			Invoke: func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error) {
				return promotionpb.NewPromotionServiceClient(connection).PromotionStockRefresh(ctx, request.(*promotionpb.PromotionStockRefreshReq))
			},
		},
		{Service: "promotion", Version: "v1", Path: "seckill"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &promotionpb.SeckillReq{} },
			Invoke: func(ctx context.Context, connection grpc.ClientConnInterface, request any) (any, error) {
				return promotionpb.NewPromotionServiceClient(connection).Seckill(ctx, request.(*promotionpb.SeckillReq))
			},
		},
		{Service: "pay", Version: "v1", Path: "create"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &paypb.CreatePaymentReq{} },
			Invoke: func(ctx context.Context, c grpc.ClientConnInterface, req any) (any, error) {
				return paypb.NewPayServiceClient(c).CreatePayment(ctx, req.(*paypb.CreatePaymentReq))
			},
		},
		{Service: "pay", Version: "v1", Path: "callback"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &paypb.ThirdPartyCallbackReq{} },
			Invoke: func(ctx context.Context, c grpc.ClientConnInterface, req any) (any, error) {
				return paypb.NewPayServiceClient(c).ThirdPartyCallback(ctx, req.(*paypb.ThirdPartyCallbackReq))
			},
		},
		{Service: "ws_gateway", Version: "v1", Path: "server_info"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &servermanagerpb.ServerInfoGetReq{} },
			Invoke: func(ctx context.Context, c grpc.ClientConnInterface, req any) (any, error) {
				return servermanagerpb.NewWsGatewayServiceClient(c).ServerInfoGet(ctx, req.(*servermanagerpb.ServerInfoGetReq))
			},
		},
		{Service: "ws_gateway", Version: "v1", Path: "channel_info"}: {
			Method:     http.MethodPost,
			NewRequest: func() any { return &servermanagerpb.ChannelInfoGetReq{} },
			Invoke: func(ctx context.Context, c grpc.ClientConnInterface, req any) (any, error) {
				return servermanagerpb.NewWsGatewayServiceClient(c).ChannelInfoGet(ctx, req.(*servermanagerpb.ChannelInfoGetReq))
			},
		},
	}
}
