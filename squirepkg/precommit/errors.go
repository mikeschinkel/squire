package precommit

import "errors"

// Sentinel errors for the precommit package
var (
	// ErrPrecommit is the base sentinel for all precommit package errors
	ErrPrecommit = errors.New("precommit error")

	// ErrCacheNotFound indicates no cached analysis results exist
	ErrCacheNotFound = errors.New("cache not found")
)
