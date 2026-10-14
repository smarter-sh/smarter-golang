package smarter

import (
	"errors"
	"fmt"
	"net/http"
)

// The SDK's errors follow the Python SDK's exception hierarchy (ADR-006). Every error that the SDK
// returns matches [ErrSmarter] with errors.Is; an api error also matches the sentinel of its http
// status, e.g. [ErrNotFound], and its details are in an [*APIStatusError] (errors.As).
var (
	// ErrSmarter matches every error that the SDK returns.
	ErrSmarter = errors.New("smarter")
	// ErrResourceNotFound matches a 404 ([ErrNotFound]), and a Find that found no resource of that name.
	ErrResourceNotFound = fmt.Errorf("%w: resource not found", ErrSmarter)
	// ErrBadRequest matches a 400: the request is invalid, e.g. a malformed manifest.
	ErrBadRequest = fmt.Errorf("%w: bad request", ErrSmarter)
	// ErrAuthentication matches a 401: the api key is missing, invalid, or inactive.
	ErrAuthentication = fmt.Errorf("%w: authentication failed", ErrSmarter)
	// ErrPermissionDenied matches a 403: the api key's user may not perform the operation.
	ErrPermissionDenied = fmt.Errorf("%w: permission denied", ErrSmarter)
	// ErrNotFound matches a 404: the resource does not exist, or the api key's user may not read it.
	// It also matches [ErrResourceNotFound].
	ErrNotFound = fmt.Errorf("%w: not found", ErrResourceNotFound)
	// ErrConflict matches a 409: the request conflicts with the resource's current state.
	ErrConflict = fmt.Errorf("%w: conflict", ErrSmarter)
	// ErrUnprocessableEntity matches a 422: the request is well-formed but cannot be processed.
	ErrUnprocessableEntity = fmt.Errorf("%w: unprocessable entity", ErrSmarter)
	// ErrRateLimit matches a 429: too many requests.
	ErrRateLimit = fmt.Errorf("%w: rate limited", ErrSmarter)
	// ErrInternalServer matches a 5xx: the platform failed to process the request.
	ErrInternalServer = fmt.Errorf("%w: internal server error", ErrSmarter)
)

var statusSentinels = map[int]error{
	http.StatusBadRequest:          ErrBadRequest,
	http.StatusUnauthorized:        ErrAuthentication,
	http.StatusForbidden:           ErrPermissionDenied,
	http.StatusNotFound:            ErrNotFound,
	http.StatusConflict:            ErrConflict,
	http.StatusUnprocessableEntity: ErrUnprocessableEntity,
	http.StatusTooManyRequests:     ErrRateLimit,
}

// ConfigurationError reports that the client is missing configuration that it needs, e.g. an api
// key, or that a value is invalid. NewClient returns it.
type ConfigurationError struct {
	Message string
}

func (e *ConfigurationError) Error() string { return "smarter: configuration: " + e.Message }

// Unwrap returns [ErrSmarter].
func (e *ConfigurationError) Unwrap() error { return ErrSmarter }

// ValueError reports an invalid argument, before any request is sent.
type ValueError struct {
	Message string
	Err     error
}

func (e *ValueError) Error() string { return "smarter: " + e.Message }

// Unwrap returns the cause, and [ErrSmarter].
func (e *ValueError) Unwrap() []error {
	if e.Err != nil {
		return []error{ErrSmarter, e.Err}
	}
	return []error{ErrSmarter}
}

// resourceNotFound is the error of a Find that found no resource of that name.
type resourceNotFound struct {
	kind, name string
}

func (e *resourceNotFound) Error() string {
	return fmt.Sprintf("smarter: %s %q not found", e.kind, e.name)
}

func (e *resourceNotFound) Unwrap() error { return ErrResourceNotFound }

// APIConnectionError reports that a request got no response, e.g. the server is unreachable, or it
// timed out: see Timeout.
type APIConnectionError struct {
	Method, URL string
	Err         error
	timeout     bool
}

func (e *APIConnectionError) Error() string {
	if e.timeout {
		return fmt.Sprintf("smarter: %s %s: request timed out", e.Method, e.URL)
	}
	return fmt.Sprintf("smarter: %s %s: connection error: %v", e.Method, e.URL, e.Err)
}

// Timeout reports whether the request timed out.
func (e *APIConnectionError) Timeout() bool { return e.timeout }

// Unwrap returns the cause, and [ErrSmarter].
func (e *APIConnectionError) Unwrap() []error { return []error{ErrSmarter, e.Err} }

// APIResponseValidationError reports that the api's response does not match the SDK's struct of it,
// which usually means that the SDK and the platform are of incompatible versions.
type APIResponseValidationError struct {
	Method, URL string
	StatusCode  int
	Body        []byte
	Err         error
}

func (e *APIResponseValidationError) Error() string {
	return fmt.Sprintf("smarter: the response of %s %s does not match the SDK's model: %v", e.Method, e.URL, e.Err)
}

// Unwrap returns the cause, and [ErrSmarter].
func (e *APIResponseValidationError) Unwrap() []error { return []error{ErrSmarter, e.Err} }

// APIStatusError reports that the api returned a 4xx or 5xx status. It matches the sentinel of its
// status with errors.Is, e.g. [ErrNotFound] for a 404.
type APIStatusError struct {
	Method, URL string
	// StatusCode is the response's http status.
	StatusCode int
	// Message is the error's description, from whichever shape of error the platform returned.
	Message string
	// Body is the response's body.
	Body []byte
	// Header is the response's headers.
	Header http.Header
	// JournalError is the Smarter Journal error, when the response is a journaled error.
	JournalError *SAMJournalAPIErrorBody
}

func (e *APIStatusError) Error() string {
	return fmt.Sprintf("smarter: error code: %d - %s", e.StatusCode, e.Message)
}

// ErrorClass returns the name of the platform's exception class, e.g. SAMBrokerErrorNotFound, when
// the platform sent it.
func (e *APIStatusError) ErrorClass() string {
	if e.JournalError == nil {
		return ""
	}
	return e.JournalError.ErrorClass
}

// Unwrap returns the sentinel of the error's status.
func (e *APIStatusError) Unwrap() error {
	if sentinel, ok := statusSentinels[e.StatusCode]; ok {
		return sentinel
	}
	if e.StatusCode >= 500 {
		return ErrInternalServer
	}
	return ErrSmarter
}

// PromptError reports that a streamed prompt failed: its result event carried an error response.
type PromptError struct {
	Message string
	// Response is the error response.
	Response *PromptChatAPIPromptResponse
}

func (e *PromptError) Error() string { return "smarter: prompt failed: " + e.Message }

// Unwrap returns [ErrSmarter].
func (e *PromptError) Unwrap() error { return ErrSmarter }
