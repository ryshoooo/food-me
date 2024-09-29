package petsaccess

import rego.v1

default allow := false

allow if {
	input.userinfo
}
