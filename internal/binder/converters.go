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
	"errors"
	"fmt"
	"image/color"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"piko.sh/piko/wdk/maths"
)

const (
	// colourLenRGB is the length of a short RGB hex colour code without the #
	// prefix.
	colourLenRGB = 3

	// colourLenRRGGBB is the length of a six-digit hex colour code (RRGGBB).
	colourLenRRGGBB = 6

	// colourLenRRGGBBAA is the length of an RRGGBBAA hex colour string.
	colourLenRRGGBBAA = 8

	// colourScaleShortForm is the multiplier to convert 4-bit colour values to
	// 8-bit.
	colourScaleShortForm = 17
)

// ConverterFunc defines a function that converts a string into a reflect.Value.
type ConverterFunc func(value string) (reflect.Value, error)

// primitiveConverter is a function type that converts a string to a value of
// a given primitive type.
type primitiveConverter func(value string) (reflect.Value, error)

var (
	// primitiveConverters acts as a dispatch table for Go's built-in types.
	primitiveConverters = map[reflect.Kind]primitiveConverter{
		reflect.String:    convertToString,
		reflect.Bool:      convertToBool,
		reflect.Int:       convertToInt,
		reflect.Int8:      convertToInt8,
		reflect.Int16:     convertToInt16,
		reflect.Int32:     convertToInt32,
		reflect.Int64:     convertToInt64,
		reflect.Uint:      convertToUint,
		reflect.Uint8:     convertToUint8,
		reflect.Uint16:    convertToUint16,
		reflect.Uint32:    convertToUint32,
		reflect.Uint64:    convertToUint64,
		reflect.Float32:   convertToFloat32,
		reflect.Float64:   convertToFloat64,
		reflect.Interface: convertToInterface,
	}

	// timeLayouts defines the supported formats for string-to-time conversion, in
	// order of preference.
	timeLayouts = []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006",
		time.RFC1123,
	}

	// wellKnownTypeConverters maps well-known types to their converters.
	wellKnownTypeConverters = map[reflect.Type]primitiveConverter{
		reflect.TypeFor[time.Time]():     convertTimeType,
		reflect.TypeFor[time.Duration](): convertDurationType,
		reflect.TypeFor[url.URL]():       convertURLType,
		reflect.TypeFor[mail.Address]():  convertMailAddressType,
		reflect.TypeFor[color.Color]():   convertColourType,
		reflect.TypeFor[maths.Decimal](): convertDecimalType,
		reflect.TypeFor[maths.Money]():   convertMoneyType,
	}
)

// convertAndSet is the primary dispatcher for type conversion. It validates
// the target field and delegates to the appropriate converter for custom or
// primitive types, using pre-cached metadata from the fieldInfo struct.
//
// Takes field (reflect.Value) which is the target struct field to set.
// Takes value (string) which is the raw string value to convert.
// Takes fullPath (string) which is the full path for error reporting.
// Takes fi (*fieldInfo) which provides cached metadata about the field.
//
// Returns error when the field is invalid, not settable, or conversion fails.
func (b *ASTBinder) convertAndSet(field reflect.Value, value string, fullPath string, fi *fieldInfo) error {
	if !field.IsValid() {
		return errSetField{err: errors.New("field is not valid (e.g., trying to set a field on a nil embedded struct pointer)"), path: fullPath, field: fi.Path, fieldType: fi.Type.String()}
	}
	if !field.CanSet() {
		return errSetField{err: errors.New("field cannot be set (it may be unexported)"), path: fullPath, field: fi.Path, fieldType: fi.Type.String()}
	}

	convertedVal, err := b.convertToType(value, fi)
	if err != nil {
		return errSetField{path: fullPath, field: fi.Path, fieldType: fi.Type.String(), err: err}
	}

	if fi.Type.Kind() == reflect.Pointer {
		ptr := reflect.New(fi.Type.Elem())

		ptr.Elem().Set(convertedVal)

		field.Set(ptr)
	} else {
		if !convertedVal.Type().AssignableTo(fi.Type) {
			convertedVal = convertedVal.Convert(fi.Type)
		}
		field.Set(convertedVal)
	}

	return nil
}

// convertToType selects a converter based on precedence: user-registered
// converters, framework converters for well-known types, TextUnmarshaler, then
// primitive kind converters.
//
// Takes value (string) which is the raw string to convert.
// Takes fi (*fieldInfo) which provides the target type and unmarshaler.
//
// Returns reflect.Value which is the converted value.
// Returns error when no suitable converter exists for the target type.
func (b *ASTBinder) convertToType(value string, fi *fieldInfo) (reflect.Value, error) {
	targetType := fi.Type
	if targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}

	if converter := b.getUserConverter(targetType); converter != nil {
		return converter(value)
	}

	if converter, ok := wellKnownTypeConverters[targetType]; ok {
		return converter(value)
	}

	if fi.unmarshaler != nil {
		return unmarshalText(targetType, value)
	}

	if primitiveConv, ok := primitiveConverters[targetType.Kind()]; ok {
		return primitiveConv(value)
	}

	return reflect.Value{}, fmt.Errorf("unsupported type: %s", targetType.String())
}

// getUserConverter retrieves a user-registered converter for the given type.
//
// Takes targetType (reflect.Type) which specifies the type to find a
// converter for.
//
// Returns ConverterFunc which is the registered converter, or nil if none
// exists.
//
// Uses sync.Map for lock-free reads, suited to read-heavy workloads. Skips
// the lookup if no converters are registered.
func (b *ASTBinder) getUserConverter(targetType reflect.Type) ConverterFunc {
	if !b.hasConverters.Load() {
		return nil
	}
	if converter, ok := b.converters.Load(targetType); ok {
		if converter, ok := converter.(ConverterFunc); ok {
			return converter
		}
	}
	return nil
}

// convertTimeType parses a string value into a time.Time reflect value.
//
// Takes value (string) which is the time string to parse.
//
// Returns reflect.Value which holds the parsed time.Time value.
// Returns error when the time string cannot be parsed.
func convertTimeType(value string) (reflect.Value, error) {
	t, err := parseTime(value)
	return reflect.ValueOf(t), err
}

// convertDurationType parses a duration string and returns it as a reflect
// value.
//
// Takes value (string) which is the duration string to parse.
//
// Returns reflect.Value which holds the parsed duration.
// Returns error when the duration string is invalid.
func convertDurationType(value string) (reflect.Value, error) {
	d, err := parseDuration(value)
	return reflect.ValueOf(d), err
}

// convertURLType parses a string as a URL and returns it as a reflect.Value.
//
// Takes value (string) which is the URL string to parse.
//
// Returns reflect.Value which holds the parsed URL.
// Returns error when the URL string cannot be parsed.
func convertURLType(value string) (reflect.Value, error) {
	u, err := parseURL(value)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(u).Elem(), nil
}

// convertMailAddressType converts a string to a mail address value.
//
// Takes value (string) which is the email address to parse.
//
// Returns reflect.Value which holds the parsed mail address.
// Returns error when the value is not a valid email address.
func convertMailAddressType(value string) (reflect.Value, error) {
	addr, err := parseMailAddress(value)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(addr).Elem(), nil
}

// convertColourType parses a colour string and returns it as a reflect.Value.
//
// Takes value (string) which is the colour string to parse.
//
// Returns reflect.Value which contains the parsed colour.
// Returns error when the colour string cannot be parsed.
func convertColourType(value string) (reflect.Value, error) {
	c, err := parseColour(value)
	return reflect.ValueOf(c), err
}

// convertDecimalType parses a string into a Decimal type.
//
// Takes value (string) which contains the decimal number to parse.
//
// Returns reflect.Value which wraps the parsed Decimal.
// Returns error when parsing fails.
func convertDecimalType(value string) (reflect.Value, error) {
	d := maths.NewDecimalFromString(value)
	return reflect.ValueOf(d), nil
}

// convertMoneyType converts a string value to a Money type.
//
// Takes value (string) which is the amount to convert.
//
// Returns reflect.Value which holds the Money instance.
// Returns error which is always nil.
func convertMoneyType(value string) (reflect.Value, error) {
	m := maths.NewMoneyFromString(value, "GBP")
	return reflect.ValueOf(m), nil
}

// unmarshalText converts a string to a value using the TextUnmarshaler
// interface.
//
// Takes targetType (reflect.Type) which specifies the type to create.
// Takes value (string) which contains the text to convert.
//
// Returns reflect.Value which holds the converted value.
// Returns error when the target type does not implement TextUnmarshaler or
// when the conversion fails.
func unmarshalText(targetType reflect.Type, value string) (reflect.Value, error) {
	newInstance := reflect.New(targetType)
	u, ok := reflect.TypeAssert[encoding.TextUnmarshaler](newInstance)
	if !ok {
		return reflect.Value{}, fmt.Errorf("type %s does not implement encoding.TextUnmarshaler", targetType.String())
	}

	if err := u.UnmarshalText([]byte(value)); err != nil {
		return reflect.Value{}, fmt.Errorf("failed to unmarshal text: %w", err)
	}
	return newInstance.Elem(), nil
}

// parseTime parses a time string using the supported time layouts.
//
// When value is empty, returns a zero time without error.
//
// Takes value (string) which is the time string to parse.
//
// Returns time.Time which is the parsed time value.
// Returns error when the value does not match any supported layout.
func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse '%s' as a valid time", value)
}

// parseDuration parses a string into a time.Duration.
//
// When the value is an empty string, returns zero duration without error.
//
// Takes value (string) which is the duration string to parse.
//
// Returns time.Duration which is the parsed duration.
// Returns error when the value is not a valid duration format.
func parseDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("could not parse '%s' as a valid duration: %w", value, err)
	}
	return d, nil
}

// parseURL parses a string as a URL.
//
// When the value is an empty string, returns an empty URL struct without
// error.
//
// Takes value (string) which is the URL string to parse.
//
// Returns *url.URL which is the parsed URL.
// Returns error when the value is not a valid URL.
func parseURL(value string) (*url.URL, error) {
	if value == "" {
		return &url.URL{}, nil
	}
	u, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a valid URL: %w", value, err)
	}
	return u, nil
}

// parseMailAddress parses a string into an email address.
//
// When value is empty, returns an empty Address without error.
//
// Takes value (string) which is the email address to parse.
//
// Returns *mail.Address which contains the parsed email address.
// Returns error when the value is not a valid email address.
func parseMailAddress(value string) (*mail.Address, error) {
	if value == "" {
		return &mail.Address{}, nil
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a valid email address: %w", value, err)
	}
	return addr, nil
}

// parseColour parses a hex colour string into a color.Color value.
//
// Takes value (string) which is the colour in #RGB, #RRGGBB, or #RRGGBBAA
// format. An empty string returns a zero-value RGBA.
//
// Returns color.Color which is the parsed colour as an RGBA value.
// Returns error when the format is not valid or the hex values cannot be
// parsed.
func parseColour(value string) (color.Color, error) {
	if value == "" {
		return color.RGBA{}, nil
	}

	if len(value) > 0 && value[0] == '#' {
		value = value[1:]
	}

	var r, g, b, a uint8 = 0, 0, 0, 255

	switch len(value) {
	case colourLenRGB:
		if _, err := fmt.Sscanf(value, "%1x%1x%1x", &r, &g, &b); err != nil {
			return nil, fmt.Errorf("could not parse '%s' as a valid colour: %w", value, err)
		}
		r, g, b = r*colourScaleShortForm, g*colourScaleShortForm, b*colourScaleShortForm
	case colourLenRRGGBB:
		if _, err := fmt.Sscanf(value, "%2x%2x%2x", &r, &g, &b); err != nil {
			return nil, fmt.Errorf("could not parse '%s' as a valid colour: %w", value, err)
		}
	case colourLenRRGGBBAA:
		if _, err := fmt.Sscanf(value, "%2x%2x%2x%2x", &r, &g, &b, &a); err != nil {
			return nil, fmt.Errorf("could not parse '%s' as a valid colour: %w", value, err)
		}
	default:
		return nil, fmt.Errorf("could not parse '%s' as a valid colour: invalid format (expected #RGB, #RRGGBB, or #RRGGBBAA)", value)
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// convertToString converts a string value to its reflect.Value form.
//
// Takes value (string) which is the string to convert.
//
// Returns reflect.Value which holds the string as a reflected value.
// Returns error which is always nil for string conversion.
func convertToString(value string) (reflect.Value, error) {
	return reflect.ValueOf(value), nil
}

// convertToInterface stores a string value as an interface{}/any type.
//
// Use it for binding to map[string]any and other dynamic types. The value is
// stored as a string. The caller must handle any further type conversion.
//
// Takes value (string) which is the string to store.
//
// Returns reflect.Value which holds the string wrapped as an any value.
// Returns error which is always nil for interface conversion.
func convertToInterface(value string) (reflect.Value, error) {
	var v any = value
	return reflect.ValueOf(v), nil
}

// convertToBool parses a string into a boolean reflect.Value.
//
// When value is empty, returns false without error. When value is "on",
// returns true to support HTML checkbox values.
//
// Takes value (string) which is the string to parse as a boolean.
//
// Returns reflect.Value which contains the parsed boolean.
// Returns error when the value cannot be parsed as a boolean.
func convertToBool(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(false), nil
	}
	if value == "on" {
		return reflect.ValueOf(true), nil
	}
	b, err := strconv.ParseBool(value)
	return reflect.ValueOf(b), err
}

// convertToInt parses a string as a base-10 integer and returns it as a
// reflect.Value.
//
// Takes value (string) which is the numeric string to parse.
//
// Returns reflect.Value which holds the parsed integer.
// Returns error when the string cannot be parsed as an integer.
func convertToInt(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(int(0)), nil
	}
	i, err := strconv.ParseInt(value, 10, 64)
	return reflect.ValueOf(int(i)), err
}

// convertToInt8 parses a string as an 8-bit signed integer.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which holds the parsed int8, or zero if empty.
// Returns error when the string is not a valid 8-bit integer.
func convertToInt8(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(int8(0)), nil
	}
	i, err := strconv.ParseInt(value, 10, 8)
	return reflect.ValueOf(int8(i)), err
}

// convertToInt16 parses a string as a 16-bit signed integer.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which holds the parsed int16, or zero if empty.
// Returns error when the string is not a valid 16-bit integer.
func convertToInt16(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(int16(0)), nil
	}
	i, err := strconv.ParseInt(value, 10, 16)
	return reflect.ValueOf(int16(i)), err
}

// convertToInt32 parses a string into a 32-bit integer reflect value.
//
// When the value is empty, returns a zero value without error.
//
// Takes value (string) which is the number string to parse.
//
// Returns reflect.Value which holds the parsed int32.
// Returns error when the string is not a valid number.
func convertToInt32(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(int32(0)), nil
	}
	i, err := strconv.ParseInt(value, 10, 32)
	return reflect.ValueOf(int32(i)), err
}

// convertToInt64 parses a string and returns it as a reflected int64 value.
//
// Takes value (string) which is the number string to parse.
//
// Returns reflect.Value which holds the parsed int64, or zero if the input is
// empty.
// Returns error when the string is not a valid integer.
func convertToInt64(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(int64(0)), nil
	}
	i, err := strconv.ParseInt(value, 10, 64)
	return reflect.ValueOf(i), err
}

// convertToUint parses a string into an unsigned integer value.
//
// When the value is empty, returns zero without error.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which holds the parsed uint.
// Returns error when the string is not a valid unsigned integer.
func convertToUint(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(uint(0)), nil
	}
	u, err := strconv.ParseUint(value, 10, 64)
	return reflect.ValueOf(uint(u)), err
}

// convertToUint8 parses a string into an uint8 value wrapped in a
// reflect.Value.
//
// When the value is empty, returns zero without error.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which contains the parsed uint8.
// Returns error when the string is not a valid decimal or exceeds uint8 range.
func convertToUint8(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(uint8(0)), nil
	}
	u, err := strconv.ParseUint(value, 10, 8)
	return reflect.ValueOf(uint8(u)), err
}

// convertToUint16 parses a string as a base-10 unsigned 16-bit integer.
//
// When the value is empty, returns zero without error.
//
// Takes value (string) which is the text to parse.
//
// Returns reflect.Value which holds the parsed uint16.
// Returns error when the string is not a valid unsigned integer.
func convertToUint16(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(uint16(0)), nil
	}
	u, err := strconv.ParseUint(value, 10, 16)
	return reflect.ValueOf(uint16(u)), err
}

// convertToUint32 parses a string into a uint32 value wrapped in reflect.Value.
//
// When the value is empty, returns zero without error.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which contains the parsed uint32.
// Returns error when the string is not a valid decimal number or is out of
// range for uint32.
func convertToUint32(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(uint32(0)), nil
	}
	u, err := strconv.ParseUint(value, 10, 32)
	return reflect.ValueOf(uint32(u)), err
}

// convertToUint64 parses a string as an unsigned 64-bit integer.
//
// When value is empty, returns zero without error.
//
// Takes value (string) which is the decimal number to parse.
//
// Returns reflect.Value which holds the parsed uint64.
// Returns error when the string is not a valid unsigned 64-bit integer.
func convertToUint64(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(uint64(0)), nil
	}
	u, err := strconv.ParseUint(value, 10, 64)
	return reflect.ValueOf(u), err
}

// convertToFloat32 parses a string into a float32 value.
//
// When the value is an empty string, returns zero without error.
//
// Takes value (string) which is the string to parse.
//
// Returns reflect.Value which holds the parsed float32.
// Returns error when the string is not a valid number.
func convertToFloat32(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(float32(0)), nil
	}
	f, err := strconv.ParseFloat(value, 32)
	return reflect.ValueOf(float32(f)), err
}

// convertToFloat64 parses a string into a float64 reflect value.
//
// When the value is an empty string, returns zero without error.
//
// Takes value (string) which is the string to parse.
//
// Returns reflect.Value which contains the parsed float64.
// Returns error when the string is not a valid number.
func convertToFloat64(value string) (reflect.Value, error) {
	if value == "" {
		return reflect.ValueOf(float64(0)), nil
	}
	f, err := strconv.ParseFloat(value, 64)
	return reflect.ValueOf(f), err
}
