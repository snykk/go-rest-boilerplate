package constants

import "fmt"

// ErrorType represents the category of a domain error.
type ErrorType int

const (
	ErrTypeInternal ErrorType = iota
	ErrTypeNotFound
	ErrTypeUnauthorized
	ErrTypeForbidden
	ErrTypeConflict
	ErrTypeBadRequest
)

// DomainError is a structured error used across the business layer.
type DomainError struct {
	Type    ErrorType
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// NewDomainError creates a new DomainError with the given type and message.
func NewDomainError(errType ErrorType, message string) *DomainError {
	return &DomainError{Type: errType, Message: message}
}

// Convenience constructors for common domain errors.
func ErrNotFound(msg string) *DomainError {
	return NewDomainError(ErrTypeNotFound, msg)
}

func ErrUnauthorized(msg string) *DomainError {
	return NewDomainError(ErrTypeUnauthorized, msg)
}

func ErrForbidden(msg string) *DomainError {
	return NewDomainError(ErrTypeForbidden, msg)
}

func ErrConflict(msg string) *DomainError {
	return NewDomainError(ErrTypeConflict, msg)
}

func ErrBadRequest(msg string) *DomainError {
	return NewDomainError(ErrTypeBadRequest, msg)
}

func ErrInternal(msg string) *DomainError {
	return NewDomainError(ErrTypeInternal, msg)
}

// Sentinel errors (for backwards compatibility where still referenced).
var (
	ErrUnexpected  = fmt.Errorf("unexpected error")
	ErrUserNotFound = fmt.Errorf("user not found")
	ErrLoadConfig  = fmt.Errorf("failed to load config file")
	ErrParseConfig = fmt.Errorf("failed to parse env to config struct")
	ErrEmptyVar    = fmt.Errorf("required variable environment is empty")
)
