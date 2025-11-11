package models

// APIError represents a standardized error response for the API
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error code constants
const (
	// General errors
	ErrBadRequest       = "BAD_REQUEST"
	ErrUnauthorized     = "UNAUTHORIZED"
	ErrForbidden        = "FORBIDDEN"
	ErrNotFound         = "NOT_FOUND"
	ErrConflict         = "CONFLICT"
	ErrInternalServer   = "INTERNAL_SERVER_ERROR"
	ErrValidationFailed = "VALIDATION_FAILED"

	// Pizza-specific errors
	ErrPizzaNotFound        = "PIZZA_NOT_FOUND"
	ErrPizzaInvalidData     = "PIZZA_INVALID_DATA"
	ErrPizzaDeleteForbidden = "PIZZA_DELETE_FORBIDDEN"

	// OAuth/Auth errors (maintain RFC 6749 compatibility)
	ErrInvalidRequest       = "invalid_request"
	ErrInvalidClient        = "invalid_client"
	ErrInvalidGrant         = "invalid_grant"
	ErrUnauthorizedClient   = "unauthorized_client"
	ErrUnsupportedGrantType = "unsupported_grant_type"
	ErrInvalidScope         = "invalid_scope"
)

// NewAPIError creates a new API error with the given code and message
func NewAPIError(code, message string, details ...map[string]interface{}) APIError {
	err := APIError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// OAuth2Error represents an OAuth2 error response (RFC 6749)
type OAuth2Error struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// NewOAuth2Error creates a new OAuth2 error response
func NewOAuth2Error(error, description string) OAuth2Error {
	return OAuth2Error{
		Error:            error,
		ErrorDescription: description,
	}
}
