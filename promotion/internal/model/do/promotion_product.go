// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
)

// PromotionProduct is the golang structure of table promotion_product for DAO operations like Where/Data.
type PromotionProduct struct {
	g.Meta      `orm:"table:promotion_product, do:true"`
	PromotionId interface{} //
	ProductId   interface{} //
	Price       interface{} //
	Stock       interface{} //
}
