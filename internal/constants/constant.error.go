package constants

import "fmt"

// Sentinel errors used by the config loader. The structured
// DomainError type lives in internal/apperror — constants is reserved
// for true constants only.
var (
	ErrLoadConfig  = fmt.Errorf("failed to load config file")
	ErrParseConfig = fmt.Errorf("failed to parse env to config struct")
	ErrEmptyVar    = fmt.Errorf("required variable environment is empty")
)
