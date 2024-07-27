package pg_namespace

import rego.v1

default allow := false

allow if {
	input.userinfo
}
