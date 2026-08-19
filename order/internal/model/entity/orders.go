// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Orders is the golang structure for table orders.
type Orders struct {
	OrderId      uint64      `json:"orderId"      orm:"order_id"      description:""`                                                                                                                                                                                                 //
	UserId       uint64      `json:"userId"       orm:"user_id"       description:""`                                                                                                                                                                                                 //
	ProductId    uint64      `json:"productId"    orm:"product_id"    description:""`                                                                                                                                                                                                 //
	Quantity     int         `json:"quantity"     orm:"quantity"      description:""`                                                                                                                                                                                                 //
	Amount       int64       `json:"amount"       orm:"amount"        description:"价格"`                                                                                                                                                                                               // 价格
	CurrencyType int         `json:"currencyType" orm:"currency_type" description:"货币类型"`                                                                                                                                                                                             // 货币类型
	OrderType    int         `json:"orderType"    orm:"order_type"    description:"订单类型"`                                                                                                                                                                                             // 订单类型
	Status       string      `json:"status"       orm:"status"        description:"PENDING_PAY	待支付	订单创建成功，等待用户付款 PAID	已支付	支付成功，等待履约 PROCESSING	处理中	支付完成，订单正在处理（可选） SHIPPED	已发货	物流已发出 COMPLETED	已完成	交易结束 CANCELLED	已取消	订单关闭 REFUNDING	退款中	正在退款 REFUNDED	已退款	退款完成 CLOSED	已关闭	订单生命周期结束"` // PENDING_PAY	待支付	订单创建成功，等待用户付款 PAID	已支付	支付成功，等待履约 PROCESSING	处理中	支付完成，订单正在处理（可选） SHIPPED	已发货	物流已发出 COMPLETED	已完成	交易结束 CANCELLED	已取消	订单关闭 REFUNDING	退款中	正在退款 REFUNDED	已退款	退款完成 CLOSED	已关闭	订单生命周期结束
	PromotionId  uint64      `json:"promotionId"  orm:"promotion_id"  description:"商品所属活动"`                                                                                                                                                                                           // 商品所属活动
	CreateTime   *gtime.Time `json:"createTime"   orm:"create_time"   description:""`                                                                                                                                                                                                 //
	UpdateTime   *gtime.Time `json:"updateTime"   orm:"update_time"   description:""`                                                                                                                                                                                                 //
	RequestId    string      `json:"requestId"    orm:"request_id"    description:""`                                                                                                                                                                                                 //
	OrderNo      string      `json:"orderNo"      orm:"order_no"      description:"订单号"`                                                                                                                                                                                              // 订单号
}
