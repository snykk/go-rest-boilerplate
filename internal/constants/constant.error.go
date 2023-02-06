package constants

import "errors"

var (
	// config
	ErrLoadConfig  = errors.New("failed to load config file")
	ErrParseConfig = errors.New("failed to parse env to config struct")
	ErrEmptyVar    = errors.New("required variabel environment is empty")
)
