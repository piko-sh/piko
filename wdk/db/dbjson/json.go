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

// Package dbjson provides JSON, a nil-capable database/sql scan target and value source
// for JSON, JSONB, and union/variant columns across every supported engine.
package dbjson

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"piko.sh/piko/internal/json"
)

// ScanInto returns a sql.Scanner that JSON-decodes a column into dest. It is used for
// array and composite columns the query surfaced as JSON via to_json(), so the driver's
// JSON bytes decode straight into the typed destination (for example []string or
// [][]int).
//
// A SQL NULL or an empty value leaves dest at its zero value, so a nullable array column
// scans to a nil slice rather than erroring.
//
// Takes dest (*T) which is the typed destination to decode the JSON column into.
//
// Returns sql.Scanner which decodes the column on Scan.
func ScanInto[T any](dest *T) sql.Scanner {
	return jsonScanner[T]{dest: dest}
}

// jsonScanner is the sql.Scanner returned by ScanInto.
type jsonScanner[T any] struct {
	// dest is the typed destination the JSON column is decoded into.
	dest *T
}

// Scan implements sql.Scanner by JSON-decoding the driver value into the destination.
//
// Takes source (any) which is the driver value: nil, raw JSON []byte, or a JSON string.
//
// Returns error when the value is non-empty and cannot be decoded as JSON, or when the
// driver hands back an unexpected type.
func (s jsonScanner[T]) Scan(source any) error {
	var raw []byte
	switch value := source.(type) {
	case nil:
		var zero T
		*s.dest = zero
		return nil
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:

		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("dbjson: re-encode %T to JSON: %w", source, err)
		}
		raw = encoded
	}

	if len(raw) == 0 {
		var zero T
		*s.dest = zero
		return nil
	}
	if err := json.Unmarshal(raw, s.dest); err != nil {
		return fmt.Errorf("dbjson: decode JSON column: %w", err)
	}
	return nil
}

// JSON holds the raw JSON encoding of a column as bytes and round-trips it through
// database/sql. Unlike a bare json.RawMessage it implements sql.Scanner, so a SQL NULL
// and the different shapes drivers surface JSON in (raw bytes, a string, or a decoded
// value) all scan without error.
type JSON []byte

// Scan implements sql.Scanner. It accepts the JSON shapes the supported drivers return: a
// NULL (nil) clears the value; raw []byte is copied; a string (SQLite) is stored as
// bytes; any other value (a DuckDB map) is re-encoded through the configured JSON
// provider.
//
// Takes source (any) which is the driver value for the column.
//
// Returns error when a non-byte, non-string value cannot be encoded to JSON.
func (j *JSON) Scan(source any) error {
	switch value := source.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		clone := make(JSON, len(value))
		copy(clone, value)
		*j = clone
		return nil
	case string:
		*j = JSON(value)
		return nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("dbjson: encode %T to JSON: %w", source, err)
		}
		*j = encoded
		return nil
	}
}

// Value implements driver.Valuer. A nil value round-trips as SQL NULL; otherwise the raw
// bytes are passed through unchanged so the driver writes them to the JSON column.
//
// Returns driver.Value which is nil or the raw JSON bytes.
// Returns error which is always nil.
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON returns the raw bytes so JSON behaves like json.RawMessage when embedded in
// a larger document. A nil value encodes as the JSON null literal.
//
// Returns []byte which is the raw JSON encoding.
// Returns error which is always nil.
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON stores a copy of the raw bytes without decoding them, matching the
// pass-through semantics of json.RawMessage.
//
// Takes data ([]byte) which is the raw JSON to retain.
//
// Returns error which is always nil.
func (j *JSON) UnmarshalJSON(data []byte) error {
	clone := make(JSON, len(data))
	copy(clone, data)
	*j = clone
	return nil
}
