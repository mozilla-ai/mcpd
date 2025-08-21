package config

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidValue     = errors.New("config value invalid")
	ErrInvalidKey       = errors.New("config key invalid")
	ErrConfigLoadFailed = errors.New("failed to load configuration")
)

// NewErrInvalidValue returns an error for an invalid configuration value.
func NewErrInvalidValue(key string, value string) error {
	return fmt.Errorf("%w: '%s' (value: '%s')", ErrInvalidValue, key, value)
}
