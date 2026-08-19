// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Orders is the golang structure of table orders for DAO operations like Where/Data.
type Orders struct {
	g.Meta       `orm:"table:orders, do:true"`
	OrderId      interface{} //
	UserId       interface{} //
	ProductId    interface{} //
	Quantity     interface{} //
	Amount       interface{} // 价格
	CurrencyType interface{} // 货币类型
	OrderType    interface{} // 订单类型
	Status       interface{} // PENDING_PAY	待支付	订单创建成功，等待用户付款 PAID	已支付	支付成功，等待履约 PROCESSING	处理中	支付完成，订单正在处理（可选） SHIPPED	已发货	物流已发出 COMPLETED	已完成	交易结束 CANCELLED	已取消	订单关闭 REFUNDING	退款中	正在退款 REFUNDED	已退款	退款完成 CLOSED	已关闭	订单生命周期结束
	PromotionId  interface{} // 商品所属活动
	CreateTime   *gtime.Time //
	UpdateTime   *gtime.Time //
	RequestId    interface{} //
	OrderNo      interface{} // 订单号
}
