package errors

// AppError represents a structured application error with i18n support
type AppError struct {
	Code       string            // Machine-readable: "ERR_NOT_FOUND"
	MessageEN  string            // English message
	MessagePT  string            // Portuguese message
	HTTPStatus int               // HTTP status code
	Internal   error             // Wrapped internal error (not exposed to API)
	Details    map[string]string // Additional context
}

func (e *AppError) Error() string {
	return e.MessageEN
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

// LocalizedMessage returns the message in the requested language
func (e *AppError) LocalizedMessage(lang string) string {
	if lang == "pt" || lang == "pt-BR" || lang == "pt_BR" {
		return e.MessagePT
	}
	return e.MessageEN
}

// WithInternal wraps an internal error while preserving the AppError fields
func (e *AppError) WithInternal(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		MessageEN:  e.MessageEN,
		MessagePT:  e.MessagePT,
		HTTPStatus: e.HTTPStatus,
		Internal:   err,
		Details:    e.Details,
	}
}

// WithDetail adds context detail to the error
func (e *AppError) WithDetail(key, value string) *AppError {
	newErr := &AppError{
		Code:       e.Code,
		MessageEN:  e.MessageEN,
		MessagePT:  e.MessagePT,
		HTTPStatus: e.HTTPStatus,
		Internal:   e.Internal,
		Details:    make(map[string]string),
	}
	for k, v := range e.Details {
		newErr.Details[k] = v
	}
	newErr.Details[key] = value
	return newErr
}
