package http

type HttpError struct {
	Body   string
	Status uint16
}

func (e *HttpError) Error() string {
	return e.Body
}
