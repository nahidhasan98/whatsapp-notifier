package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application-specific error codes
type ErrorCode string

const (
	// Client errors
	ErrCodeInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeTooManyRequests  ErrorCode = "TOO_MANY_REQUESTS"

	// WhatsApp errors
	ErrCodeClientNotConnected ErrorCode = "CLIENT_NOT_CONNECTED"
	ErrCodeConnectionFailed   ErrorCode = "CONNECTION_FAILED"
	ErrCodeMessageSendFailed  ErrorCode = "MESSAGE_SEND_FAILED"
	ErrCodeInvalidJID         ErrorCode = "INVALID_JID"

	// Server errors
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
)

// AppError represents an application error with additional context
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
	Err        error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new application error
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: getStatusCodeForError(code),
	}
}

// Wrap wraps an existing error with application context
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: getStatusCodeForError(code),
		Err:        err,
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: getStatusCodeForError(code),
		Err:        err,
	}
}

// getStatusCodeForError maps error codes to HTTP status codes
func getStatusCodeForError(code ErrorCode) int {
	switch code {
	case ErrCodeInvalidRequest, ErrCodeValidationFailed, ErrCodeInvalidJID:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrCodeClientNotConnected, ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeInternalError, ErrCodeConnectionFailed, ErrCodeMessageSendFailed, ErrCodeDatabaseError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Common error constructors for convenience

// ValidationError creates a validation error
func ValidationError(message string) *AppError {
	return New(ErrCodeValidationFailed, message)
}

// InvalidRequest creates an invalid request error
func InvalidRequest(message string) *AppError {
	return New(ErrCodeInvalidRequest, message)
}

// ClientNotConnected creates a client not connected error
func ClientNotConnected() *AppError {
	return New(ErrCodeClientNotConnected, "WhatsApp client is not connected")
}

// ConnectionFailed creates a connection failed error
func ConnectionFailed(err error) *AppError {
	return Wrap(err, ErrCodeConnectionFailed, "Failed to connect to WhatsApp")
}

// MessageSendFailed creates a message send failed error
func MessageSendFailed(err error) *AppError {
	return Wrap(err, ErrCodeMessageSendFailed, "Failed to send message")
}

// InvalidJID creates an invalid JID error
func InvalidJID(jid string) *AppError {
	return New(ErrCodeInvalidJID, fmt.Sprintf("Invalid WhatsApp JID: %s", jid))
}

// InternalError creates an internal server error
func InternalError(err error) *AppError {
	return Wrap(err, ErrCodeInternalError, "Internal server error")
}

// DatabaseError creates a database error
func DatabaseError(err error) *AppError {
	return Wrap(err, ErrCodeDatabaseError, "Database operation failed")
}
