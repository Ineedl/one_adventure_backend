// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// PaymentRefundDao is the data access object for the table payment_refund.
type PaymentRefundDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  PaymentRefundColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// PaymentRefundColumns defines and stores column names for the table payment_refund.
type PaymentRefundColumns struct {
	Id             string // 主键
	RefundNo       string // 退款单号，系统内部唯一
	OrderId        string // 业务订单ID
	PaymentOrderId string // 支付单ID
	UserId         string // 用户ID
	RefundAmount   string // 退款金额，单位：分
	Reason         string // 退款原因
	Status         string // 退款状态 SUCCESS 成功 RUNNING 执行中 STOP        终止
	ThirdRefundNo  string // 第三方退款单号
	RetryCount     string // 退款执行/查询重试次数
	NextRetryAt    string // 下一次重试时间
	LastError      string // 最近一次错误信息
	RefundTime     string // 退款成功时间
	CreatedAt      string //
	UpdatedAt      string //
}

// paymentRefundColumns holds the columns for the table payment_refund.
var paymentRefundColumns = PaymentRefundColumns{
	Id:             "id",
	RefundNo:       "refund_no",
	OrderId:        "order_id",
	PaymentOrderId: "payment_order_id",
	UserId:         "user_id",
	RefundAmount:   "refund_amount",
	Reason:         "reason",
	Status:         "status",
	ThirdRefundNo:  "third_refund_no",
	RetryCount:     "retry_count",
	NextRetryAt:    "next_retry_at",
	LastError:      "last_error",
	RefundTime:     "refund_time",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
}

// NewPaymentRefundDao creates and returns a new DAO object for table data access.
func NewPaymentRefundDao(handlers ...gdb.ModelHandler) *PaymentRefundDao {
	return &PaymentRefundDao{
		group:    "default",
		table:    "payment_refund",
		columns:  paymentRefundColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *PaymentRefundDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *PaymentRefundDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *PaymentRefundDao) Columns() PaymentRefundColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *PaymentRefundDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *PaymentRefundDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *PaymentRefundDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
