package email

import "fmt"

type InvalidEmailError struct {
	Email  string
	Reason string
	Status uint16
}

func (e *InvalidEmailError) Error() string {
	return fmt.Sprintf("the email `%s` is invalid:\t%s", e.Email, e.Reason)
}

type ServerError struct {
	Err error
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error:\t%s", e.Err.Error())
}
