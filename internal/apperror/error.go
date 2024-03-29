package apperror

import (
	"encoding/json"
	"fmt"
)

var (
	ErrNotFound = NewAppError("not found", "CR-000010", "")
)

type AppError struct {
	Err              error  `json:"-"`
	Message          string `json:"message,omitempty"`
	DeveloperMessage string `json:"developer_message,omitempty"`
	Code             string `json:"code,omitempty"`
}

func NewAppError(message, code, developerMessage string) *AppError {
	return &AppError{
		Err:              fmt.Errorf(message),
		Message:          message,
		DeveloperMessage: developerMessage,
		Code:             code,
	}
}

func (e *AppError) Error() string {
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Marshal() []byte {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return bytes
}

func UnauthorizedError(message string) *AppError {
	return NewAppError(message, "CR-000003", "")
}

func BadRequestError(message string) *AppError {
	return NewAppError(message, "CR-000002", "some thing wrong with user data")
}

func systemError(developerMessage string) *AppError {
	return NewAppError("system error", "CR-00001", developerMessage)
}

func APIError(code, message, developerMessage string) *AppError {
	return NewAppError(message, code, developerMessage)
}
