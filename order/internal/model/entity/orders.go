// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// Orders is the golang structure for table orders.
type Orders struct {
	OrderId      uint64 `json:"orderId"      orm:"order_id"      description:""`                  //
	UserId       uint64 `json:"userId"       orm:"user_id"       description:""`                  //
	ProductId    uint64 `json:"productId"    orm:"product_id"    description:""`                  //
	Quantity     int    `json:"quantity"     orm:"quantity"      description:""`                  //
	Amount       int64  `json:"amount"       orm:"amount"        description:"价格"`                // 价格
	CurrencyType int    `json:"currencyType" orm:"currency_type" description:"货币类型"`              // 货币类型
	OrderType    int    `json:"orderType"    orm:"order_type"    description:"订单类型"`              // 订单类型
	Status       int    `json:"status"       orm:"status"        description:"0:未完成，1:已完成，2:已取消"` // 0:未完成，1:已完成，2:已取消
	PromotionId  uint64 `json:"promotionId"  orm:"promotion_id"  description:"商品所属活动"`            // 商品所属活动
}
