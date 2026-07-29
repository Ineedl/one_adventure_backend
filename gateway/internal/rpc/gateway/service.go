package gateway

import "context"

const successMessage = "success"

// gatewayService is the initial implementation of GatewayService. Registry
// persistence can be connected here when the computing service is available.
type gatewayService struct {
	UnimplementedGatewayServiceServer
}

func newGatewayService() GatewayServiceServer {
	return &gatewayService{}
}

func (s *gatewayService) RegisterGateway(_ context.Context, _ *RegisterReq) (*RegisterResp, error) {
	return &RegisterResp{Code: 0, Message: successMessage}, nil
}

func (s *gatewayService) Heart(_ context.Context, _ *HeartReq) (*HeartResp, error) {
	return &HeartResp{Code: 0, Message: successMessage}, nil
}
