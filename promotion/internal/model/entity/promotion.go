// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Promotion is the golang structure for table promotion.
type Promotion struct {
	PromotionId uint64      `json:"promotionId" orm:"promotion_id" description:""` //
	Name        string      `json:"name"        orm:"name"         description:""` //
	Type        int         `json:"type"        orm:"type"         description:""` //
	Status      int         `json:"status"      orm:"status"       description:""` //
	StartTime   *gtime.Time `json:"startTime"   orm:"start_time"   description:""` //
	EndTime     *gtime.Time `json:"endTime"     orm:"end_time"     description:""` //
}
