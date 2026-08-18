// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// PromotionInventory is the golang structure for table promotion_inventory.
type PromotionInventory struct {
	ProductId uint64 `json:"productId" orm:"product_id" description:""` //
	Stock     int    `json:"stock"     orm:"stock"      description:""` //
	Locked    int    `json:"locked"    orm:"locked"     description:""` //
}
