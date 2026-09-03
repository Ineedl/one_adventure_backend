// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentOrder is the golang structure of table payment_order for DAO operations like Where/Data.
type PaymentOrder struct {
	g.Meta     `orm:"table:payment_order, do:true"`
	Id         interface{} // 主键ID
	PaymentNo  interface{} // 支付单号
	OrderNo    interface{} // 业务订单号
	UserId     interface{} // 用户ID
	Amount     interface{} // 支付金额
	Currency   interface{} // 币种
	Status     interface{} // INIT	初始化	支付单刚创建 WAIT_PAY	等待支付	等待用户付款 PAYING	支付中	已发起第三方支付 SUCCESS	支付成功	支付完成 FAILED	支付失败	支付失败 CLOSED	已关闭	支付单失效 REFUNDING	退款中	退款处理中 REFUNDED	已退款	退款完成
	Subject    interface{} // 支付标题
	ExpireTime *gtime.Time // 支付过期时间
	PaidTime   *gtime.Time // 支付完成时间
	CreatedAt  *gtime.Time // 创建时间
	UpdatedAt  *gtime.Time // 更新时间
}
