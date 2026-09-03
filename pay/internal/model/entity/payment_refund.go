// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// PaymentRefund is the golang structure for table payment_refund.
type PaymentRefund struct {
	Id             uint64      `json:"id"             orm:"id"               description:"主键"`                                         // 主键
	RefundNo       string      `json:"refundNo"       orm:"refund_no"        description:"退款单号，系统内部唯一"`                                // 退款单号，系统内部唯一
	OrderId        int64       `json:"orderId"        orm:"order_id"         description:"业务订单ID"`                                     // 业务订单ID
	PaymentOrderId int64       `json:"paymentOrderId" orm:"payment_order_id" description:"支付单ID"`                                      // 支付单ID
	UserId         int64       `json:"userId"         orm:"user_id"          description:"用户ID"`                                       // 用户ID
	RefundAmount   int64       `json:"refundAmount"   orm:"refund_amount"    description:"退款金额，单位：分"`                                  // 退款金额，单位：分
	Reason         string      `json:"reason"         orm:"reason"           description:"退款原因"`                                       // 退款原因
	Status         string      `json:"status"         orm:"status"           description:"退款状态 SUCCESS 成功 RUNNING 执行中 STOP        终止"` // 退款状态 SUCCESS 成功 RUNNING 执行中 STOP        终止
	ThirdRefundNo  string      `json:"thirdRefundNo"  orm:"third_refund_no"  description:"第三方退款单号"`                                    // 第三方退款单号
	RetryCount     int         `json:"retryCount"     orm:"retry_count"      description:"退款执行/查询重试次数"`                                // 退款执行/查询重试次数
	NextRetryAt    *gtime.Time `json:"nextRetryAt"    orm:"next_retry_at"    description:"下一次重试时间"`                                    // 下一次重试时间
	LastError      string      `json:"lastError"      orm:"last_error"       description:"最近一次错误信息"`                                   // 最近一次错误信息
	RefundTime     *gtime.Time `json:"refundTime"     orm:"refund_time"      description:"退款成功时间"`                                     // 退款成功时间
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"       description:""`                                           //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"       description:""`                                           //
}
