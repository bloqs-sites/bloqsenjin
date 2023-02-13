package auth

type Payload struct {
	Client      string
	Permissions uint
}

type Tokener interface {
	GenToken(p Payload) string
	VerifyToken(t string, auths uint) bool
}
