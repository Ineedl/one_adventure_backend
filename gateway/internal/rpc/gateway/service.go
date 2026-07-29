package gateway

import (
	"context"

	gatewaypb "one_adventure_rpc/proto/gateway"
)

const successMessage = "success"

// gatewayService is the initial implementation of GatewayService. Registry
// persistence can be connected here when the computing service is available.
type gatewayService struct {
	gatewaypb.UnimplementedGatewayServiceServer
}

func newGatewayService() gatewaypb.GatewayServiceServer {
	return &gatewayService{}
}

func (s *gatewayService) RegisterGateway(_ context.Context, _ *gatewaypb.RegisterReq) (*gatewaypb.RegisterResp, error) {
	return &gatewaypb.RegisterResp{Code: 0, Message: successMessage}, nil
}

func (s *gatewayService) Heart(_ context.Context, _ *gatewaypb.HeartReq) (*gatewaypb.HeartResp, error) {
	return &gatewaypb.HeartResp{Code: 0, Message: successMessage}, nil
}
