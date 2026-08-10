// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// ItemTemplate is the golang structure of table item_template for DAO operations like Where/Data.
type ItemTemplate struct {
	g.Meta `orm:"table:item_template, do:true"`
	Id     interface{} //
	ItemId interface{} //
	Type   interface{} // 物品属性
}
