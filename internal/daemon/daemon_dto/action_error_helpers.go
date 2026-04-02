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

// NewValidationError creates a validation error with field-specific messages.
//
// Takes fields (map[string]string) which maps field names to error messages.
//
// Returns *ValidationError which contains the field-specific validation errors.
func NewValidationError(fields map[string]string) *ValidationError {
	return &ValidationError{Fields: fields}
}

// ValidationField creates a single-field validation error.
//
// This is a convenience function for the common case of one field failing
// validation.
//
// Takes field (string) which is the name of the field that failed validation.
// Takes message (string) which describes why the validation failed.
//
// Returns *ValidationError which contains the field name and error message.
func ValidationField(field, message string) *ValidationError {
	return &ValidationError{Fields: map[string]string{field: message}}
}

// NotFound creates a not found error for a resource.
//
// Takes resource (string) which is the type of resource that was not found.
// Takes id (string) which is the identifier of the missing resource.
//
// Returns *NotFoundError which wraps the resource type and identifier.
func NotFound(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

// NotFoundResource creates a not found error without a specific ID.
// Use this when the resource type is known but there is no specific identifier.
//
// Takes resource (string) which specifies the type of resource that was not found.
//
// Returns *NotFoundError which contains the resource type for error messages.
func NotFoundResource(resource string) *NotFoundError {
	return &NotFoundError{Resource: resource}
}

// Conflict creates a conflict error with the default CONFLICT code.
//
// Takes message (string) which describes the conflict that occurred.
//
// Returns *ConflictError which contains the message and default code.
func Conflict(message string) *ConflictError {
	return &ConflictError{Message: message, Code: "CONFLICT"}
}

// ConflictWithCode creates a conflict error with a custom error code.
// Use this when the client needs to distinguish between different conflict
// types.
//
// Takes message (string) which describes the conflict that occurred.
// Takes code (string) which identifies the type of conflict for clients.
//
// Returns *ConflictError which contains the message and code.
func ConflictWithCode(message, code string) *ConflictError {
	return &ConflictError{Message: message, Code: code}
}

// Forbidden creates a forbidden error.
//
// Takes message (string) which describes why access is forbidden.
//
// Returns *ForbiddenError which contains the forbidden error details.
func Forbidden(message string) *ForbiddenError {
	return &ForbiddenError{Message: message}
}

// Unauthorised creates an unauthorised error.
//
// Takes message (string) which specifies the error message.
//
// Returns *UnauthorisedError which contains the error details.
func Unauthorised(message string) *UnauthorisedError {
	return &UnauthorisedError{Message: message}
}

// BadRequest creates a bad request error.
//
// Takes message (string) which describes the error condition.
//
// Returns *BadRequestError which wraps the message for error handling.
func BadRequest(message string) *BadRequestError {
	return &BadRequestError{Message: message}
}

// PageError creates a generic page error with an arbitrary HTTP status code.
// Use this when none of the specific error helpers (NotFound, Forbidden, etc.)
// match the status code you need.
//
// Takes statusCode (int) which is the HTTP status code.
// Takes message (string) which describes the error.
//
// Returns *GenericPageError which contains the status code and message.
func PageError(statusCode int, message string) *GenericPageError {
	return &GenericPageError{Status: statusCode, Message: message}
}

// Teapot creates an HTTP 418 I'm a Teapot error. Short and stout.
//
// Takes message (string) which describes why this teapot refuses to brew
// coffee. Pass an empty string for the default message.
//
// Returns *TeapotError which is a teapot error.
func Teapot(message string) *TeapotError {
	return &TeapotError{Message: message}
}
