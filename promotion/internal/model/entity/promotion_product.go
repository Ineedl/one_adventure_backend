// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// PromotionProduct is the golang structure for table promotion_product.
type PromotionProduct struct {
	PromotionId  uint64      `json:"promotionId"  orm:"promotion_id"  description:""`                       //
	ProductId    uint64      `json:"productId"    orm:"product_id"    description:""`                       //
	Price        int64       `json:"price"        orm:"price"         description:""`                       //
	Stock        int         `json:"stock"        orm:"stock"         description:""`                       //
	CurrencyType int         `json:"currencyType" orm:"currency_type" description:"货币类型"`                   // 货币类型
	LimitType    int         `json:"limitType"    orm:"limit_type"    description:"限购类型 0:不限购 1:针对用户的数量限购"` // 限购类型 0:不限购 1:针对用户的数量限购
	LimitNum     int         `json:"limitNum"     orm:"limit_num"     description:"限购数量"`                   // 限购数量
	CreateTime   *gtime.Time `json:"createTime"   orm:"create_time"   description:""`                       //
	UpdateTime   *gtime.Time `json:"updateTime"   orm:"update_time"   description:""`                       //
}
