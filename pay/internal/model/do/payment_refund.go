// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentRefund is the golang structure of table payment_refund for DAO operations like Where/Data.
type PaymentRefund struct {
	g.Meta         `orm:"table:payment_refund, do:true"`
	Id             interface{} // 主键
	RefundNo       interface{} // 退款单号，系统内部唯一
	OrderId        interface{} // 业务订单ID
	PaymentOrderId interface{} // 支付单ID
	UserId         interface{} // 用户ID
	RefundAmount   interface{} // 退款金额，单位：分
	Reason         interface{} // 退款原因
	Status         interface{} // 退款状态 SUCCESS 成功 RUNNING 执行中 STOP        终止
	ThirdRefundNo  interface{} // 第三方退款单号
	RetryCount     interface{} // 退款执行/查询重试次数
	NextRetryAt    *gtime.Time // 下一次重试时间
	LastError      interface{} // 最近一次错误信息
	RefundTime     *gtime.Time // 退款成功时间
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
}
