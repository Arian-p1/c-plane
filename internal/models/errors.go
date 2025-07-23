package models

import "errors"

// Common errors
var (
	// Device errors
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceAlreadyExists = errors.New("device already exists")
	ErrInvalidDeviceID     = errors.New("invalid device ID")
	ErrDeviceOffline       = errors.New("device is offline")
	ErrDeviceNotResponding = errors.New("device is not responding")

	// Fault errors
	ErrFaultNotFound            = errors.New("fault not found")
	ErrFaultAlreadyExists       = errors.New("fault already exists")
	ErrInvalidFaultID           = errors.New("invalid fault ID")
	ErrFaultAlreadyAcknowledged = errors.New("fault already acknowledged")
	ErrFaultAlreadyResolved     = errors.New("fault already resolved")

	// Task errors
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrInvalidTaskID     = errors.New("invalid task ID")
	ErrTaskInProgress    = errors.New("task is already in progress")
	ErrTaskCompleted     = errors.New("task is already completed")
	ErrTaskFailed        = errors.New("task execution failed")

	// Parameter errors
	ErrParameterNotFound     = errors.New("parameter not found")
	ErrParameterReadOnly     = errors.New("parameter is read-only")
	ErrInvalidParameterValue = errors.New("invalid parameter value")
	ErrParameterTypeMismatch = errors.New("parameter type mismatch")

	// Connection errors
	ErrConnectionFailed     = errors.New("connection failed")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrTimeout              = errors.New("operation timed out")
	ErrConnectionRefused    = errors.New("connection refused")

	// Validation errors
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingRequired = errors.New("missing required field")
	ErrInvalidFormat   = errors.New("invalid format")
	ErrOutOfRange      = errors.New("value out of range")

	// GenieACS errors
	ErrGenieACSUnavailable = errors.New("GenieACS service unavailable")
	ErrGenieACSTimeout     = errors.New("GenieACS request timeout")
	ErrGenieACSAPIError    = errors.New("GenieACS API error")
	ErrGenieACSAuthError   = errors.New("GenieACS authentication error")

	// Database errors
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")
	ErrDatabaseTimeout    = errors.New("database operation timeout")
	ErrRecordNotFound     = errors.New("record not found")
	ErrDuplicateRecord    = errors.New("duplicate record")

	// Permission errors
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface for ValidationErrors
func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	return ve.Errors[0].Message
}

// IsNotFound checks if an error is a "not found" type error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrDeviceNotFound) ||
		errors.Is(err, ErrFaultNotFound) ||
		errors.Is(err, ErrTaskNotFound) ||
		errors.Is(err, ErrParameterNotFound) ||
		errors.Is(err, ErrRecordNotFound)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(ValidationErrors)
	return ok
}

// IsConnectionError checks if an error is a connection-related error
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrConnectionRefused) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrGenieACSUnavailable) ||
		errors.Is(err, ErrGenieACSTimeout) ||
		errors.Is(err, ErrDatabaseConnection) ||
		errors.Is(err, ErrDatabaseTimeout)
}

// IsAuthError checks if an error is an authentication/authorization error
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed) ||
		errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrInsufficientPermissions) ||
		errors.Is(err, ErrGenieACSAuthError)
}
