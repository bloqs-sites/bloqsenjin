package auth

import "github.com/bloqs-sites/bloqsenjin/pkg/auth"

const (
	CREATE_PREFERENCE auth.Permissions = 1 << (auth.NEEDLE_FOR_NEXT_PERMISSION + iota)
	UPDATE_PREFERENCE
	DELETE_PREFERENCE

	CREATE_ACCOUNT
	UPDATE_ACCOUNT
	DELETE_ACCOUNT

	CREATE_BLOQ
	UPDATE_BLOQ
	DELETE_BLOQ

	PREFERENCE_MANAGER = CREATE_PREFERENCE | UPDATE_PREFERENCE | DELETE_PREFERENCE

	DEFAULT_PERMISSIONS = CREATE_ACCOUNT | CREATE_BLOQ | UPDATE_BLOQ
)

var Permissions = map[string]auth.Permissions{
	"create_bloq":    CREATE_BLOQ,
	"update_bloq":    UPDATE_BLOQ,
	"delete_bloq":    DELETE_BLOQ,
	"create_account": CREATE_ACCOUNT,
	"update_account": UPDATE_ACCOUNT,
	"delete_account": DELETE_ACCOUNT,
	"default":        DEFAULT_PERMISSIONS,
}

var SuperPermissions = map[string]auth.Permissions{
	"create_preference": CREATE_PREFERENCE,
	"update_preference": UPDATE_PREFERENCE,
	"delete_preference": DELETE_PREFERENCE,
}
