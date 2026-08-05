package ping

import (
	"context"
	"testing"

	pingpb "one_adventure_rpc/proto/ping"
)

func TestServiceAcceptsOnlyActiveRegistration(t *testing.T) {
	service, err := New("Computing", "computing-1")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	service.Activate("current-token")

	response, err := service.Ping(context.Background(), &pingpb.PingReq{
		Type:              "computing",
		InstanceId:        "computing-1",
		RegistrationToken: "current-token",
	})
	if err != nil || response.GetCode() != responseCodeSuccess {
		t.Fatalf("Ping(active) response = %#v, error = %v", response, err)
	}
	select {
	case <-service.Activity():
	default:
		t.Fatal("valid Ping did not report activity")
	}

	response, err = service.Ping(context.Background(), &pingpb.PingReq{
		Type:              "computing",
		InstanceId:        "computing-1",
		RegistrationToken: "stale-token",
	})
	if err != nil || response.GetCode() != responseCodeNotFound {
		t.Fatalf("Ping(stale) response = %#v, error = %v", response, err)
	}

	service.Deactivate("current-token")
	response, err = service.Ping(context.Background(), &pingpb.PingReq{
		Type:              "computing",
		InstanceId:        "computing-1",
		RegistrationToken: "current-token",
	})
	if err != nil || response.GetCode() != responseCodeNotFound {
		t.Fatalf("Ping(inactive) response = %#v, error = %v", response, err)
	}
}
