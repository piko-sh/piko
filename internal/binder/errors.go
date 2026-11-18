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

package binder

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/runeutil"
)

const (
	// errorCodeValidationFailed is the machine-readable code for refused input, whether a
	// value would not convert or broke a `validate:"..."` tag.
	errorCodeValidationFailed = "VALIDATION_FAILED"

	// errorCodePayloadTooLarge is the machine-readable code for input that broke one of the
	// binder's size limits.
	errorCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"

	// errorCodeBadRequest is the machine-readable code for malformed JSON or a top-level
	// array or scalar other than null.
	errorCodeBadRequest = "BAD_REQUEST"

	// malformedJSONSafeMessage is shown for malformed JSON or a top-level array or scalar
	// other than null. It omits decoder details.
	malformedJSONSafeMessage = "request body is not a valid JSON object"

	// limitSafeMessage is the message a client is shown when its input broke a size limit.
	limitSafeMessage = "request has too many fields or elements"

	// fieldMessageNumberOutOfRange is shown for a number that does not fit its field.
	fieldMessageNumberOutOfRange = "Number is too large or too small"

	// fieldMessageInvalidValue is shown for any other value a field could not accept.
	fieldMessageInvalidValue = "Enter a valid value"

	// maxMultiErrorTextFields caps how many fields MultiError.Error lists, so a log line
	// stays small however many fields failed.
	maxMultiErrorTextFields = 20

	// maxFieldMessageKeyRunes caps how much of a field name FieldMessages echoes back to the
	// client, since the name is whatever key the client sent.
	maxFieldMessageKeyRunes = 256

	// maxQuotedValueRunes caps how much of a refused value, field name or path an error
	// message repeats.
	maxQuotedValueRunes = 64

	// maxCauseRunes caps how much of an underlying error's text an error message repeats,
	// since that text can hold the value the client sent.
	maxCauseRunes = 256
)

var (
	// syntaxFieldMessages maps the strconv function that could not read a value to the
	// message that tells the user what the field expects.
	syntaxFieldMessages = map[string]string{
		"ParseInt":   "Enter a whole number",
		"ParseUint":  "Enter a whole number that is 0 or more",
		"ParseFloat": "Enter a number",
		"ParseBool":  "Enter true or false",
	}
)

// MultiError holds multiple binding errors from a single form. It implements the error
// interface.
//
// Every entry describes a value the client sent, so it answers with 422 and
// VALIDATION_FAILED, the same as a failed `validate:"..."` tag, and not as a server
// fault.
type MultiError map[string]error

// Error implements the error interface for MultiError.
//
// Fields are listed in name order, at most maxMultiErrorTextFields of them, and each name
// and cause is shortened, so the text stays small enough to log whatever the client sent.
//
// Returns string which lists the failed fields, or "(0 errors)" when the map is empty.
func (m MultiError) Error() string {
	if len(m) == 0 {
		return "(0 errors)"
	}

	names := slices.Sorted(maps.Keys(m))
	listed := names[:min(len(names), maxMultiErrorTextFields)]

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%d validation errors:\n", len(m))
	for _, name := range listed {
		_, _ = fmt.Fprintf(&b, "  - field '%s': %s\n", runeutil.Truncate(name, maxQuotedValueRunes), describeCause(m[name]))
	}
	if hidden := len(names) - len(listed); hidden > 0 {
		_, _ = fmt.Fprintf(&b, "  ... and %d more\n", hidden)
	}
	return b.String()
}

// SafeMessage returns a user-safe error message that does not expose field names or
// internal types.
//
// Returns string which is a generic validation failure message.
func (MultiError) SafeMessage() string { return "validation failed" }

// StatusCode reports the HTTP status a binding failure answers with.
//
// Returns int which is 422.
func (MultiError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// ErrorCode reports the machine-readable code a binding failure answers with.
//
// Returns string which is the code a failed `validate:"..."` tag also answers with, so a
// client handles both in the same way.
func (MultiError) ErrorCode() string {
	return errorCodeValidationFailed
}

// FieldMessages maps each field that could not be bound to a message the client can show
// next to it.
//
// A message says what the field expects. It never repeats the value, the Go field name or
// the Go type. Fields are selected in name order when the response is capped. The
// binder's configured limit and DefaultMaxFieldErrors bound the response size. A trimmed
// map also holds FieldErrorsTruncatedKey.
//
// A field name is the key the client sent, so it is cut to maxFieldMessageKeyRunes runes
// before it is echoed back. When two names share that prefix, the first in name order
// keeps it.
//
// Returns map[string]string keyed by the field name the client sent, or nil when there
// are no failures.
func (m MultiError) FieldMessages() map[string]string {
	if len(m) == 0 {
		return nil
	}

	names, total := m.failedFields()
	kept := names[:min(int64(len(names)), DefaultMaxFieldErrors)]

	messages := make(map[string]string, len(kept)+1)
	for _, name := range kept {
		key := runeutil.Truncate(name, maxFieldMessageKeyRunes)
		if _, taken := messages[key]; !taken {
			messages[key] = fieldMessage(m[name])
		}
	}
	if len(kept) < total {
		messages[FieldErrorsTruncatedKey] = fieldErrorsTruncatedNotice(int64(len(kept)), total)
	}
	return messages
}

// Unwrap exposes the per-field errors so errors.Is and errors.As reach the sentinels
// underneath, which lets a caller detect a cancelled bind or an exhausted budget without
// parsing the message.
//
// Returns []error which contains every per-field error, in map iteration order.
func (m MultiError) Unwrap() []error {
	if len(m) == 0 {
		return nil
	}

	unwrapped := make([]error, 0, len(m))
	for _, err := range m {
		unwrapped = append(unwrapped, err)
	}
	return unwrapped
}

// failedFields lists the fields that failed, leaving out the truncation marker.
//
// Returns []string which holds the failed field names in name order.
// Returns int which is how many fields failed in total, counting any the binder already
// trimmed away.
func (m MultiError) failedFields() ([]string, int) {
	names := make([]string, 0, len(m))
	for name := range m {
		if name != FieldErrorsTruncatedKey {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	total := len(names)
	if marker, ok := errors.AsType[errFieldErrorsTruncated](m[FieldErrorsTruncatedKey]); ok {
		total = max(total, marker.total)
	}
	return names, total
}

// errInvalidTarget is returned when the destination for binding is not a pointer to a
// struct.
type errInvalidTarget struct {
	// targetType is the name of the type that was given instead of a struct pointer.
	targetType string
}

// Error implements the error interface for errInvalidTarget.
//
// Returns string which describes the invalid target type that was given.
func (e errInvalidTarget) Error() string {
	return fmt.Sprintf("binder: destination must be a pointer to a struct, but got %s", e.targetType)
}

// SafeMessage returns a user-safe error message that does not expose internal type
// information.
//
// Returns string which is a generic binding failure message.
func (errInvalidTarget) SafeMessage() string { return "invalid form data" }

// errInvalidPath is returned when a key in the source map is not a valid Piko expression
// path.
type errInvalidPath struct {
	// err is the underlying error that explains why the path is invalid.
	err error

	// path is the file path that failed validation.
	path string
}

// Error implements the error interface for errInvalidPath.
//
// Returns string which contains the invalid path and the underlying error.
func (e errInvalidPath) Error() string {
	return fmt.Sprintf("binder: invalid path '%s': %s", runeutil.Truncate(e.path, maxQuotedValueRunes), describeCause(e.err))
}

// SafeMessage returns a user-safe error message that does not expose internal path
// details.
//
// Returns string which is a generic binding failure message.
func (errInvalidPath) SafeMessage() string { return "invalid form data" }

// Unwrap exposes the underlying cause so errors.Is and errors.As can match it.
//
// Returns error which is the cause of the path being rejected.
func (e errInvalidPath) Unwrap() error { return e.err }

// errSetField is returned when a value cannot be converted or set on a struct field.
type errSetField struct {
	// err is the underlying error from the failed field assignment.
	err error

	// path is the binder path where setting the field failed.
	path string

	// field is the name of the struct field that could not be set.
	field string

	// fieldType is the expected type name for the field.
	fieldType string
}

// Error implements the error interface for errSetField.
//
// The path and the cause are shortened, so a long value sent by the client is never
// repeated in full.
//
// Returns string which describes the field binding failure with context.
func (e errSetField) Error() string {
	return fmt.Sprintf("binder: could not set field '%s' for path '%s' (expected type %s): %s",
		runeutil.Truncate(e.field, maxQuotedValueRunes),
		runeutil.Truncate(e.path, maxQuotedValueRunes),
		e.fieldType,
		describeCause(e.err),
	)
}

// SafeMessage returns a user-safe error message that does not expose field names or
// internal type information.
//
// Returns string which is a generic binding failure message.
func (errSetField) SafeMessage() string { return "invalid form data" }

// Unwrap exposes the underlying cause so errors.Is and errors.As can match it.
//
// Returns error which is the cause of the field assignment failing.
func (e errSetField) Unwrap() error { return e.err }

// errFieldErrorsTruncated marks a MultiError that the binder trimmed to its limit. It is
// stored under FieldErrorsTruncatedKey.
type errFieldErrorsTruncated struct {
	// reported is how many fields the trimmed map kept.
	reported int64

	// total is how many fields failed before the map was trimmed.
	total int
}

// Error implements the error interface for errFieldErrorsTruncated.
//
// Returns string which says how many of the failed fields are reported.
func (e errFieldErrorsTruncated) Error() string {
	return fieldErrorsTruncatedNotice(e.reported, e.total)
}

// limitError reports that a request broke one of the binder's size limits, such as the
// number of fields or slice elements it may hold.
//
// It is not a problem with any one field, so it is never stored in a MultiError, and it
// answers with 413 rather than 422.
type limitError struct {
	// err describes the limit that was broken and wraps its sentinel.
	err error
}

// Error implements the error interface for limitError.
//
// Returns string which describes the limit that was broken.
func (e *limitError) Error() string {
	return e.err.Error()
}

// Unwrap exposes the cause so errors.Is reaches ErrTooManyFields or
// ErrSliceElementBudgetExhausted.
//
// Returns error which describes the limit that was broken.
func (e *limitError) Unwrap() error {
	return e.err
}

// StatusCode reports the HTTP status a limit breach answers with.
//
// Returns int which is 413.
func (*limitError) StatusCode() int {
	return http.StatusRequestEntityTooLarge
}

// ErrorCode reports the machine-readable code a limit breach answers with.
//
// Returns string which is PAYLOAD_TOO_LARGE.
func (*limitError) ErrorCode() string {
	return errorCodePayloadTooLarge
}

// SafeMessage returns a message that names the problem without the limits or values.
//
// Returns string which is safe to show to the client.
func (*limitError) SafeMessage() string {
	return limitSafeMessage
}

// malformedJSONError reports that BindJSON was given bytes that are not a JSON object.
//
// It is a problem with the whole body rather than with any one field, so it answers with
// 400, and its safe message says nothing about where the decoder stopped.
type malformedJSONError struct {
	// err is the decoder's error.
	err error
}

// Error implements the error interface for malformedJSONError.
//
// Returns string which wraps the decoder's error for the logs.
func (e *malformedJSONError) Error() string {
	return "decoding JSON for binding: " + e.err.Error()
}

// Unwrap exposes the decoder's error so errors.As can reach it.
//
// Returns error which is the decoder's error.
func (e *malformedJSONError) Unwrap() error {
	return e.err
}

// StatusCode reports the HTTP status a malformed body answers with.
//
// Returns int which is 400.
func (*malformedJSONError) StatusCode() int {
	return http.StatusBadRequest
}

// ErrorCode reports the machine-readable code a malformed body answers with.
//
// Returns string which is BAD_REQUEST.
func (*malformedJSONError) ErrorCode() string {
	return errorCodeBadRequest
}

// SafeMessage returns a message that names the problem without the decoder's text.
//
// Returns string which is safe to show to the client.
func (*malformedJSONError) SafeMessage() string {
	return malformedJSONSafeMessage
}

// newLimitError wraps err as a breach of a size limit.
//
// Takes err (error) which describes the limit and wraps its sentinel.
//
// Returns error which answers with 413 and PAYLOAD_TOO_LARGE.
func newLimitError(err error) error {
	return &limitError{err: err}
}

// describeCause renders an underlying error for an error message without repeating a long
// value in full.
//
// A number that would not parse has its quoted value shortened so its reason survives,
// and the whole text is then shortened as well.
//
// Takes err (error) which is the underlying error.
//
// Returns string which is the shortened description.
func describeCause(err error) string {
	if err == nil {
		return "unknown cause"
	}

	message := err.Error()
	if numberErr, ok := errors.AsType[*strconv.NumError](err); ok {
		message = strings.Replace(message,
			strconv.Quote(numberErr.Num),
			strconv.Quote(runeutil.Truncate(numberErr.Num, maxQuotedValueRunes)),
			1,
		)
	}
	return runeutil.Truncate(message, maxCauseRunes)
}

// fieldMessage chooses the message a client is shown for one field that could not be
// bound.
//
// Takes err (error) which is the binding failure recorded for the field.
//
// Returns string which says what the field expects.
func fieldMessage(err error) string {
	if errors.Is(err, strconv.ErrRange) {
		return fieldMessageNumberOutOfRange
	}
	if errors.Is(err, strconv.ErrSyntax) {
		return syntaxFieldMessage(err)
	}
	return fieldMessageInvalidValue
}

// syntaxFieldMessage chooses the message for a value strconv could not read, based on the
// kind of value the field holds.
//
// Takes err (error) which wraps strconv.ErrSyntax.
//
// Returns string which names the kind of value the field expects, or the general message
// when the kind is not known.
func syntaxFieldMessage(err error) string {
	numberErr, ok := errors.AsType[*strconv.NumError](err)
	if !ok {
		return fieldMessageInvalidValue
	}

	message, known := syntaxFieldMessages[numberErr.Func]
	if !known {
		return fieldMessageInvalidValue
	}
	return message
}

// fieldErrorsTruncatedNotice describes a field-error list that was trimmed to its limit.
//
// Takes reported (int64) which is how many fields the list kept.
// Takes total (int) which is how many fields failed.
//
// Returns string which is stored under FieldErrorsTruncatedKey.
func fieldErrorsTruncatedNotice(reported int64, total int) string {
	return fmt.Sprintf("reporting %d of %d failed fields", reported, total)
}
