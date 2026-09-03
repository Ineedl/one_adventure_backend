// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentTransaction is the golang structure of table payment_transaction for DAO operations like Where/Data.
type PaymentTransaction struct {
	g.Meta          `orm:"table:payment_transaction, do:true"`
	Id              interface{} // 主键ID
	PaymentNo       interface{} // 支付单号
	OrderNo         interface{} // 业务订单号(冗余)
	TransactionNo   interface{} // 第三方交易号
	Channel         interface{} // 支付渠道
	RequestAmount   interface{} // 请求支付金额
	PaidAmount      interface{} // 实际支付金额
	Status          interface{} // 交易状态 CREATED	已创建	准备请求第三方 PROCESSING	处理中	第三方处理中 SUCCESS	成功	第三方确认成功 FAILED	失败	交易失败 CLOSED	关闭	交易关闭 REFUNDING	退款中	退款处理中 REFUNDED	退款成功	资金已退回
	CallbackTime    *gtime.Time // 第三方回调时间
	CallbackContent interface{} // 第三方回调原始内容
	RequestContent  interface{} // 请求第三方参数
	ResponseContent interface{} // 第三方响应内容
	CreatedAt       *gtime.Time // 创建时间
	UpdatedAt       *gtime.Time // 更新时间
}
