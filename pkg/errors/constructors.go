package errors

// New creates a new AppError with the given fields
func New(code, messageEN, messagePT string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		MessageEN:  messageEN,
		MessagePT:  messagePT,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an internal error with an AppError
func Wrap(base *AppError, internal error) *AppError {
	return base.WithInternal(internal)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to convert an error to an AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}
