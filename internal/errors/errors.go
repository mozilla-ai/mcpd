package errors

import (
	"errors"
)

var (
	ErrBadRequest            = errors.New("bad request")
	ErrServerNotFound        = errors.New("server not found")
	ErrToolsNotFound         = errors.New("tools not found")
	ErrToolForbidden         = errors.New("tool not allowed")
	ErrToolListFailed        = errors.New("tool list failed")
	ErrToolCallFailed        = errors.New("tool call failed")
	ErrToolCallFailedUnknown = errors.New("tool call failed (unknown error)")
	ErrHealthNotTracked      = errors.New("server health is not being tracked")
)
