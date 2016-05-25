package multitenancyerrors

type GeneralError struct {
	msg string
}

func (e GeneralError) Error() string {
	return e.msg
}