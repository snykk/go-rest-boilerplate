// Package apperror is the canonical home for the typed error envelope
// the business layer hands up to HTTP handlers. The HTTP layer maps
// each Type to a status code; usecases attach a Cause for logging
// without leaking internals to clients.
package apperror

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

// DomainError is the typed error returned across the business layer.
// Cause preserves the original error so errors.Is / errors.As can
// still reach wrapped causes (e.g. sql.ErrNoRows, ctx errors) without
// leaking the cause's text into client responses.
type DomainError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *DomainError) Error() string { return e.Message }

// Unwrap enables errors.Is / errors.As to traverse the cause chain.
func (e *DomainError) Unwrap() error { return e.Cause }

// New creates a DomainError with the given type and message.
func New(errType ErrorType, message string) *DomainError {
	return &DomainError{Type: errType, Message: message}
}

// Wrap attaches an underlying cause to an existing DomainError without
// changing its message. Returns a new value (does not mutate input).
func Wrap(err *DomainError, cause error) *DomainError {
	if err == nil {
		return nil
	}
	return &DomainError{Type: err.Type, Message: err.Message, Cause: cause}
}

// Convenience constructors for common domain errors.

func NotFound(msg string) *DomainError     { return New(ErrTypeNotFound, msg) }
func Unauthorized(msg string) *DomainError { return New(ErrTypeUnauthorized, msg) }
func Forbidden(msg string) *DomainError    { return New(ErrTypeForbidden, msg) }
func Conflict(msg string) *DomainError     { return New(ErrTypeConflict, msg) }
func BadRequest(msg string) *DomainError   { return New(ErrTypeBadRequest, msg) }
func Internal(msg string) *DomainError     { return New(ErrTypeInternal, msg) }

// InternalCause builds an internal error with a fixed, generic
// user-facing message and stashes the real cause for logging. Use
// this anywhere a low-level error would otherwise be turned into a
// 500 response — internal/library messages must not leak to clients.
func InternalCause(cause error) *DomainError {
	return &DomainError{
		Type:    ErrTypeInternal,
		Message: "internal server error",
		Cause:   cause,
	}
}
