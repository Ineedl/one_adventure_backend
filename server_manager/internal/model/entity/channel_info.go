// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// ChannelInfo is the golang structure for table channel_info.
type ChannelInfo struct {
	Id       uint64 `json:"id"       orm:"id"        description:""` //
	Name     string `json:"name"     orm:"name"      description:""` //
	Index    int    `json:"index"    orm:"index"     description:""` //
	ServerId uint64 `json:"serverId" orm:"server_id" description:""` //
}
