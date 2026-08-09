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
	"context"
	"reflect"

	"piko.sh/piko/internal/binder"
)

// ConverterFunc defines a function that converts a string value into a custom type during
// form binding.
//
// Example:
//
//	type MyCustomID string
//	piko.RegisterConverter(MyCustomID(""), func(value string) (reflect.Value, error) {
//	    return reflect.ValueOf(MyCustomID("ID-" + value)), nil
//	})
type ConverterFunc = binder.ConverterFunc

// Option is a functional option for setting up a Bind operation. Options can change
// global settings for a single Bind call.
type Option = binder.Option

// Bind populates the fields of the destination struct from the source data.
//
// Takes destination (any) which is the destination struct to populate.
// Takes source (map[string][]string) which contains the source data to bind from.
// Takes opts (...Option) which provides optional settings that override global
// configuration for this call.
//
// Returns error when binding fails due to type mismatch or validation errors.
//
// Example with options:
//
//	err := piko.Bind(&form, data,
//	    piko.IgnoreUnknownKeys(true),
//	    piko.WithMaxFieldCount(500),
//	)
func Bind(ctx context.Context, destination any, source map[string][]string, opts ...Option) error {
	return binder.GetBinder().Bind(ctx, destination, source, opts...)
}

// BindMap populates the fields of the destination struct from a map[string]any, typically
// produced by JSON decoding. It flattens the nested map into bracket-notation form data
// and delegates to the standard Bind pipeline.
//
// Takes destination (any) which is the destination struct to populate.
// Takes source (map[string]any) which contains the source data to bind from.
// Takes opts (...Option) which provides optional settings that override global
// configuration for this call.
//
// Returns error when binding fails due to type mismatch or validation errors.
//
// Example:
//
//	var form MyStruct
//	err := piko.BindMap(&form, jsonMap,
//	    piko.IgnoreUnknownKeys(true),
//	)
func BindMap(ctx context.Context, destination any, source map[string]any, opts ...Option) error {
	return binder.GetBinder().BindMap(ctx, destination, source, opts...)
}

// BindJSON populates the fields of the destination struct from raw JSON bytes. It decodes
// the JSON into a map, then delegates to BindMap.
//
// Takes destination (any) which is the destination struct to populate.
// Takes source ([]byte) which contains the raw JSON bytes to decode.
// Takes opts (...Option) which provides optional settings that override global
// configuration for this call.
//
// Returns error when JSON decoding fails or binding errors occur.
//
// Example:
//
//	var form MyStruct
//	err := piko.BindJSON(&form, jsonBytes,
//	    piko.IgnoreUnknownKeys(true),
//	)
func BindJSON(ctx context.Context, destination any, source []byte, opts ...Option) error {
	return binder.GetBinder().BindJSON(ctx, destination, source, opts...)
}

// IgnoreUnknownKeys returns an Option that controls how unknown form fields are handled
// during binding.
//
// When set to true, the binder silently ignores fields in the source data that do not map
// to a field in the destination struct. When set to false, it returns an error for each
// unknown key. This overrides the global setting for a single Bind call.
//
// Takes ignore (bool) which specifies whether to ignore unknown keys.
//
// Returns Option which configures unknown key handling for a Bind call.
func IgnoreUnknownKeys(ignore bool) Option {
	return binder.IgnoreUnknownKeys(ignore)
}

// StructValidator validates a populated struct against its field tags.
type StructValidator = binder.StructValidator

// ValidationFailedError reports that binding succeeded but the destination violated its
// `validate:"..."` tags. It answers 422 and carries per-field messages when the
// configured validator could supply them.
type ValidationFailedError = binder.ValidationFailedError

var (
	// ErrValidationFailed reports that a bound destination violated its `validate:"..."`
	// tags. Use errors.Is against it to detect a validation failure from a Bind call that
	// opted in via WithValidation.
	ErrValidationFailed = binder.ErrValidationFailed
)

// WithValidation returns an Option that runs the configured struct validator against the
// destination once binding succeeds, so `validate:"..."` tags are enforced.
//
// Validation is opt-in per call rather than global: binding is also used for
// partially-filled payloads, such as a form partial re-rendering mid-edit, where
// rejecting incomplete input would be wrong. Generated action wrappers set this because
// an action input is expected to be complete.
//
// Has no effect unless a validator is configured, which piko.WithValidator does.
//
// Takes validate (bool) which specifies whether to validate after binding.
//
// Returns Option which configures post-bind validation for a Bind call.
func WithValidation(validate bool) Option {
	return binder.WithValidation(validate)
}

// SetStructValidator installs the validator used by Bind calls that opt in via
// WithValidation. Passing nil disables validation.
//
// Applications do not normally call this: piko.WithValidator records the validator and
// framework initialisation installs it here, before any request is served.
//
// Takes v (StructValidator) which validates bound destinations.
func SetStructValidator(v StructValidator) {
	binder.GetBinder().SetStructValidator(v)
}

// ActionInputSource selects the map an action input binds from, given the request's
// decoded arguments and that input's parameter name. Generated action wrappers call it so
// nested and flat payloads bind alike.
//
// It always returns a map, so the wrapper binds even when the payload is absent or
// malformed and the input's `validate:"..."` tags still decide the outcome.
//
// Takes arguments (map[string]any) which are the request's decoded arguments.
// Takes key (string) which is the input's parameter name.
//
// Returns map[string]any which is the source to bind from.
func ActionInputSource(arguments map[string]any, key string) map[string]any {
	return binder.ActionInputSource(arguments, key)
}

// WithMaxSliceSize returns an Option to set a per-call limit for slice growth. This
// overrides the global limit for this specific Bind call.
//
// Takes size (int) which specifies the maximum number of elements allowed.
//
// Returns Option which configures the slice size limit for a Bind call.
func WithMaxSliceSize(size int) Option {
	return binder.WithMaxSliceSize(size)
}

// WithMaxPathDepth returns an Option to set a per-call limit for path depth. This
// overrides the global limit for this specific Bind call.
//
// Takes depth (int) which specifies the maximum path depth allowed.
//
// Returns Option which configures the path depth limit for a Bind call.
func WithMaxPathDepth(depth int) Option {
	return binder.WithMaxPathDepth(depth)
}

// WithMaxPathLength returns an Option to set a per-call limit for path length. This
// overrides the global limit for this specific Bind call.
//
// Takes length (int) which specifies the maximum path length allowed.
//
// Returns Option which applies the path length limit when passed to Bind.
func WithMaxPathLength(length int) Option {
	return binder.WithMaxPathLength(length)
}

// WithDocumentScaleLimits returns an Option that raises the width limits to document
// scale in one step, covering field count, slice size, and value length.
//
// Returns Option which raises the width limits to document scale.
func WithDocumentScaleLimits() Option {
	return binder.WithDocumentScaleLimits()
}

// WithMaxFieldCount returns an Option to set a per-call limit for field count. This
// overrides the global limit for this specific Bind call.
//
// Takes count (int) which specifies the maximum number of fields allowed.
//
// Returns Option which configures the field count limit for a Bind call.
func WithMaxFieldCount(count int) Option {
	return binder.WithMaxFieldCount(count)
}

// WithMaxValueLength returns an Option to set a per-call limit for value length. This
// overrides the global limit for this specific Bind call.
//
// Takes length (int) which specifies the maximum allowed value length.
//
// Returns Option which configures the value length limit for a Bind call.
func WithMaxValueLength(length int) Option {
	return binder.WithMaxValueLength(length)
}

// RegisterConverter registers a custom function to convert a string form value into a
// specific type. This is used for handling custom data types in form submissions that are
// not natively supported by the binder.
//
// Takes typ (any) which is a zero value of the target type (e.g., MyType{}).
// Takes converter (ConverterFunc) which performs the string-to-type conversion.
func RegisterConverter(typ any, converter ConverterFunc) {
	binder.GetBinder().RegisterConverter(reflect.TypeOf(typ), converter)
}

// SetMaxFormFields sets the maximum number of fields allowed in a form submission. This
// is a security measure to prevent hash-flooding DoS attacks.
//
// Takes count (int) which specifies the maximum number of form fields. A value of 0 means
// no limit. The default is 1000.
func SetMaxFormFields(count int) {
	binder.GetBinder().SetMaxFieldCount(count)
}

// SetMaxFormPathLength sets the maximum length in characters for a single form field
// name.
//
// This prevents CPU and memory exhaustion from parsing overly long path strings. The
// default is 4096. A value of 0 means no limit.
//
// Takes length (int) which specifies the maximum path length in characters.
func SetMaxFormPathLength(length int) {
	binder.GetBinder().SetMaxPathLength(length)
}

// SetMaxFormValueLength sets the maximum length of a single form field value.
//
// Takes length (int) which specifies the maximum characters allowed.
//
// This prevents memory exhaustion from malicious TextUnmarshaler implementations or huge
// inputs. The default is 65536 (64KB). A value of 0 means no limit.
func SetMaxFormValueLength(length int) {
	binder.GetBinder().SetMaxValueLength(length)
}

// SetMaxFormPathDepth sets the maximum nesting depth for form paths.
//
// Form paths use dot notation such as "a.b.c.d". This limit is a security measure to
// prevent stack overflow from deeply nested paths. The default is 32. A value of 0 means
// no limit.
//
// Takes depth (int) which specifies the maximum allowed path depth.
func SetMaxFormPathDepth(depth int) {
	binder.GetBinder().SetMaxPathDepth(depth)
}

// SetMaxSliceSize sets the maximum allowed index for slices in form binding.
//
// This is a security measure to prevent memory exhaustion attacks from inputs like
// "items[9999999]". The default is 1000. A value of 0 means no limit.
//
// Takes size (int) which specifies the maximum slice index allowed.
func SetMaxSliceSize(size int) {
	binder.GetBinder().SetMaxSliceSize(size)
}

// SetIgnoreUnknownKeys sets the global default for ignoring unknown form fields.
//
// Takes ignore (bool) which when true causes the binder to silently ignore fields in the
// source data that do not map to a field in the destination struct. When false (the
// default), the binder returns an error for each unknown key. This setting can be
// overridden on a per-call basis using the IgnoreUnknownKeys option.
func SetIgnoreUnknownKeys(ignore bool) {
	binder.GetBinder().SetIgnoreUnknownKeys(ignore)
}
