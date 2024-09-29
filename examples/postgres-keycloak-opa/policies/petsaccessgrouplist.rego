package petsaccessgrouplist

import rego.v1

default allow := false

allow if {
	"admin" in input.userinfo.groups
}

allow if {
	input.userinfo
	some data.tables.petsaccessgrouplist.group_id in input.userinfo.groups
}
