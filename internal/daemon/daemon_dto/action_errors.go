// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package daemon_dto

import (
	"fmt"
	"net/http"
)

// ActionError is the base interface for action errors with HTTP semantics.
// Errors implementing this interface are automatically discriminated by the
// action handler to return appropriate HTTP status codes and structured responses.
type ActionError interface {
	error

	// StatusCode returns the HTTP status code.
	//
	// Returns int which is the HTTP status code for the response.
	StatusCode() int

	// ErrorCode returns the error code as a string.
	ErrorCode() string
}

// ValidationError represents validation failures (HTTP 422).
// Use this when user input fails validation rules.
type ValidationError struct {
	// Fields contains field-specific error messages, keyed by field name.
	Fields map[string]string
}

// Error implements the error interface.
//
// Returns string which contains the validation failure message.
func (*ValidationError) Error() string {
	return "validation failed"
}

// StatusCode returns HTTP 422 Unprocessable Entity.
//
// Returns int which is the HTTP status code for validation errors.
func (*ValidationError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "VALIDATION_FAILED".
func (*ValidationError) ErrorCode() string {
	return "VALIDATION_FAILED"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *ValidationError) SafeMessage() string { return e.Error() }

// NotFoundError represents resource not found (HTTP 404).
// Use this when a requested resource does not exist.
type NotFoundError struct {
	// Resource is the type of resource that was not found (e.g., "user", "order").
	Resource string

	// ID is the identifier that was searched for.
	ID string
}

// Error implements the error interface.
//
// Returns string which describes the resource that was not found.
func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// StatusCode returns HTTP 404 Not Found.
//
// Returns int which is the HTTP status code for this error.
func (*NotFoundError) StatusCode() int {
	return http.StatusNotFound
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "NOT_FOUND".
func (*NotFoundError) ErrorCode() string {
	return "NOT_FOUND"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *NotFoundError) SafeMessage() string { return e.Error() }

// ConflictError represents a conflict (HTTP 409).
// Use this when an operation cannot complete due to a conflict with current state.
type ConflictError struct {
	// Message describes the conflict.
	Message string

	// Code is a machine-readable error code for the specific conflict type.
	Code string
}

// Error implements the error interface.
//
// Returns string which is the conflict message.
func (e *ConflictError) Error() string {
	return e.Message
}

// StatusCode returns HTTP 409 Conflict.
//
// Returns int which is the HTTP status code for this error.
func (*ConflictError) StatusCode() int {
	return http.StatusConflict
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the custom code if set, otherwise "CONFLICT".
func (e *ConflictError) ErrorCode() string {
	if e.Code != "" {
		return e.Code
	}
	return "CONFLICT"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *ConflictError) SafeMessage() string { return e.Error() }

// ForbiddenError represents authorisation failure (HTTP 403).
// Use this when the user is authenticated but lacks permission.
type ForbiddenError struct {
	// Message describes why access was denied.
	Message string
}

// Error implements the error interface.
//
// Returns string which is the error message, or "access denied" if no message
// is set.
func (e *ForbiddenError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "access denied"
}

// StatusCode returns HTTP 403 Forbidden.
//
// Returns int which is the HTTP status code for forbidden access.
func (*ForbiddenError) StatusCode() int {
	return http.StatusForbidden
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "FORBIDDEN".
func (*ForbiddenError) ErrorCode() string {
	return "FORBIDDEN"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *ForbiddenError) SafeMessage() string { return e.Error() }

// UnauthorisedError represents authentication failure (HTTP 401).
// Use this when the user needs to authenticate.
type UnauthorisedError struct {
	// Message describes why authentication failed.
	Message string
}

// Error implements the error interface.
//
// Returns string which is the custom message if set, or a default message.
func (e *UnauthorisedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "authentication required"
}

// StatusCode returns HTTP 401 Unauthorised.
//
// Returns int which is the HTTP status code.
func (*UnauthorisedError) StatusCode() int {
	return http.StatusUnauthorized
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "UNAUTHORISED".
func (*UnauthorisedError) ErrorCode() string {
	return "UNAUTHORISED"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *UnauthorisedError) SafeMessage() string { return e.Error() }

// BadRequestError represents a malformed request (HTTP 400).
// Use this when the request itself is invalid, not just the data within it.
type BadRequestError struct {
	// Message describes what was wrong with the request.
	Message string
}

// Error implements the error interface.
//
// Returns string which is the error message, or "bad request" if empty.
func (e *BadRequestError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "bad request"
}

// StatusCode returns HTTP 400 Bad Request.
//
// Returns int which is the HTTP status code for bad request errors.
func (*BadRequestError) StatusCode() int {
	return http.StatusBadRequest
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "BAD_REQUEST".
func (*BadRequestError) ErrorCode() string {
	return "BAD_REQUEST"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *BadRequestError) SafeMessage() string { return e.Error() }

// GenericPageError represents an error with an arbitrary HTTP status code.
// Use this when none of the specific error types (NotFoundError, ForbiddenError,
// etc.) match the status code you need.
type GenericPageError struct {
	// Message is the human-readable error message.
	Message string

	// Status is the HTTP status code for the error.
	Status int
}

// Error implements the error interface.
//
// Returns string which is the error message.
func (e *GenericPageError) Error() string {
	return e.Message
}

// StatusCode returns the HTTP status code.
//
// Returns int which is the status code set when the error was created.
func (e *GenericPageError) StatusCode() int {
	return e.Status
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "PAGE_ERROR".
func (*GenericPageError) ErrorCode() string {
	return "PAGE_ERROR"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *GenericPageError) SafeMessage() string { return e.Error() }

// TeapotError represents HTTP 418 I'm a Teapot (RFC 2324).
type TeapotError struct {
	// Message describes why this teapot refuses to brew coffee.
	Message string
}

// Error implements the error interface.
//
// Returns string which is the error message, or a default teapot message.
func (e *TeapotError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "I'm a teapot"
}

// StatusCode returns HTTP 418 I'm a Teapot.
//
// Returns int which is the HTTP status code for teapots.
func (*TeapotError) StatusCode() int {
	return http.StatusTeapot
}

// ErrorCode returns the error code for client-side discrimination.
//
// Returns string which is the constant "TEAPOT".
func (*TeapotError) ErrorCode() string {
	return "TEAPOT"
}

// SafeMessage returns the user-safe error message.
//
// Returns string which is the safe message for this error.
func (e *TeapotError) SafeMessage() string { return e.Error() }
