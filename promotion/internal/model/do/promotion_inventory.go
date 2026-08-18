// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// PromotionInventory is the golang structure of table promotion_inventory for DAO operations like Where/Data.
type PromotionInventory struct {
	g.Meta    `orm:"table:promotion_inventory, do:true"`
	ProductId interface{} //
	Stock     interface{} //
	Locked    interface{} //
}
