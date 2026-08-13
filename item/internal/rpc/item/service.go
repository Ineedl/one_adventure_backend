package item

import (
	"context"
	"item/internal/dao"
	"item/internal/model/entity"
	itempb "one_adventure_rpc/proto/item"
	"one_adventure_servicekit/token"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type itemService struct {
	itempb.UnimplementedItemServiceServer
}

func newItemService() itempb.ItemServiceServer { return &itemService{} }

func (itemService) InventoryConfGet(ctx context.Context, _ *itempb.InventoryConfGetReq) (*itempb.InventoryConfGetResp, error) {
	userInfo, ok := token.UserInfoFromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user info is required")
	}

	var inventoryConf entity.InventoryConf
	columns := dao.InventoryConf.Columns()
	if err := dao.InventoryConf.Ctx(ctx).
		Where(columns.UserId, userInfo.ID).
		Scan(&inventoryConf); err != nil {
		return nil, status.Error(codes.Internal, "query inventory config failed")
	}
	return &itempb.InventoryConfGetResp{ItemNum: uint32(inventoryConf.ItemNum)}, nil
}
