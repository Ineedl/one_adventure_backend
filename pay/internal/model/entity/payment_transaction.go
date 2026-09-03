// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentTransaction is the golang structure for table payment_transaction.
type PaymentTransaction struct {
	Id              int64       `json:"id"              orm:"id"               description:"主键ID"`                                                                                                                                    // 主键ID
	PaymentNo       string      `json:"paymentNo"       orm:"payment_no"       description:"支付单号"`                                                                                                                                    // 支付单号
	OrderNo         string      `json:"orderNo"         orm:"order_no"         description:"业务订单号(冗余)"`                                                                                                                               // 业务订单号(冗余)
	TransactionNo   string      `json:"transactionNo"   orm:"transaction_no"   description:"第三方交易号"`                                                                                                                                  // 第三方交易号
	Channel         string      `json:"channel"         orm:"channel"          description:"支付渠道"`                                                                                                                                    // 支付渠道
	RequestAmount   float64     `json:"requestAmount"   orm:"request_amount"   description:"请求支付金额"`                                                                                                                                  // 请求支付金额
	PaidAmount      float64     `json:"paidAmount"      orm:"paid_amount"      description:"实际支付金额"`                                                                                                                                  // 实际支付金额
	Status          int         `json:"status"          orm:"status"           description:"交易状态 CREATED	已创建	准备请求第三方 PROCESSING	处理中	第三方处理中 SUCCESS	成功	第三方确认成功 FAILED	失败	交易失败 CLOSED	关闭	交易关闭 REFUNDING	退款中	退款处理中 REFUNDED	退款成功	资金已退回"` // 交易状态 CREATED	已创建	准备请求第三方 PROCESSING	处理中	第三方处理中 SUCCESS	成功	第三方确认成功 FAILED	失败	交易失败 CLOSED	关闭	交易关闭 REFUNDING	退款中	退款处理中 REFUNDED	退款成功	资金已退回
	CallbackTime    *gtime.Time `json:"callbackTime"    orm:"callback_time"    description:"第三方回调时间"`                                                                                                                                 // 第三方回调时间
	CallbackContent string      `json:"callbackContent" orm:"callback_content" description:"第三方回调原始内容"`                                                                                                                               // 第三方回调原始内容
	RequestContent  string      `json:"requestContent"  orm:"request_content"  description:"请求第三方参数"`                                                                                                                                 // 请求第三方参数
	ResponseContent string      `json:"responseContent" orm:"response_content" description:"第三方响应内容"`                                                                                                                                 // 第三方响应内容
	CreatedAt       *gtime.Time `json:"createdAt"       orm:"created_at"       description:"创建时间"`                                                                                                                                    // 创建时间
	UpdatedAt       *gtime.Time `json:"updatedAt"       orm:"updated_at"       description:"更新时间"`                                                                                                                                    // 更新时间
}
