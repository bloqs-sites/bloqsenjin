package auth

import (
	"strconv"
)

const (
	CREATE_PREFERENCE Permission = 1 << (NEEDLE_FOR_NEXT_PERMISSION + iota)
	UPDATE_PREFERENCE
	DELETE_PREFERENCE

	CREATE_ACCOUNT
	READ_ACCOUNT
	UPDATE_ACCOUNT
	DELETE_ACCOUNT

	CREATE_BLOQ
	UPDATE_BLOQ
	DELETE_BLOQ

	PREFERENCE_MANAGER = CREATE_PREFERENCE | UPDATE_PREFERENCE | DELETE_PREFERENCE

	DEFAULT_PERMISSIONS = CREATE_ACCOUNT | READ_ACCOUNT | CREATE_BLOQ | UPDATE_BLOQ
)

var Permissions = map[string]Permission{
	"create_bloq":    CREATE_BLOQ,
	"update_bloq":    UPDATE_BLOQ,
	"delete_bloq":    DELETE_BLOQ,
	"create_account": CREATE_ACCOUNT,
	"read_account":   CREATE_ACCOUNT,
	"update_account": UPDATE_ACCOUNT,
	"delete_account": DELETE_ACCOUNT,
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
