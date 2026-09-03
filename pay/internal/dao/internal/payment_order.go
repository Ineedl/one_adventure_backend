// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// PaymentOrderDao is the data access object for the table payment_order.
type PaymentOrderDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  PaymentOrderColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// PaymentOrderColumns defines and stores column names for the table payment_order.
type PaymentOrderColumns struct {
	Id         string // 主键ID
	PaymentNo  string // 支付单号
	OrderNo    string // 业务订单号
	UserId     string // 用户ID
	Amount     string // 支付金额
	Currency   string // 币种
	Status     string // INIT	初始化	支付单刚创建 WAIT_PAY	等待支付	等待用户付款 PAYING	支付中	已发起第三方支付 SUCCESS	支付成功	支付完成 FAILED	支付失败	支付失败 CLOSED	已关闭	支付单失效 REFUNDING	退款中	退款处理中 REFUNDED	已退款	退款完成
	Subject    string // 支付标题
	ExpireTime string // 支付过期时间
	PaidTime   string // 支付完成时间
	CreatedAt  string // 创建时间
	UpdatedAt  string // 更新时间
}

// paymentOrderColumns holds the columns for the table payment_order.
var paymentOrderColumns = PaymentOrderColumns{
	Id:         "id",
	PaymentNo:  "payment_no",
	OrderNo:    "order_no",
	UserId:     "user_id",
	Amount:     "amount",
	Currency:   "currency",
	Status:     "status",
	Subject:    "subject",
	ExpireTime: "expire_time",
	PaidTime:   "paid_time",
	CreatedAt:  "created_at",
	UpdatedAt:  "updated_at",
}

// NewPaymentOrderDao creates and returns a new DAO object for table data access.
func NewPaymentOrderDao(handlers ...gdb.ModelHandler) *PaymentOrderDao {
	return &PaymentOrderDao{
		group:    "default",
		table:    "payment_order",
		columns:  paymentOrderColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *PaymentOrderDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *PaymentOrderDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *PaymentOrderDao) Columns() PaymentOrderColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *PaymentOrderDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *PaymentOrderDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *PaymentOrderDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
