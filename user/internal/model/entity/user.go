// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// User is the golang structure for table user.
type User struct {
	Id         uint64      `json:"id"         orm:"id"          description:""` //
	Username   string      `json:"username"   orm:"username"    description:""` //
	Password   string      `json:"password"   orm:"password"    description:""` //
	CreateTime *gtime.Time `json:"createTime" orm:"create_time" description:""` //
	UpdateTime *gtime.Time `json:"updateTime" orm:"update_time" description:""` //
	Status     int         `json:"status"     orm:"status"      description:""` //
	IsDeleted  int         `json:"isDeleted"  orm:"is_deleted"  description:""` //
}
