package petsaccessuserlist

import rego.v1

default allow := false

allow if {
	"admin" in input.userinfo.groups
}

allow if {
	input.userinfo
	data.tables.petsaccessuserlist.user_id = input.userinfo.preferred_username
}
