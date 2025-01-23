package z

type AppError struct {
	Msg string
}

func (e *AppError) Error() string {
	return e.Msg
}

type SystemError struct {
	Msg string
}

func (e *SystemError) Error() string {
	return e.Msg
}
