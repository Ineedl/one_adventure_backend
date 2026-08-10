package discovery

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	computingpb "one_adventure_rpc/proto/computing"
	itempb "one_adventure_rpc/proto/item"
	userpb "one_adventure_rpc/proto/user"
)

// ClientFactory is the central mapping from etcd server_name to its generated gRPC client.
type ClientFactory func(grpc.ClientConnInterface) any

var clientFactories = map[string]ClientFactory{
	"computing": func(connection grpc.ClientConnInterface) any {
		return computingpb.NewComputingServiceClient(connection)
	},
	"item": func(connection grpc.ClientConnInterface) any { return itempb.NewItemServiceClient(connection) },
	"user": func(connection grpc.ClientConnInterface) any { return userpb.NewUserServiceClient(connection) },
}

func (d *Discoverer) Client(serverName string) (any, error) {
	name := strings.ToLower(strings.TrimSpace(serverName))
	factory, ok := clientFactories[name]
	if !ok {
		return nil, fmt.Errorf("no grpc client factory for service %q", name)
	}
	connection, err := d.Connection(name)
	if err != nil {
		return nil, err
	}
	return factory(connection), nil
}
