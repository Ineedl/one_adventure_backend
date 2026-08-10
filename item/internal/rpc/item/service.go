package item

import itempb "one_adventure_rpc/proto/item"

type itemService struct {
	itempb.UnimplementedItemServiceServer
}

func newItemService() itempb.ItemServiceServer { return &itemService{} }
