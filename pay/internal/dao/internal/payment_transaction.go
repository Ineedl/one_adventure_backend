// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// PaymentTransactionDao is the data access object for the table payment_transaction.
type PaymentTransactionDao struct {
	table    string                    // table is the underlying table name of the DAO.
	group    string                    // group is the database configuration group name of the current DAO.
	columns  PaymentTransactionColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler        // handlers for customized model modification.
}

// PaymentTransactionColumns defines and stores column names for the table payment_transaction.
type PaymentTransactionColumns struct {
	Id              string // 主键ID
	PaymentNo       string // 支付单号
	OrderNo         string // 业务订单号(冗余)
	TransactionNo   string // 第三方交易号
	Channel         string // 支付渠道
	RequestAmount   string // 请求支付金额
	PaidAmount      string // 实际支付金额
	Status          string // 交易状态 CREATED	已创建	准备请求第三方 PROCESSING	处理中	第三方处理中 SUCCESS	成功	第三方确认成功 FAILED	失败	交易失败 CLOSED	关闭	交易关闭 REFUNDING	退款中	退款处理中 REFUNDED	退款成功	资金已退回
	CallbackTime    string // 第三方回调时间
	CallbackContent string // 第三方回调原始内容
	RequestContent  string // 请求第三方参数
	ResponseContent string // 第三方响应内容
	CreatedAt       string // 创建时间
	UpdatedAt       string // 更新时间
}

// paymentTransactionColumns holds the columns for the table payment_transaction.
var paymentTransactionColumns = PaymentTransactionColumns{
	Id:              "id",
	PaymentNo:       "payment_no",
	OrderNo:         "order_no",
	TransactionNo:   "transaction_no",
	Channel:         "channel",
	RequestAmount:   "request_amount",
	PaidAmount:      "paid_amount",
	Status:          "status",
	CallbackTime:    "callback_time",
	CallbackContent: "callback_content",
	RequestContent:  "request_content",
	ResponseContent: "response_content",
	CreatedAt:       "created_at",
	UpdatedAt:       "updated_at",
}

// NewPaymentTransactionDao creates and returns a new DAO object for table data access.
func NewPaymentTransactionDao(handlers ...gdb.ModelHandler) *PaymentTransactionDao {
	return &PaymentTransactionDao{
		group:    "default",
		table:    "payment_transaction",
		columns:  paymentTransactionColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *PaymentTransactionDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *PaymentTransactionDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *PaymentTransactionDao) Columns() PaymentTransactionColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *PaymentTransactionDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *PaymentTransactionDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *PaymentTransactionDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
