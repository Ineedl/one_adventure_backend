package xds

import (
	"testing"

	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

func TestLocalityLbEndpointsGroupsEndpointsByLocality(t *testing.T) {
	defaultEndpoint := &endpoint.LbEndpoint{}
	zoneAEndpoint1 := &endpoint.LbEndpoint{}
	zoneAEndpoint2 := &endpoint.LbEndpoint{}
	zoneBEndpoint := &endpoint.LbEndpoint{}

	groups := localityLbEndpoints(map[localityKey][]*endpoint.LbEndpoint{
		{}:                                 {defaultEndpoint},
		{region: "cn", zone: "shanghai-a"}: {zoneAEndpoint1, zoneAEndpoint2},
		{region: "cn", zone: "shanghai-b"}: {zoneBEndpoint},
	})

	if len(groups) != 3 {
		t.Fatalf("got %d locality groups, want 3", len(groups))
	}
	if groups[0].Locality != nil || len(groups[0].LbEndpoints) != 1 {
		t.Fatalf("default locality group = %#v, want one endpoint and no locality metadata", groups[0])
	}
	if got := groups[1].Locality; got == nil || got.Region != "cn" || got.Zone != "shanghai-a" {
		t.Fatalf("first named locality = %#v, want cn/shanghai-a", got)
	}
	if len(groups[1].LbEndpoints) != 2 {
		t.Fatalf("cn/shanghai-a has %d endpoints, want 2", len(groups[1].LbEndpoints))
	}
	if got := groups[2].Locality; got == nil || got.Region != "cn" || got.Zone != "shanghai-b" {
		t.Fatalf("second named locality = %#v, want cn/shanghai-b", got)
	}
}

func TestLocalityLbEndpointsUsesOneDefaultGroup(t *testing.T) {
	groups := localityLbEndpoints(map[localityKey][]*endpoint.LbEndpoint{
		{}: {{}, {}},
	})

	if len(groups) != 1 {
		t.Fatalf("got %d locality groups, want 1", len(groups))
	}
	if len(groups[0].LbEndpoints) != 2 {
		t.Fatalf("default locality has %d endpoints, want 2", len(groups[0].LbEndpoints))
	}
}
