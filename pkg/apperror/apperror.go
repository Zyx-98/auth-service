package apperror

import "net/http"

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, status int, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

func BadRequest(message string, err error) *AppError {
	return NewAppError("bad_request", message, http.StatusBadRequest, err)
}

func Unauthorized(message string) *AppError {
	return NewAppError("unauthorized", message, http.StatusUnauthorized, nil)
}

func Forbidden(message string) *AppError {
	return NewAppError("forbidden", message, http.StatusForbidden, nil)
}

func NotFound(message string) *AppError {
	return NewAppError("not_found", message, http.StatusNotFound, nil)
}

func Conflict(message string, err error) *AppError {
	return NewAppError("conflict", message, http.StatusConflict, err)
}

func InternalServerError(message string, err error) *AppError {
	return NewAppError("internal_server_error", message, http.StatusInternalServerError, err)
}
