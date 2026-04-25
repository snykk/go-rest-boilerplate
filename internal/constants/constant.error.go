package constants

import "fmt"

// Sentinel errors used by config loading and a handful of legacy call
// sites. The structured DomainError type lives in
// internal/apperror — constants is reserved for true constants only.
var (
	ErrUnexpected   = fmt.Errorf("unexpected error")
	ErrUserNotFound = fmt.Errorf("user not found")
	ErrLoadConfig   = fmt.Errorf("failed to load config file")
	ErrParseConfig  = fmt.Errorf("failed to parse env to config struct")
	ErrEmptyVar     = fmt.Errorf("required variable environment is empty")
)
