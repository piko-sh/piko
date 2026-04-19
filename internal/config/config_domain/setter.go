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

package config_domain

import (
	"encoding"
	stdjson "encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// setField assigns a string value to a struct field, converting it to the
// appropriate type. It handles pointer dereferencing, unmarshalling, and
// various primitive types including strings, integers, floats, booleans,
// slices, and maps.
//
// When the field has an "overwrite" tag set to "false" and already contains a
// non-zero value, the field is left unchanged.
//
// Takes field (reflect.Value) which is the target field to set.
// Takes configValue (string) which is the string value to convert and assign.
// Takes tags (reflect.StructTag) which provides struct field metadata.
//
// Returns error when the value cannot be parsed or the field type is
// unsupported.
func setField(field reflect.Value, configValue string, tags reflect.StructTag) error {
	if overwriteTag := tags.Get("overwrite"); overwriteTag == "false" && !field.IsZero() {
		return nil
	}
	if handled, err := tryUnmarshal(field, configValue); handled {
		if err != nil {
			return fmt.Errorf("unmarshalling value %q: %w", configValue, err)
		}
		return nil
	}
	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(configValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setIntField(field, configValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUintField(field, configValue)
	case reflect.Float32, reflect.Float64:
		return setFloatField(field, configValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(configValue)
		if err != nil {
			return fmt.Errorf("parsing bool value %q: %w", configValue, err)
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		return setSliceField(field, configValue, tags)
	case reflect.Map:
		return setMapField(field, configValue, tags)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}
	return nil
}

// tryUnmarshal attempts to decode value into field using standard unmarshaler
// interfaces.
//
// Takes field (reflect.Value) which is the target field to unmarshal into.
// Takes configValue (string) which is the string value to decode.
//
// Returns bool which indicates whether an unmarshaler was found and used.
// Returns error when the unmarshaler fails to decode the value.
func tryUnmarshal(field reflect.Value, configValue string) (bool, error) {
	if !field.CanAddr() {
		return false, nil
	}
	ptr := field.Addr().Interface()
	if unmarshaler, ok := ptr.(encoding.TextUnmarshaler); ok {
		return true, unmarshaler.UnmarshalText([]byte(configValue))
	}
	if unmarshaler, ok := ptr.(stdjson.Unmarshaler); ok {
		return true, unmarshaler.UnmarshalJSON([]byte(configValue))
	}
	if unmarshaler, ok := ptr.(encoding.BinaryUnmarshaler); ok {
		return true, unmarshaler.UnmarshalBinary([]byte(configValue))
	}
	return false, nil
}

// isUnmarshaler checks whether a value implements a known unmarshaler
// interface.
//
// Takes value (reflect.Value) which is the value to check.
//
// Returns bool which is true if the value implements TextUnmarshaler,
// Unmarshaler, or BinaryUnmarshaler.
func isUnmarshaler(value reflect.Value) bool {
	if !value.CanAddr() {
		return false
	}
	ptr := value.Addr().Interface()
	_, isTextUnmarshaler := ptr.(encoding.TextUnmarshaler)
	_, isJSONUnmarshaler := ptr.(stdjson.Unmarshaler)
	_, isBinaryUnmarshaler := ptr.(encoding.BinaryUnmarshaler)
	return isTextUnmarshaler || isJSONUnmarshaler || isBinaryUnmarshaler
}

// setIntField sets an integer field from a string value.
//
// When the field type is time.Duration, parses the value as a duration string.
// Otherwise, parses the value as a base-10 integer.
//
// Takes field (reflect.Value) which is the integer field to set.
// Takes configValue (string) which is the string value to parse.
//
// Returns error when parsing fails.
func setIntField(field reflect.Value, configValue string) error {
	if field.Type() == reflect.TypeFor[time.Duration]() {
		duration, err := time.ParseDuration(configValue)
		if err != nil {
			return fmt.Errorf("parsing duration value %q: %w", configValue, err)
		}
		field.SetInt(int64(duration))
		return nil
	}
	intValue, err := strconv.ParseInt(configValue, 10, field.Type().Bits())
	if err != nil {
		return fmt.Errorf("parsing int value %q: %w", configValue, err)
	}
	field.SetInt(intValue)
	return nil
}

// setUintField parses a string as an unsigned integer and sets the field.
//
// Takes field (reflect.Value) which is the struct field to set.
// Takes configValue (string) which is the string value to parse.
//
// Returns error when the string cannot be parsed as an unsigned integer.
func setUintField(field reflect.Value, configValue string) error {
	uintValue, err := strconv.ParseUint(configValue, 10, field.Type().Bits())
	if err != nil {
		return fmt.Errorf("parsing uint value %q: %w", configValue, err)
	}
	field.SetUint(uintValue)
	return nil
}

// setFloatField parses a string and sets it on a reflect float field.
//
// Takes field (reflect.Value) which is the float field to set.
// Takes configValue (string) which is the string value to parse.
//
// Returns error when the string cannot be parsed as a float.
func setFloatField(field reflect.Value, configValue string) error {
	floatValue, err := strconv.ParseFloat(configValue, field.Type().Bits())
	if err != nil {
		return fmt.Errorf("parsing float value %q: %w", configValue, err)
	}
	field.SetFloat(floatValue)
	return nil
}

// setSliceField fills a slice field from a string value by splitting it with a
// delimiter.
//
// For byte slices, assigns the string directly as bytes. For other slice types,
// splits the value by the delimiter tag (or comma by default) and converts each
// part using setField.
//
// Takes field (reflect.Value) which is the slice field to fill.
// Takes configValue (string) which is the string to split and parse.
// Takes tags (reflect.StructTag) which may contain a "delimiter" tag.
//
// Returns error when any slice element cannot be converted to the target type.
func setSliceField(field reflect.Value, configValue string, tags reflect.StructTag) error {
	if field.Type().Elem().Kind() == reflect.Uint8 {
		field.Set(reflect.ValueOf([]byte(configValue)))
		return nil
	}
	delimiter := tags.Get("delimiter")
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	stringValues := strings.Split(configValue, delimiter)
	if len(stringValues) == 1 && stringValues[0] == "" {
		stringValues = []string{}
	}
	slice := reflect.MakeSlice(field.Type(), len(stringValues), len(stringValues))
	for i, v := range stringValues {
		if err := setField(slice.Index(i), strings.TrimSpace(v), ""); err != nil {
			return fmt.Errorf("slice element %d: %w", i, err)
		}
	}
	field.Set(slice)
	return nil
}

// setMapField parses a delimited string into a map field using reflection.
//
// When val is empty, sets the field to an empty map.
//
// Takes field (reflect.Value) which is the map field to fill.
// Takes configValue (string) which holds key-value pairs to parse.
// Takes tags (reflect.StructTag) which may set "delimiter" and "separator".
//
// Returns error when a pair is not valid or key/value conversion fails.
func setMapField(field reflect.Value, configValue string, tags reflect.StructTag) error {
	delimiter := tags.Get("delimiter")
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	separator := tags.Get("separator")
	if separator == "" {
		separator = defaultSeparator
	}
	if configValue == "" {
		field.Set(reflect.MakeMap(field.Type()))
		return nil
	}
	pairs := strings.Split(configValue, delimiter)
	theMap := reflect.MakeMapWithSize(field.Type(), len(pairs))
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, separator, 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid map item: %q", pair)
		}
		key := reflect.New(field.Type().Key()).Elem()
		if err := setField(key, strings.TrimSpace(kv[0]), ""); err != nil {
			return fmt.Errorf("map key %q: %w", kv[0], err)
		}
		value := reflect.New(field.Type().Elem()).Elem()
		if err := setField(value, strings.TrimSpace(kv[1]), ""); err != nil {
			return fmt.Errorf("map value for key %q: %w", kv[0], err)
		}
		theMap.SetMapIndex(key, value)
	}
	field.Set(theMap)
	return nil
}
