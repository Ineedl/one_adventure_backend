package computing

import (
	"context"

	computingpb "one_adventure_rpc/proto/computing"
)

// computingService implements the computing gRPC contract. Collision logic
// can be connected here as the calculation model is defined.
type computingService struct {
	computingpb.UnimplementedComputingServiceServer
}

func newComputingService() computingpb.ComputingServiceServer {
	return &computingService{}
}

func (s *computingService) CollisionCalculation(_ context.Context, _ *computingpb.CollisionCalculationReq) (*computingpb.CollisionCalculationResp, error) {
	return &computingpb.CollisionCalculationResp{}, nil
}
