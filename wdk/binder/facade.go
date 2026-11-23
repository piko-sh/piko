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
//	binder.RegisterConverter(MyCustomID(""), func(value string) (reflect.Value, error) {
//	    return reflect.ValueOf(MyCustomID("ID-" + value)), nil
//	})
type ConverterFunc = binder.ConverterFunc

// Option is a functional option for setting up a Bind operation. Options override the
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
//	err := binder.Bind(ctx, &form, request.Form,
//	    binder.IgnoreUnknownKeys(true),
//	    binder.WithMaxFieldCount(500),
//	)
func Bind(ctx context.Context, destination any, source map[string][]string, opts ...Option) error {
	return binder.GetBinder().Bind(ctx, destination, source, opts...)
}

// BindMap populates the fields of the destination struct from a map[string]any, typically
// produced by JSON decoding.
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
//	err := binder.BindMap(ctx, &form, jsonMap, binder.IgnoreUnknownKeys(true))
func BindMap(ctx context.Context, destination any, source map[string]any, opts ...Option) error {
	return binder.GetBinder().BindMap(ctx, destination, source, opts...)
}

// BindJSON populates a destination struct from raw JSON bytes.
//
// Takes destination (any) which is the destination struct to populate.
// Takes source ([]byte) which contains the raw JSON bytes to decode.
// Takes opts (...Option) which provides optional settings that override global
// configuration for this call.
//
// Returns error when JSON decoding fails or binding errors occur. Input larger than the
// configured byte limit wraps ErrBindJSONTooLarge.
//
// Example:
//
//	var form MyStruct
//	err := binder.BindJSON(ctx, &form, jsonBytes, binder.IgnoreUnknownKeys(true))
func BindJSON(ctx context.Context, destination any, source []byte, opts ...Option) error {
	return binder.GetBinder().BindJSON(ctx, destination, source, opts...)
}

// IgnoreUnknownKeys overrides the unknown-field policy for one Bind call.
//
// Takes ignore (bool) which allows unknown keys when true and reports them as errors when
// false.
//
// Returns Option which sets the field policy.
func IgnoreUnknownKeys(ignore bool) Option {
	return binder.IgnoreUnknownKeys(ignore)
}

// StructValidator validates a populated struct against its field tags.
type StructValidator = binder.StructValidator

// ValidationFailedError reports that binding succeeded but the destination violated its
// `validate:"..."` tags.
type ValidationFailedError = binder.ValidationFailedError

var (
	// ErrBindJSONTooLarge matches BindJSON failures when the request body exceeds the
	// configured byte limit.
	ErrBindJSONTooLarge = binder.ErrBindJSONTooLarge

	// ErrValidationFailed reports that a bound destination violated its `validate:"..."`
	// tags.
	ErrValidationFailed = binder.ErrValidationFailed
)

// WithValidation validates the destination after binding succeeds.
//
// Validation has no effect unless a struct validator is configured.
//
// Takes validate (bool) which specifies whether to validate after binding.
//
// Returns Option which configures post-bind validation for a Bind call.
func WithValidation(validate bool) Option {
	return binder.WithValidation(validate)
}

// SetStructValidator installs the validator used by Bind calls that opt in via
// WithValidation.
//
// Takes v (StructValidator) which validates bound destinations.
func SetStructValidator(v StructValidator) {
	binder.GetBinder().SetStructValidator(v)
}

// ActionInputSource selects the map an action input binds from, given the request's
// decoded arguments and that input's parameter name.
//
// Takes arguments (map[string]any) which are the request's decoded arguments.
// Takes key (string) which is the input's parameter name.
//
// Returns map[string]any which is the source to bind from.
func ActionInputSource(arguments map[string]any, key string) map[string]any {
	return binder.ActionInputSource(arguments, key)
}

// WithMaxSliceSize returns an Option to set a per-call limit for slice growth. It
// overrides the global limit for this Bind call only.
//
// Takes size (int) which specifies the maximum number of elements allowed.
//
// Returns Option which configures the slice size limit for a Bind call.
func WithMaxSliceSize(size int) Option {
	return binder.WithMaxSliceSize(size)
}

// WithMaxPathDepth returns an Option to set a per-call limit for path depth. It overrides
// the global limit for this Bind call only.
//
// Takes depth (int) which specifies the maximum path depth allowed.
//
// Returns Option which configures the path depth limit for a Bind call.
func WithMaxPathDepth(depth int) Option {
	return binder.WithMaxPathDepth(depth)
}

// WithMaxPathLength returns an Option to set a per-call limit for path length. It
// overrides the global limit for this Bind call only.
//
// Takes length (int) which specifies the maximum path length allowed.
//
// Returns Option which applies the path length limit when passed to Bind.
func WithMaxPathLength(length int) Option {
	return binder.WithMaxPathLength(length)
}

// WithDocumentScaleLimits returns an Option that raises the width limits to document
// scale in one step, covering field count, slice size, and value length. It overrides the
// global limits for this Bind call only.
//
// Returns Option which raises the width limits to document scale.
func WithDocumentScaleLimits() Option {
	return binder.WithDocumentScaleLimits()
}

// WithMaxFieldCount returns an Option to set a per-call limit for field count. It
// overrides the global limit for this Bind call only.
//
// Takes count (int) which specifies the maximum number of fields allowed.
//
// Returns Option which configures the field count limit for a Bind call.
func WithMaxFieldCount(count int) Option {
	return binder.WithMaxFieldCount(count)
}

// WithMaxValueLength returns an Option to set a per-call limit for value length. It
// overrides the global limit for this Bind call only.
//
// Takes length (int) which specifies the maximum allowed value length.
//
// Returns Option which configures the value length limit for a Bind call.
func WithMaxValueLength(length int) Option {
	return binder.WithMaxValueLength(length)
}

// RegisterConverter registers a custom function to convert a string form value into a
// specific type.
//
// Takes typ (any) which is a zero value of the target type (e.g., MyType{}).
// Takes converter (ConverterFunc) which performs the string-to-type conversion.
func RegisterConverter(typ any, converter ConverterFunc) {
	binder.GetBinder().RegisterConverter(reflect.TypeOf(typ), converter)
}

// SetMaxFormFields sets the maximum number of fields allowed in a form submission.
//
// Takes count (int) which specifies the maximum number of form fields. A value of 0 means
// no limit. The default is 1000.
func SetMaxFormFields(count int) {
	binder.GetBinder().SetMaxFieldCount(count)
}

// SetMaxFormPathLength sets the maximum length in characters for a single form field
// name.
//
// Takes length (int) which specifies the maximum path length in characters. The default
// is 4096, and zero disables the limit.
func SetMaxFormPathLength(length int) {
	binder.GetBinder().SetMaxPathLength(length)
}

// SetMaxFormValueLength sets the maximum length of a single form field value.
//
// Takes length (int) which specifies the maximum characters allowed. The default is
// 65536, and zero disables the limit.
func SetMaxFormValueLength(length int) {
	binder.GetBinder().SetMaxValueLength(length)
}

// SetMaxFormPathDepth sets the maximum nesting depth for form paths.
//
// Takes depth (int) which specifies the maximum allowed path depth. The default is 32,
// and zero disables the limit.
func SetMaxFormPathDepth(depth int) {
	binder.GetBinder().SetMaxPathDepth(depth)
}

// SetMaxSliceSize sets the maximum allowed index for slices in form binding.
//
// Takes size (int) which specifies the maximum slice index allowed. The default is 1000,
// and zero disables the limit.
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
