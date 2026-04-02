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

package pdfparse

import (
	"fmt"
	"strings"
)

// ObjectType identifies the kind of PDF object.
type ObjectType int

const (
	// ObjectNull represents the PDF null object.
	ObjectNull ObjectType = iota

	// ObjectBoolean represents a PDF boolean (true/false).
	ObjectBoolean

	// ObjectInteger represents a PDF integer number.
	ObjectInteger

	// ObjectReal represents a PDF real (floating point) number.
	ObjectReal

	// ObjectString represents a PDF literal string.
	ObjectString

	// ObjectHexString represents a PDF hexadecimal string.
	ObjectHexString

	// ObjectName represents a PDF name object.
	ObjectName

	// ObjectArray represents a PDF array.
	ObjectArray

	// ObjectDictionary represents a PDF dictionary.
	ObjectDictionary

	// ObjectStream represents a PDF stream (dictionary + data).
	ObjectStream

	// ObjectReference represents an indirect object reference.
	ObjectReference
)

// Object represents a parsed PDF object.
//
// The concrete value is stored in the Value field; its type is determined
// by Type. For ObjectStream, the decoded stream bytes are in StreamData
// and the stream dictionary is in Value (as Dict).
type Object struct {
	// Value holds the Go representation of this PDF object.
	Value any

	// StreamData holds the raw (possibly compressed) stream bytes for
	// ObjectStream objects. Nil for all other types.
	StreamData []byte

	// Type identifies which PDF object type this represents.
	Type ObjectType
}

// Ref represents an indirect object reference (e.g. "5 0 R").
type Ref struct {
	// Number is the object number.
	Number int

	// Generation is the generation number.
	Generation int
}

// Dict represents a PDF dictionary as an ordered list of key-value pairs.
//
// Keys are PDF name objects (without the leading /). Order is preserved
// to allow deterministic output when rewriting.
type Dict struct {
	// Pairs holds the key-value pairs in document order.
	Pairs []DictPair
}

// DictPair is a single key-value entry in a PDF dictionary.
type DictPair struct {
	// Key is the dictionary key (a PDF name without the leading /).
	Key string

	// Value is the dictionary value.
	Value Object
}

// Get returns the value for the given key, or a null Object if not found.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns Object which is the value associated with the key, or a null
// object if the key is absent.
func (d Dict) Get(key string) Object {
	for _, pair := range d.Pairs {
		if pair.Key == key {
			return pair.Value
		}
	}
	return Object{Type: ObjectNull}
}

// GetName returns the name value for the given key, or empty string if the
// key is absent or not a name.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns string which is the name value, or empty string if not found.
func (d Dict) GetName(key string) string {
	obj := d.Get(key)
	if s, ok := obj.Value.(string); ok && obj.Type == ObjectName {
		return s
	}
	return ""
}

// GetInt returns the integer value for the given key, or 0 if the key is
// absent or not an integer.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns int64 which is the integer value, or 0 if not found.
func (d Dict) GetInt(key string) int64 {
	obj := d.Get(key)
	if v, ok := obj.Value.(int64); ok && obj.Type == ObjectInteger {
		return v
	}
	return 0
}

// GetArray returns the array value for the given key, or nil if the key is
// absent or not an array.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns []Object which is the array value, or nil if not found.
func (d Dict) GetArray(key string) []Object {
	obj := d.Get(key)
	if arr, ok := obj.Value.([]Object); ok && obj.Type == ObjectArray {
		return arr
	}
	return nil
}

// GetDict returns the dictionary value for the given key, or an empty Dict
// if the key is absent or not a dictionary.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns Dict which is the nested dictionary, or an empty Dict if not
// found.
func (d Dict) GetDict(key string) Dict {
	obj := d.Get(key)
	if dict, ok := obj.Value.(Dict); ok && obj.Type == ObjectDictionary {
		return dict
	}
	return Dict{}
}

// GetRef returns the reference value for the given key, or a zero Ref if
// the key is absent or not a reference.
//
// Takes key (string) which specifies the dictionary key to look up.
//
// Returns Ref which is the indirect reference, or a zero Ref if not found.
func (d Dict) GetRef(key string) Ref {
	obj := d.Get(key)
	if ref, ok := obj.Value.(Ref); ok && obj.Type == ObjectReference {
		return ref
	}
	return Ref{}
}

// Has returns true if the dictionary contains the given key.
//
// Takes key (string) which specifies the dictionary key to check.
//
// Returns bool which indicates whether the key exists.
func (d Dict) Has(key string) bool {
	for _, pair := range d.Pairs {
		if pair.Key == key {
			return true
		}
	}
	return false
}

// Set adds or replaces a key-value pair in the dictionary.
//
// Takes key (string) which specifies the dictionary key to set.
// Takes value (Object) which is the value to associate with the key.
func (d *Dict) Set(key string, value Object) {
	for i, pair := range d.Pairs {
		if pair.Key == key {
			d.Pairs[i].Value = value
			return
		}
	}
	d.Pairs = append(d.Pairs, DictPair{Key: key, Value: value})
}

// Remove deletes a key from the dictionary.
//
// Takes key (string) which specifies the dictionary key to remove.
//
// Returns bool which indicates whether the key existed and was removed.
func (d *Dict) Remove(key string) bool {
	for i, pair := range d.Pairs {
		if pair.Key == key {
			d.Pairs = append(d.Pairs[:i], d.Pairs[i+1:]...)
			return true
		}
	}
	return false
}

// Keys returns all dictionary keys in order.
//
// Returns []string which holds the keys in document order.
func (d Dict) Keys() []string {
	keys := make([]string, len(d.Pairs))
	for i, pair := range d.Pairs {
		keys[i] = pair.Key
	}
	return keys
}

// Null returns a null PDF object.
//
// Returns Object which represents the PDF null value.
func Null() Object { return Object{Type: ObjectNull} }

// Bool returns a boolean PDF object.
//
// Takes v (bool) which specifies the boolean value.
//
// Returns Object which represents the PDF boolean.
func Bool(v bool) Object { return Object{Type: ObjectBoolean, Value: v} }

// Int returns an integer PDF object.
//
// Takes v (int64) which specifies the integer value.
//
// Returns Object which represents the PDF integer.
func Int(v int64) Object { return Object{Type: ObjectInteger, Value: v} }

// Real returns a real number PDF object.
//
// Takes v (float64) which specifies the floating point value.
//
// Returns Object which represents the PDF real number.
func Real(v float64) Object { return Object{Type: ObjectReal, Value: v} }

// Str returns a literal string PDF object.
//
// Takes v (string) which specifies the string content.
//
// Returns Object which represents the PDF literal string.
func Str(v string) Object { return Object{Type: ObjectString, Value: v} }

// HexStr returns a hex string PDF object.
//
// Takes v (string) which specifies the decoded hex string content.
//
// Returns Object which represents the PDF hexadecimal string.
func HexStr(v string) Object { return Object{Type: ObjectHexString, Value: v} }

// Name returns a name PDF object (without leading /).
//
// Takes v (string) which specifies the name value without the leading slash.
//
// Returns Object which represents the PDF name.
func Name(v string) Object { return Object{Type: ObjectName, Value: v} }

// Arr returns an array PDF object.
//
// Takes items ([]Object) which specifies the array elements.
//
// Returns Object which represents the PDF array.
func Arr(items ...Object) Object { return Object{Type: ObjectArray, Value: items} }

// DictObj returns a dictionary PDF object.
//
// Takes d (Dict) which specifies the dictionary content.
//
// Returns Object which represents the PDF dictionary.
func DictObj(d Dict) Object { return Object{Type: ObjectDictionary, Value: d} }

// RefObj returns an indirect reference PDF object.
//
// Takes number (int) which specifies the object number.
// Takes generation (int) which specifies the generation number.
//
// Returns Object which represents the PDF indirect reference.
func RefObj(number, generation int) Object {
	return Object{Type: ObjectReference, Value: Ref{Number: number, Generation: generation}}
}

// StreamObj returns a stream PDF object with the given dictionary and data.
//
// Takes d (Dict) which specifies the stream dictionary.
// Takes data ([]byte) which holds the raw stream bytes.
//
// Returns Object which represents the PDF stream.
func StreamObj(d Dict, data []byte) Object {
	return Object{Type: ObjectStream, Value: d, StreamData: data}
}

// String returns a human-readable representation of the object for
// debugging.
//
// Returns string which is the formatted representation.
func (o Object) String() string {
	switch o.Type {
	case ObjectNull:
		return "null"
	case ObjectBoolean:
		return stringifyBool(o)
	case ObjectInteger:
		return stringifyInt(o)
	case ObjectReal:
		return stringifyReal(o)
	case ObjectString:
		return stringifyLiteral(o)
	case ObjectHexString:
		return stringifyHex(o)
	case ObjectName:
		return stringifyName(o)
	case ObjectArray:
		return stringifyArray(o)
	case ObjectDictionary:
		return stringifyDict(o)
	case ObjectStream:
		return fmt.Sprintf("stream(%d bytes)", len(o.StreamData))
	case ObjectReference:
		return stringifyRef(o)
	default:
		return "unknown"
	}
}

// stringifyBool formats a boolean Object as "true" or "false".
//
// Takes o (Object) which holds the boolean value.
//
// Returns string which is the formatted boolean.
func stringifyBool(o Object) string {
	if v, ok := o.Value.(bool); ok && v {
		return "true"
	}
	return "false"
}

// stringifyInt formats an integer Object as a decimal string.
//
// Takes o (Object) which holds the integer value.
//
// Returns string which is the formatted integer.
func stringifyInt(o Object) string {
	if v, ok := o.Value.(int64); ok {
		return fmt.Sprintf("%d", v)
	}
	return "0"
}

// stringifyReal formats a real Object as a decimal string.
//
// Takes o (Object) which holds the float64 value.
//
// Returns string which is the formatted real number.
func stringifyReal(o Object) string {
	if v, ok := o.Value.(float64); ok {
		return fmt.Sprintf("%g", v)
	}
	return "0"
}

// stringifyLiteral formats a literal string Object in parentheses.
//
// Takes o (Object) which holds the string value.
//
// Returns string which is the parenthesised literal string.
func stringifyLiteral(o Object) string {
	if s, ok := o.Value.(string); ok {
		return fmt.Sprintf("(%s)", s)
	}
	return "()"
}

// stringifyHex formats a hex string Object in angle brackets.
//
// Takes o (Object) which holds the hex string value.
//
// Returns string which is the angle-bracketed hex string.
func stringifyHex(o Object) string {
	if s, ok := o.Value.(string); ok {
		return fmt.Sprintf("<%s>", s)
	}
	return "<>"
}

// stringifyName formats a name Object with a leading slash.
//
// Takes o (Object) which holds the name string.
//
// Returns string which is the slash-prefixed name.
func stringifyName(o Object) string {
	if s, ok := o.Value.(string); ok {
		return fmt.Sprintf("/%s", s)
	}
	return "/"
}

// stringifyArray formats an array Object as a bracketed list.
//
// Takes o (Object) which holds the array items.
//
// Returns string which is the bracketed array representation.
func stringifyArray(o Object) string {
	items, ok := o.Value.([]Object)
	if !ok {
		return "[]"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, " "))
}

// stringifyDict formats a dictionary Object as double angle brackets.
//
// Takes o (Object) which holds the Dict value.
//
// Returns string which is the angle-bracketed dictionary representation.
func stringifyDict(o Object) string {
	d, ok := o.Value.(Dict)
	if !ok {
		return "<<>>"
	}
	parts := make([]string, len(d.Pairs))
	for i, pair := range d.Pairs {
		parts[i] = fmt.Sprintf("/%s %s", pair.Key, pair.Value.String())
	}
	return fmt.Sprintf("<<%s>>", strings.Join(parts, " "))
}

// stringifyRef formats a reference Object as "N G R".
//
// Takes o (Object) which holds the Ref value.
//
// Returns string which is the formatted indirect reference.
func stringifyRef(o Object) string {
	if ref, ok := o.Value.(Ref); ok {
		return fmt.Sprintf("%d %d R", ref.Number, ref.Generation)
	}
	return "0 0 R"
}

// IsNull returns true if this is a null object.
//
// Returns bool which indicates whether the object type is ObjectNull.
func (o Object) IsNull() bool { return o.Type == ObjectNull }
