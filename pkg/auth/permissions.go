package auth

import (
	"fmt"
	"strconv"
)

const (
	CREATE_PREFERENCE Permission = 1 << (NEEDLE_FOR_NEXT_PERMISSION + iota)
	UPDATE_PREFERENCE
	DELETE_PREFERENCE

	CREATE_PROFILE
	READ_PROFILE
	UPDATE_PROFILE
	DELETE_PROFILE

	CREATE_BLOQ
	UPDATE_BLOQ
	DELETE_BLOQ

	PREFERENCE_MANAGER = CREATE_PREFERENCE | UPDATE_PREFERENCE | DELETE_PREFERENCE

	DEFAULT_PERMISSIONS = CREATE_PROFILE | READ_PROFILE | CREATE_BLOQ | UPDATE_BLOQ
)

var Permissions = map[string]Permission{
	"create_bloq":    CREATE_BLOQ,
	"update_bloq":    UPDATE_BLOQ,
	"delete_bloq":    DELETE_BLOQ,
	"create_profile": CREATE_PROFILE,
	"read_profile":   CREATE_PROFILE,
	"update_profile": UPDATE_PROFILE,
	"delete_profile": DELETE_PROFILE,
	"default":        DEFAULT_PERMISSIONS,
}

var SuperPermissions = map[string]Permission{
	"create_preference": CREATE_PREFERENCE,
	"update_preference": UPDATE_PREFERENCE,
	"delete_preference": DELETE_PREFERENCE,
}

func GetPermissionsList(super bool) map[string]Permission {
	list := make(map[string]Permission, len(Permissions))
	for k, v := range Permissions {
		list[k] = v
	}
	if super {
		for k, v := range SuperPermissions {
			list[k] = v
		}
	}
	return list
}

func GetPermissionsHash(p Permission) string {
	list := GetPermissionsList(true)

	for k, v := range list {
		if v == p {
			return k
		}
	}

	return strconv.Itoa(int(p))
}

type NoPermissionsError struct {
	Permission Permission
}

func (err NoPermissionsError) Error() string {
	format := "The token provided does not have the `%s` permission."
	hash := GetPermissionsHash(err.Permission)
	return fmt.Sprintf(format, hash)
}
