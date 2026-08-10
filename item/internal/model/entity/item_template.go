// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// ItemTemplate is the golang structure for table item_template.
type ItemTemplate struct {
	Id     uint64 `json:"id"     orm:"id"      description:""`     //
	ItemId int64  `json:"itemId" orm:"item_id" description:""`     //
	Type   string `json:"type"   orm:"type"    description:"物品属性"` // 物品属性
}
