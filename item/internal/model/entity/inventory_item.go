// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// InventoryItem is the golang structure for table inventory_item.
type InventoryItem struct {
	Id         uint64 `json:"id"         orm:"id"          description:""` //
	TemplateId uint64 `json:"templateId" orm:"template_id" description:""` //
	InstanceId string `json:"instanceId" orm:"instance_id" description:""` //
	Index      int    `json:"index"      orm:"index"       description:""` //
	UserId     uint64 `json:"userId"     orm:"user_id"     description:""` //
}
