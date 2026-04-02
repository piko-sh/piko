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

// Package binder provides form-data binding for populating Go structs
// from key-value form submissions.
//
// It supports nested structs, slices, maps, and custom type converters.
// Configurable security limits protect against hash-flooding, memory
// exhaustion, and stack overflow from malicious input.
//
// # Binding form data
//
// Use [Bind] to populate a struct from form data:
//
//	var form struct {
//	    Name  string   `form:"name"`
//	    Email string   `form:"email"`
//	    Tags  []string `form:"tags"`
//	}
//
//	err := binder.Bind(&form, r.Form)
//
// Per-call options override global defaults:
//
//	err := binder.Bind(&form, data,
//	    binder.IgnoreUnknownKeys(true),
//	    binder.WithMaxFieldCount(500),
//	)
//
// # Custom type converters
//
// Register a [ConverterFunc] to handle types not natively supported:
//
//	type UserID string
//
//	binder.RegisterConverter(UserID(""), func(v string) (reflect.Value, error) {
//	    return reflect.ValueOf(UserID("uid-" + v)), nil
//	})
//
// # Security limits
//
// Global limits control field count, nesting depth, path length,
// value length, and slice size. These can be set at startup with
// the Set* functions, or overridden per call with the corresponding
// option functions.
package binder
