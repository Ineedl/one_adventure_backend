// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ChannelInfoDao is the data access object for the table channel_info.
type ChannelInfoDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  ChannelInfoColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// ChannelInfoColumns defines and stores column names for the table channel_info.
type ChannelInfoColumns struct {
	Id       string //
	Name     string //
	Index    string //
	ServerId string //
}

// channelInfoColumns holds the columns for the table channel_info.
var channelInfoColumns = ChannelInfoColumns{
	Id:       "id",
	Name:     "name",
	Index:    "index",
	ServerId: "server_id",
}

// NewChannelInfoDao creates and returns a new DAO object for table data access.
func NewChannelInfoDao(handlers ...gdb.ModelHandler) *ChannelInfoDao {
	return &ChannelInfoDao{
		group:    "default",
		table:    "channel_info",
		columns:  channelInfoColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ChannelInfoDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ChannelInfoDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ChannelInfoDao) Columns() ChannelInfoColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ChannelInfoDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ChannelInfoDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ChannelInfoDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
