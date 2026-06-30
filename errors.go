package smartcomplete

import (
	"errors"
	"fmt"
)

// Standard errors
var (
	ErrFileNotAuthorized  = errors.New("file not authorized")
	ErrProjectNotFound    = errors.New("project not found")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrContextTooLarge    = errors.New("context exceeds token limit")
	ErrLLMTimeout         = errors.New("LLM request timeout")
	ErrInvalidRequest     = errors.New("invalid completion request")
	ErrCacheMiss          = errors.New("cache miss")
	ErrFileNotFound       = errors.New("file not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
)

// CompletionError wraps errors with context
type CompletionError struct {
	Code    string
	Message string
	Err     error
}

// Error returns the error message
func (e *CompletionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *CompletionError) Unwrap() error {
	return e.Err
}

// NewCompletionError creates a new CompletionError
func NewCompletionError(code, message string, err error) *CompletionError {
	return &CompletionError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes
const (
	CodeValidation     = "VALIDATION_ERROR"
	CodeRateLimit      = "RATE_LIMIT"
	CodeFileAccess     = "FILE_ACCESS"
	CodeProjectAccess  = "PROJECT_ACCESS"
	CodeContextError   = "CONTEXT_ERROR"
	CodeLLMError       = "LLM_ERROR"
	CodeCacheError     = "CACHE_ERROR"
	CodeTimeout        = "TIMEOUT"
	CodeInternal       = "INTERNAL_ERROR"
)

// WrapValidationError wraps a validation error
func WrapValidationError(message string, err error) *CompletionError {
	return NewCompletionError(CodeValidation, message, err)
}

// WrapRateLimitError wraps a rate limit error
func WrapRateLimitError(message string, err error) *CompletionError {
	return NewCompletionError(CodeRateLimit, message, err)
}

// WrapFileAccessError wraps a file access error
func WrapFileAccessError(message string, err error) *CompletionError {
	return NewCompletionError(CodeFileAccess, message, err)
}

// WrapProjectAccessError wraps a project access error
func WrapProjectAccessError(message string, err error) *CompletionError {
	return NewCompletionError(CodeProjectAccess, message, err)
}

// WrapContextError wraps a context gathering error
func WrapContextError(message string, err error) *CompletionError {
	return NewCompletionError(CodeContextError, message, err)
}

// WrapLLMError wraps an LLM error
func WrapLLMError(message string, err error) *CompletionError {
	return NewCompletionError(CodeLLMError, message, err)
}

// WrapCacheError wraps a cache error
func WrapCacheError(message string, err error) *CompletionError {
	return NewCompletionError(CodeCacheError, message, err)
}

// WrapTimeoutError wraps a timeout error
func WrapTimeoutError(message string, err error) *CompletionError {
	return NewCompletionError(CodeTimeout, message, err)
}

// WrapInternalError wraps an internal error
func WrapInternalError(message string, err error) *CompletionError {
	return NewCompletionError(CodeInternal, message, err)
}
