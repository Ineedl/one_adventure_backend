// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// PromotionProduct is the golang structure for table promotion_product.
type PromotionProduct struct {
	PromotionId uint64 `json:"promotionId" orm:"promotion_id" description:""` //
	ProductId   uint64 `json:"productId"   orm:"product_id"   description:""` //
	Price       int64  `json:"price"       orm:"price"        description:""` //
	Stock       int    `json:"stock"       orm:"stock"        description:""` //
}
