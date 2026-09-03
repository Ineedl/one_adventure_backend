// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentOrder is the golang structure for table payment_order.
type PaymentOrder struct {
	Id         int64       `json:"id"         orm:"id"          description:"主键ID"`                                                                                                                                               // 主键ID
	PaymentNo  string      `json:"paymentNo"  orm:"payment_no"  description:"支付单号"`                                                                                                                                               // 支付单号
	OrderNo    string      `json:"orderNo"    orm:"order_no"    description:"业务订单号"`                                                                                                                                              // 业务订单号
	UserId     int64       `json:"userId"     orm:"user_id"     description:"用户ID"`                                                                                                                                               // 用户ID
	Amount     float64     `json:"amount"     orm:"amount"      description:"支付金额"`                                                                                                                                               // 支付金额
	Currency   string      `json:"currency"   orm:"currency"    description:"币种"`                                                                                                                                                 // 币种
	Status     string      `json:"status"     orm:"status"      description:"INIT	初始化	支付单刚创建 WAIT_PAY	等待支付	等待用户付款 PAYING	支付中	已发起第三方支付 SUCCESS	支付成功	支付完成 FAILED	支付失败	支付失败 CLOSED	已关闭	支付单失效 REFUNDING	退款中	退款处理中 REFUNDED	已退款	退款完成"` // INIT	初始化	支付单刚创建 WAIT_PAY	等待支付	等待用户付款 PAYING	支付中	已发起第三方支付 SUCCESS	支付成功	支付完成 FAILED	支付失败	支付失败 CLOSED	已关闭	支付单失效 REFUNDING	退款中	退款处理中 REFUNDED	已退款	退款完成
	Subject    string      `json:"subject"    orm:"subject"     description:"支付标题"`                                                                                                                                               // 支付标题
	ExpireTime *gtime.Time `json:"expireTime" orm:"expire_time" description:"支付过期时间"`                                                                                                                                             // 支付过期时间
	PaidTime   *gtime.Time `json:"paidTime"   orm:"paid_time"   description:"支付完成时间"`                                                                                                                                             // 支付完成时间
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"  description:"创建时间"`                                                                                                                                               // 创建时间
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"  description:"更新时间"`                                                                                                                                               // 更新时间
}
