// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// OrdersDao is the data access object for the table orders.
type OrdersDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  OrdersColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// OrdersColumns defines and stores column names for the table orders.
type OrdersColumns struct {
	OrderId      string //
	UserId       string //
	ProductId    string //
	Quantity     string //
	Amount       string // 价格
	CurrencyType string // 货币类型
	OrderType    string // 订单类型
	Status       string // PENDING_PAY	待支付	订单创建成功，等待用户付款 PAID	已支付	支付成功，等待履约 PROCESSING	处理中	支付完成，订单正在处理（可选） SHIPPED	已发货	物流已发出 COMPLETED	已完成	交易结束 CANCELLED	已取消	订单关闭 REFUNDING	退款中	正在退款 REFUNDED	已退款	退款完成 CLOSED	已关闭	订单生命周期结束
	PromotionId  string // 商品所属活动
	CreateTime   string //
	UpdateTime   string //
	RequestId    string //
	OrderNo      string // 订单号
}

// ordersColumns holds the columns for the table orders.
var ordersColumns = OrdersColumns{
	OrderId:      "order_id",
	UserId:       "user_id",
	ProductId:    "product_id",
	Quantity:     "quantity",
	Amount:       "amount",
	CurrencyType: "currency_type",
	OrderType:    "order_type",
	Status:       "status",
	PromotionId:  "promotion_id",
	CreateTime:   "create_time",
	UpdateTime:   "update_time",
	RequestId:    "request_id",
	OrderNo:      "order_no",
}

// NewOrdersDao creates and returns a new DAO object for table data access.
func NewOrdersDao(handlers ...gdb.ModelHandler) *OrdersDao {
	return &OrdersDao{
		group:    "default",
		table:    "orders",
		columns:  ordersColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *OrdersDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *OrdersDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *OrdersDao) Columns() OrdersColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *OrdersDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *OrdersDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *OrdersDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
