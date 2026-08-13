// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// InventoryConfDao is the data access object for the table inventory_conf.
type InventoryConfDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  InventoryConfColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// InventoryConfColumns defines and stores column names for the table inventory_conf.
type InventoryConfColumns struct {
	Id      string //
	UserId  string //
	ItemNum string //
}

// inventoryConfColumns holds the columns for the table inventory_conf.
var inventoryConfColumns = InventoryConfColumns{
	Id:      "id",
	UserId:  "user_id",
	ItemNum: "item_num",
}

// NewInventoryConfDao creates and returns a new DAO object for table data access.
func NewInventoryConfDao(handlers ...gdb.ModelHandler) *InventoryConfDao {
	return &InventoryConfDao{
		group:    "default",
		table:    "inventory_conf",
		columns:  inventoryConfColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *InventoryConfDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *InventoryConfDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *InventoryConfDao) Columns() InventoryConfColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *InventoryConfDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *InventoryConfDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *InventoryConfDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
