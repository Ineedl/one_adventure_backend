// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Promotion is the golang structure of table promotion for DAO operations like Where/Data.
type Promotion struct {
	g.Meta      `orm:"table:promotion, do:true"`
	PromotionId interface{} //
	Name        interface{} //
	Type        interface{} //
	Status      interface{} //
	StartTime   *gtime.Time //
	EndTime     *gtime.Time //
}
