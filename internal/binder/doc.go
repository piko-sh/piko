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

// Package binder bridges HTTP form data with Go struct population, using the
// Piko Expression Language for field mapping.
//
// It is a native replacement for gorilla/schema, supporting nested structs,
// slices, maps, and custom type converters with built-in DoS protection.
//
// # Usage
//
// Use the shared binder instance for typical form binding:
//
//	type Form struct {
//	    Name  string `bind:"name"`
//	    Email string `bind:"email"`
//	    Age   int    `bind:"age"`
//	}
//
//	var form Form
//	err := binder.GetBinder().Bind(&form, r.Form)
//
// For nested paths and slice indexing, the binder parses Piko
// expressions:
//
//	// Form data: {"user.address.city": ["London"],
//	//             "items[0]": ["apple"]}
//	type Order struct {
//	    User  User     `bind:"user"`
//	    Items []string `bind:"items"`
//	}
//
// # DoS protection
//
// The binder includes configurable limits to prevent resource
// exhaustion:
//
//   - MaxSliceSize: Limits slice index values (default: 1000)
//   - MaxPathDepth: Limits nesting depth (default: 32)
//   - MaxPathLength: Limits path string length (default: 4096 bytes)
//   - MaxFieldCount: Limits form field count (default: 1000)
//   - MaxValueLength: Limits field value length (default: 64KB)
//
// Limits can be set globally or per-call using functional options:
//
//	b := binder.GetBinder()
//	b.SetMaxSliceSize(500)
//
//	// Or per-call:
//	err := b.Bind(&form, data, binder.WithMaxSliceSize(100))
//
// # Custom converters
//
// Register custom type converters for non-standard types:
//
//	b.RegisterConverter(
//	    reflect.TypeOf(MyType{}),
//	    func(s string) (reflect.Value, error) {
//	        // Parse and return the value
//	    },
//	)
//
// The binder also supports [encoding.TextUnmarshaler] for automatic
// conversion. Built-in converters exist for [time.Time],
// [time.Duration], [net/url.URL], [net/mail.Address],
// [image/colour.Colour], and Piko's [maths.Decimal] and [maths.Money]
// types.
//
// # Integration
//
// The binder uses [ast_domain.ExpressionParser] to parse complex
// field paths (e.g. "user.addresses[0].city") into AST nodes, which
// are then walked to navigate the destination struct. Parsed ASTs are
// cached for repeated use. Simple identifier paths bypass the parser
// entirely for a fast path.
//
// # Thread safety
//
// All methods on [ASTBinder] are safe for concurrent use. The shared
// instance returned by [GetBinder] caches struct metadata for improved
// performance. Converter registration and limit updates use atomic
// operations and [sync.Map] for lock-free reads.
package binder
