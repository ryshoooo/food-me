package pg_class

import rego.v1

default allow := false

allow if {
	input.userinfo
}
