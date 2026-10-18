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
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/hex"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
)

// ScanInto returns a sql.Scanner that decodes a JSON column into dest.
//
// SQL NULL and empty values leave dest at its zero value. Null array elements or object
// values require nullable Go types. Database dates and timestamps decode to time.Time,
// with unzoned timestamps read as UTC. Numbers can decode to strings or
// text-unmarshalling types, and hexadecimal byte strings decode to []byte.
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
// Takes source (any) which holds nil, raw JSON []byte, a JSON string, or an already
// decoded value, which is re-encoded first.
//
// Returns error when the value is non-empty and cannot be decoded as JSON into the
// destination, when it holds a null element the destination cannot hold, or when an
// already decoded value cannot be re-encoded. The destination is left at its zero value
// on a decoding error.
func (s jsonScanner[T]) Scan(source any) error {
	raw, err := columnJSON(source)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		var zero T
		*s.dest = zero
		return nil
	}
	if err := decodeColumn(raw, s.dest); err != nil {
		var zero T
		*s.dest = zero
		return fmt.Errorf("dbjson: decode JSON column: %w", err)
	}
	return nil
}

// columnJSON returns the JSON text a driver value holds.
//
// Takes source (any) which holds nil, raw JSON []byte, a JSON string, or an already
// decoded value.
//
// Returns []byte which is the JSON text, empty for a NULL.
// Returns error when an already decoded value cannot be re-encoded.
func columnJSON(source any) ([]byte, error) {
	switch value := source.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("dbjson: re-encode %T to JSON: %w", source, err)
		}
		return encoded, nil
	}
}

// decodeColumn decodes a non-empty JSON column into dest, retrying a document
// json.Unmarshal refuses in the forms the engine wrote, and then refuses a null element
// or object value dest cannot hold.
//
// Takes raw ([]byte) which is the column's JSON text.
// Takes dest (*T) which is the typed destination.
//
// Returns error when the document cannot be decoded into dest, is followed by more data,
// or holds a null dest cannot hold.
func decodeColumn[T any](raw []byte, dest *T) error {
	if err := json.Unmarshal(raw, dest); err != nil && !decodeEngineJSON(raw, dest) {
		return err
	}
	document, err := decodeDocument(raw)
	if err != nil {
		return err
	}
	return jsonNulls(document, reflect.TypeFor[T]())
}

// decodeDocument decodes a column's JSON text into generic values, keeping numbers as
// json.Number so a number wider than a float64, which a decimal destination reads, still
// decodes, and refusing anything after the one document the column holds.
//
// Takes raw ([]byte) which is the column's JSON text.
//
// Returns any which is the decoded document.
// Returns error when the text is not exactly one JSON document.
func decodeDocument(raw []byte) (any, error) {
	decoder := stdjson.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid character after the JSON document at offset %d", decoder.InputOffset())
	}
	return document, nil
}

// decodeEngineJSON decodes a document json.Unmarshal refused, after rewriting the values
// an engine writes in its own form, as its to-JSON function does for the elements of an
// array, in the forms Go reads, as engineJSONValue describes.
//
// Takes raw ([]byte) which is the column's JSON text.
// Takes dest (*T) which is the typed destination, reset before the retry.
//
// Returns bool which reports whether the rewritten document decoded into dest.
func decodeEngineJSON[T any](raw []byte, dest *T) bool {
	decoder := stdjson.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if decoder.Decode(&document) != nil {
		return false
	}
	rewritten, err := stdjson.Marshal(engineJSONValue(document, reflect.TypeFor[T]()))
	if err != nil {
		return false
	}
	var zero T
	*dest = zero
	return json.Unmarshal(rewritten, dest) == nil
}

// engineJSONValue recursively converts database JSON values to representations accepted
// by their destination Go types.
//
// Takes value (any) which is the value decoded with numbers kept as json.Number.
// Takes target (reflect.Type) which is the Go type that reads the value.
//
// Returns any which is the value in the form target reads, or value unchanged.
func engineJSONValue(value any, target reflect.Type) any {
	target = withoutPointers(target)
	switch typed := value.(type) {
	case []any:
		if isSequence(target) {
			return engineJSONElements(typed, target.Elem())
		}
	case map[string]any:
		if target.Kind() == reflect.Map {
			rewritten := make(map[string]any, len(typed))
			for key, element := range typed {
				rewritten[key] = engineJSONValue(element, target.Elem())
			}
			return rewritten
		}
	case stdjson.Number:
		if target.Kind() == reflect.String || unmarshalsText(target) {
			return string(typed)
		}
	case string:
		return engineJSONText(typed, target)
	}
	return value
}

// engineJSONElements rewrites each element of a JSON array as engineJSONValue does.
//
// Takes elements ([]any) which are the decoded array elements.
// Takes target (reflect.Type) which is the Go type of the elements.
//
// Returns []any which holds the rewritten elements.
func engineJSONElements(elements []any, target reflect.Type) []any {
	rewritten := make([]any, len(elements))
	for index, element := range elements {
		rewritten[index] = engineJSONValue(element, target)
	}
	return rewritten
}

// engineJSONText converts database date, timestamp, and hexadecimal byte strings for
// their destination Go types.
//
// Takes text (string) which is the JSON string's value.
// Takes target (reflect.Type) which is the Go type that reads it.
//
// Returns any which is the rewritten value, or text unchanged.
func engineJSONText(text string, target reflect.Type) any {
	switch {
	case target == reflect.TypeFor[time.Time]():
		for _, layout := range []string{"2006-01-02T15:04:05.999999999", time.DateOnly} {
			if moment, err := time.ParseInLocation(layout, text, time.UTC); err == nil {
				return moment.Format(time.RFC3339Nano)
			}
		}
	case target.Kind() == reflect.Slice && target.Elem().Kind() == reflect.Uint8 && !unmarshalsJSON(target):
		if digits, found := strings.CutPrefix(text, `\x`); found {
			if decoded, err := hex.DecodeString(digits); err == nil {
				return decoded
			}
		}
	}
	return text
}

// jsonNulls reports the first array element or map value whose Go type cannot hold NULL.
//
// Struct fields follow encoding/json rules.
//
// Takes document (any) which is the decoded JSON value.
// Takes target (reflect.Type) which is the Go type that reads it.
//
// Returns error which names the position of the first null target cannot hold.
func jsonNulls(document any, target reflect.Type) error {
	target = withoutPointers(target)
	switch typed := document.(type) {
	case []any:
		if isSequence(target) {
			return jsonElementNulls(typed, target.Elem())
		}
	case map[string]any:
		if target.Kind() == reflect.Map {
			return jsonValueNulls(typed, target.Elem())
		}
	}
	return nil
}

// jsonElementNulls reports the first null among a JSON array's elements, or inside them,
// that the Go type of the elements cannot hold.
//
// Takes elements ([]any) which are the decoded array elements.
// Takes element (reflect.Type) which is the Go type of the elements.
//
// Returns error which names the element's index.
func jsonElementNulls(elements []any, element reflect.Type) error {
	for index, value := range elements {
		if err := jsonNull(value, element); err != nil {
			return fmt.Errorf("element %d: %w", index, err)
		}
	}
	return nil
}

// jsonValueNulls reports the first null among a JSON object's values, or inside them, in
// key order, that the Go type of the values cannot hold.
//
// Takes values (map[string]any) which is the decoded JSON object.
// Takes element (reflect.Type) which is the Go type of the values.
//
// Returns error which names the value's key.
func jsonValueNulls(values map[string]any, element reflect.Type) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if err := jsonNull(values[key], element); err != nil {
			return fmt.Errorf("value of key %s: %w", key, err)
		}
	}
	return nil
}

// jsonNull reports a JSON null read into a Go type that cannot hold NULL (one that is not
// a pointer, an interface, a slice or a map and does not decode JSON itself), and the
// nulls inside a non-null value.
//
// Takes value (any) which is the decoded JSON value.
// Takes target (reflect.Type) which is the Go type that reads it.
//
// Returns error when value is null and target cannot hold it, or holds such a null.
func jsonNull(value any, target reflect.Type) error {
	if value != nil {
		return jsonNulls(value, target)
	}
	switch target.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
		return nil
	default:
	}
	if unmarshalsJSON(target) {
		return nil
	}
	return fmt.Errorf("converting NULL to %s is unsupported", target)
}

// withoutPointers returns the type a chain of pointers points to.
//
// Takes target (reflect.Type) which may be a pointer type.
//
// Returns reflect.Type which is target with every pointer removed.
func withoutPointers(target reflect.Type) reflect.Type {
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	return target
}

// isSequence reports whether a Go type reads a JSON array element by element.
//
// Takes target (reflect.Type) which is the Go type that reads the array.
//
// Returns bool which is true for a slice or an array.
func isSequence(target reflect.Type) bool {
	return target.Kind() == reflect.Slice || target.Kind() == reflect.Array
}

// unmarshalsText reports whether a Go type reads itself from text and not from JSON, so
// json.Unmarshal reads it only from a JSON string.
//
// Takes target (reflect.Type) which is the Go type to inspect.
//
// Returns bool which is true when a pointer to target is an encoding.TextUnmarshaler and
// not a json.Unmarshaler.
func unmarshalsText(target reflect.Type) bool {
	return reflect.PointerTo(target).Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) && !unmarshalsJSON(target)
}

// unmarshalsJSON reports whether a Go type reads itself from JSON.
//
// Takes target (reflect.Type) which is the Go type to inspect.
//
// Returns bool which is true when a pointer to target is a json.Unmarshaler.
func unmarshalsJSON(target reflect.Type) bool {
	return reflect.PointerTo(target).Implements(reflect.TypeFor[stdjson.Unmarshaler]())
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

// Value returns SQL NULL for nil values and otherwise returns the JSON as text.
//
// Returns driver.Value which is nil or the JSON text as a string.
// Returns error which is always nil.
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return string(j), nil
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
