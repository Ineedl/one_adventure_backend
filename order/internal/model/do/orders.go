// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
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
	Status       interface{} // 0:未完成，1:已完成，2:已取消
	PromotionId  interface{} // 商品所属活动
}
