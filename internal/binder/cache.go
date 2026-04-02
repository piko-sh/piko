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
	"encoding"
	"image/color"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko/wdk/maths"
)

// fieldInfo holds pre-computed metadata for a single struct field.
// Fields are ordered to reduce memory use through alignment.
type fieldInfo struct {
	// Type is the reflect.Type of the field, used for conversion and validation.
	Type reflect.Type

	// unmarshaler holds the TextUnmarshaler interface if the field type implements it.
	unmarshaler encoding.TextUnmarshaler

	// Path is the struct field path used in error messages.
	Path string

	// Index is the struct field index path for accessing nested fields.
	Index []int

	// Offset is the byte offset from the struct start for direct memory access.
	Offset uintptr

	// Kind is the reflect.Kind of the field, used for fast type switching.
	Kind reflect.Kind

	// CanDirect indicates whether the field can be set using direct unsafe access.
	// True for single-level, non-pointer basic types.
	CanDirect bool
}

// structInfo holds cached metadata for a struct type.
type structInfo struct {
	// Fields maps a Piko expression path to its field metadata.
	Fields map[string]*fieldInfo
}

// parseFieldPath extracts the field path from a struct field tag.
//
// It checks tags in this order: bind tag first, then json tag, then the Go
// field name. The function has no side effects.
//
// Takes field (*reflect.StructField) which points to the struct field to get
// the path from. A pointer is used to satisfy the hugeParam linter.
// Takes parentPath (string) which is the dot-separated path of the parent
// struct for nested fields.
//
// Returns currentPath (string) which is the full field path, joined with
// parentPath if one is provided.
// Returns ignored (bool) which is true when the field should be skipped
// because its tag value is "-".
func parseFieldPath(field *reflect.StructField, parentPath string) (currentPath string, ignored bool) {
	var fieldName string

	tag := field.Tag.Get("bind")
	if tag != "" {
		if tag == "-" {
			return "", true
		}
		fieldName, _, _ = strings.Cut(tag, ",")
	}

	if fieldName == "" {
		tag = field.Tag.Get("json")
		if tag != "" {
			if tag == "-" {
				return "", true
			}
			fieldName, _, _ = strings.Cut(tag, ",")
		}
	}

	if fieldName == "" {
		fieldName = field.Name
	}

	if parentPath != "" {
		return parentPath + "." + fieldName, false
	}
	return fieldName, false
}

// isCustomType checks if a type has built-in conversion logic and should be
// treated as a terminal field in the cache walk. This stops the walk from
// going into the inner fields of these types.
//
// Takes t (reflect.Type) which specifies the type to check.
//
// Returns bool which is true if the type has built-in conversion logic.
func isCustomType(t reflect.Type) bool {
	switch t {
	case reflect.TypeFor[time.Time](),
		reflect.TypeFor[time.Duration](),
		reflect.TypeFor[uuid.UUID](),
		reflect.TypeFor[url.URL](),
		reflect.TypeFor[net.IP](),
		reflect.TypeFor[mail.Address](),
		reflect.TypeFor[color.RGBA](),
		reflect.TypeFor[maths.Decimal](),
		reflect.TypeFor[maths.Money]():
		return true
	default:
		return false
	}
}

// isPrimitiveKind reports whether the given kind is a simple type that can be
// set directly using unsafe pointer casting without reflection.
//
// Takes k (reflect.Kind) which is the kind to check.
//
// Returns bool which is true if the kind is a simple type.
func isPrimitiveKind(k reflect.Kind) bool {
	switch k {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// hasWellKnownConverter reports whether the type has a built-in converter
// that needs special parsing logic. For example, time.Duration is stored as
// int64 but must be parsed with time.ParseDuration instead of a simple cast.
//
// Takes t (reflect.Type) which is the type to check.
//
// Returns bool which is true if the type has a well-known converter.
func hasWellKnownConverter(t reflect.Type) bool {
	switch t {
	case reflect.TypeFor[time.Time](),
		reflect.TypeFor[time.Duration](),
		reflect.TypeFor[url.URL](),
		reflect.TypeFor[mail.Address](),
		reflect.TypeFor[color.RGBA](),
		reflect.TypeFor[maths.Decimal](),
		reflect.TypeFor[maths.Money]():
		return true
	default:
		return false
	}
}

// implementsTextUnmarshaler checks whether a type or its pointer form implements
// the encoding.TextUnmarshaler interface.
//
// Takes t (reflect.Type) which is the type to check.
//
// Returns encoding.TextUnmarshaler which is an instance that can call
// UnmarshalText, or nil if the type does not implement the interface.
// Returns bool which is true when the type implements the interface.
func implementsTextUnmarshaler(t reflect.Type) (encoding.TextUnmarshaler, bool) {
	ptr := reflect.New(t)

	if unmarshaler, ok := reflect.TypeAssert[encoding.TextUnmarshaler](ptr); ok {
		return unmarshaler, true
	}

	if unmarshaler, ok := reflect.TypeAssert[encoding.TextUnmarshaler](ptr.Elem()); ok {
		return unmarshaler, true
	}

	return nil, false
}
