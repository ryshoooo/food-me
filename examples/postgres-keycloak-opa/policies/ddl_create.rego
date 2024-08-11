package ddl_create

import rego.v1

default allow := false

allow if {
	"admin" = input.userinfo.preferred_username
}

allow if {
	"admin" in input.userinfo.groups
}
