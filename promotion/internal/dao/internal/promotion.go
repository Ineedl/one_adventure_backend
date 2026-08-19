// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// PromotionDao is the data access object for the table promotion.
type PromotionDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  PromotionColumns   // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// PromotionColumns defines and stores column names for the table promotion.
type PromotionColumns struct {
	PromotionId string //
	Name        string //
	Type        string //
	Status      string // 1:启用 0:停止
	StartTime   string //
	EndTime     string //
	CreateTime  string //
	UpdateTime  string //
}

// promotionColumns holds the columns for the table promotion.
var promotionColumns = PromotionColumns{
	PromotionId: "promotion_id",
	Name:        "name",
	Type:        "type",
	Status:      "status",
	StartTime:   "start_time",
	EndTime:     "end_time",
	CreateTime:  "create_time",
	UpdateTime:  "update_time",
}

// NewPromotionDao creates and returns a new DAO object for table data access.
func NewPromotionDao(handlers ...gdb.ModelHandler) *PromotionDao {
	return &PromotionDao{
		group:    "default",
		table:    "promotion",
		columns:  promotionColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *PromotionDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *PromotionDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *PromotionDao) Columns() PromotionColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *PromotionDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *PromotionDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *PromotionDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
