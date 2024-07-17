package pets

import rego.v1

default allow := false

# Admins can see everything
allow if {
	"admin" in input.userinfo.groups
}

# Alpha users can see everything (after sanity check)
allow if {
	"alpha" in input.userinfo.groups
	sanity
}

# Otherwise follow the rules
allow if {
	sanity
	allowed
}

# Is owner
allowed if {
	data.tables.pets.owner = input.userinfo.preferred_username
}

# Is veterinarian
allowed if {
	data.tables.pets.veterinarian = input.userinfo.preferred_username
	data.tables.pets.clinic = input.userinfo.department
}

allowed if {
	data.tables.pets.n_owners < 1.23
}

allowed if {
	data.tables.pets.blah >= 20
	data.tables.pets.another_guess != true
}

allowed if {
	data.tables.pets.pet_id = data.tables.petsaccess.pet_id
	data.tables.petsaccess.type = "public"
}

allowed if {
	data.tables.pets.pet_id = data.tables.petsaccess.pet_id
	data.tables.petsaccess.type = "logged_in"
	input.userinfo
}

allowed if {
	data.tables.pets.pet_id = data.tables.petsaccess.pet_id
	data.tables.petsaccess.type = "userlist"
	data.tables.petsaccess.userlist_id = data.tables.petsacessuserlist.userlist_id
	data.tables.petsacessuserlist.user_id = input.userinfo.preferred_username
}

allowed if {
	data.tables.pets.pet_id = data.tables.petsaccess.pet_id
	data.tables.petsaccess.type = "grouplist"
	data.tables.petsaccess.grouplist_id = data.tables.petsacessgrouplist.grouplist_id
	some data.tables.petsacessgrouplist.group_id in input.userinfo.groups
}

is_pet_killer if {
	"killer" in input.userinfo.groups
}

is_pet_deleted if {
	data.tables.pets.deleted = true
}

sanity if {
	not is_pet_killer
	not is_pet_deleted
}
