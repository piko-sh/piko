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

// BindOptions holds settings for a single Bind call. Each field is a pointer
// so that nil means "not set" and a zero value means "set to zero".
type BindOptions struct {
	// IgnoreUnknownKeys controls whether unknown form fields cause errors.
	IgnoreUnknownKeys *bool

	// MaxSliceSize limits how many elements a slice may hold; nil uses the default.
	MaxSliceSize *int

	// MaxPathDepth limits how deep nested path lookups can go; nil uses the default.
	MaxPathDepth *int

	// MaxPathLength sets the limit for path length; nil uses the default.
	MaxPathLength *int

	// MaxFieldCount limits the number of fields in a struct; nil uses the default.
	MaxFieldCount *int

	// MaxValueLength sets the maximum length for bound values; nil uses the default.
	MaxValueLength *int
}

// Option is a function that configures BindOptions.
type Option func(*BindOptions)

// IgnoreUnknownKeys returns an Option that controls behaviour for unknown form
// fields in the source data.
//
// When ignore is true, the binder will silently ignore fields that do not map
// to a field in the destination struct. When false, it will return an error
// for each unknown key.
//
// Takes ignore (bool) which specifies whether to ignore unknown keys.
//
// Returns Option which configures the binder's unknown key handling.
func IgnoreUnknownKeys(ignore bool) Option {
	return func(opts *BindOptions) {
		opts.IgnoreUnknownKeys = &ignore
	}
}

// WithMaxSliceSize returns an Option that sets a limit for slice growth on a
// single call. This overrides the global limit for this specific Bind call.
//
// Takes size (int) which specifies the maximum number of elements allowed.
//
// Returns Option which sets the slice size limit when applied.
func WithMaxSliceSize(size int) Option {
	return func(opts *BindOptions) {
		opts.MaxSliceSize = &size
	}
}

// WithMaxPathDepth returns an Option that sets a limit for path depth on a
// single call. This overrides the global limit for the specific Bind call.
//
// Takes depth (int) which specifies the maximum allowed path depth.
//
// Returns Option which sets the path depth limit when applied.
func WithMaxPathDepth(depth int) Option {
	return func(opts *BindOptions) {
		opts.MaxPathDepth = &depth
	}
}

// WithMaxPathLength returns an Option that sets the maximum path length for a
// single call. This overrides the global limit for that Bind call.
//
// Takes length (int) which specifies the maximum path length allowed.
//
// Returns Option which applies the path length limit when called.
func WithMaxPathLength(length int) Option {
	return func(opts *BindOptions) {
		opts.MaxPathLength = &length
	}
}

// WithMaxFieldCount returns an Option that sets a limit for how many fields
// are allowed in a single Bind call. This overrides the global limit.
//
// Takes count (int) which specifies the maximum number of fields allowed.
//
// Returns Option which sets the field count limit when applied.
func WithMaxFieldCount(count int) Option {
	return func(opts *BindOptions) {
		opts.MaxFieldCount = &count
	}
}

// WithMaxValueLength returns an Option that sets the maximum value length for
// a single call. This overrides the global limit for this specific Bind call.
//
// Takes length (int) which specifies the maximum allowed length for a value.
//
// Returns Option which applies the value length limit when used.
func WithMaxValueLength(length int) Option {
	return func(opts *BindOptions) {
		opts.MaxValueLength = &length
	}
}
