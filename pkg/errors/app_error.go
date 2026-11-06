package errors

import "fmt"

type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func Validation(msg string) *AppError {
	return &AppError{Status: 422, Code: "validation_failed", Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Status: 404, Code: "not_found", Message: msg}
}

func Internal(msg string, err error) *AppError {
	return &AppError{Status: 500, Code: "internal_error", Message: msg, Err: err}
}

func Unavailable(msg string, err error) *AppError {
	return &AppError{Status: 503, Code: "service_unavailable", Message: msg, Err: err}
}
