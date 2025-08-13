package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorType represents different types of errors in the application
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypeCrypto        ErrorType = "crypto"
	ErrorTypeWorker        ErrorType = "worker"
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeTUI           ErrorType = "tui"
	ErrorTypeGeneration    ErrorType = "generation"
	ErrorTypeTimeout       ErrorType = "timeout"
	ErrorTypeCancellation  ErrorType = "cancellation"
)

// BlocoError represents a structured error with context
type BlocoError struct {
	Type      ErrorType              `json:"type"`
	Operation string                 `json:"operation"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Stack     []string               `json:"stack,omitempty"`
}

// Error implements the error interface
func (e *BlocoError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error in %s: %s (caused by: %v)",
			e.Type, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error in %s: %s", e.Type, e.Operation, e.Message)
}

// Unwrap returns the underlying cause for error unwrapping
func (e *BlocoError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *BlocoError) WithContext(key string, value interface{}) *BlocoError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithStack captures the current stack trace
func (e *BlocoError) WithStack() *BlocoError {
	e.Stack = captureStack()
	return e
}

// NewBlocoError creates a new structured error
func NewBlocoError(errorType ErrorType, operation, message string) *BlocoError {
	return &BlocoError{
		Type:      errorType,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// NewBlocoErrorWithCause creates a new structured error with a cause
func NewBlocoErrorWithCause(errorType ErrorType, operation, message string, cause error) *BlocoError {
	return &BlocoError{
		Type:      errorType,
		Operation: operation,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// Specific error constructors for common cases

// NewValidationError creates a validation error
func NewValidationError(operation, message string) *BlocoError {
	return NewBlocoError(ErrorTypeValidation, operation, message)
}

// NewCryptoError creates a cryptographic error
func NewCryptoError(operation, message string, cause error) *BlocoError {
	return NewBlocoErrorWithCause(ErrorTypeCrypto, operation, message, cause)
}

// NewWorkerError creates a worker-related error
func NewWorkerError(operation, message string) *BlocoError {
	return NewBlocoError(ErrorTypeWorker, operation, message)
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(operation, message string) *BlocoError {
	return NewBlocoError(ErrorTypeConfiguration, operation, message)
}

// NewGenerationError creates a wallet generation error
func NewGenerationError(operation, message string, cause error) *BlocoError {
	return NewBlocoErrorWithCause(ErrorTypeGeneration, operation, message, cause)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, timeout time.Duration) *BlocoError {
	return NewBlocoError(ErrorTypeTimeout, operation,
		fmt.Sprintf("operation timed out after %v", timeout))
}

// NewCancellationError creates a cancellation error
func NewCancellationError(operation, message string) *BlocoError {
	return NewBlocoError(ErrorTypeCancellation, operation, message)
}

// captureStack captures the current stack trace
func captureStack() []string {
	var stack []string
	for i := 2; i < 10; i++ { // Skip captureStack and the calling function
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}
	return stack
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	if blocoErr, ok := err.(*BlocoError); ok {
		return blocoErr.Type == errorType
	}
	return false
}

// GetErrorContext extracts context from a BlocoError
func GetErrorContext(err error) map[string]interface{} {
	if blocoErr, ok := err.(*BlocoError); ok {
		return blocoErr.Context
	}
	return nil
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, operation, message string) *BlocoError {
	return NewBlocoErrorWithCause(errorType, operation, message, err)
}
