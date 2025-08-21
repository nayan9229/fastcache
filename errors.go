package fastcache

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrCacheClosed is returned when operations are attempted on a closed cache
	ErrCacheClosed = errors.New("cache is closed")

	// ErrKeyNotFound is returned when a key is not found in the cache
	ErrKeyNotFound = errors.New("key not found")

	// ErrInvalidKey is returned when an invalid key is provided
	ErrInvalidKey = errors.New("invalid key")

	// ErrMemoryLimitExceeded is returned when memory limit would be exceeded
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")
)

// ErrInvalidConfig represents a configuration validation error
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid config field '%s': %s", e.Field, e.Message)
}

// ErrOperationFailed represents an operation failure
type ErrOperationFailed struct {
	Operation string
	Key       string
	Reason    string
}

func (e ErrOperationFailed) Error() string {
	return fmt.Sprintf("operation '%s' failed for key '%s': %s", e.Operation, e.Key, e.Reason)
}

// ErrShardError represents a shard-specific error
type ErrShardError struct {
	ShardID int
	Err     error
}

func (e ErrShardError) Error() string {
	return fmt.Sprintf("shard %d error: %v", e.ShardID, e.Err)
}

func (e ErrShardError) Unwrap() error {
	return e.Err
}

// IsTemporaryError checks if an error is temporary and the operation can be retried
func IsTemporaryError(err error) bool {
	switch err {
	case ErrMemoryLimitExceeded:
		return true
	default:
		return false
	}
}

// IsPermanentError checks if an error is permanent and the operation should not be retried
func IsPermanentError(err error) bool {
	switch err {
	case ErrCacheClosed, ErrInvalidKey:
		return true
	default:
		var configErr ErrInvalidConfig
		return errors.As(err, &configErr)
	}
}
