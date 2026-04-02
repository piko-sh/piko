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

package generator_helpers

import (
	"encoding/base64"
	"reflect"
	"strconv"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// decimalBase is the numeric base for decimal string formatting.
	decimalBase = 10

	// float32BitSize is the bit size used when formatting float32 values.
	float32BitSize = 32

	// float64BitSize is the number of bits used when formatting float64 values.
	float64BitSize = 64

	// controlCharMax is the upper bound for JSON control characters that need
	// \uXXXX escaping (bytes 0x00 to 0x1F).
	controlCharMax = 0x20

	// jsonNull is the JSON null literal.
	jsonNull = "null"

	// jsonEmptyArray is the JSON empty array literal.
	jsonEmptyArray = "[]"

	// jsonPayloadPrefix is the JSON object opener with function field.
	jsonPayloadPrefix = `{"f":"`
	// jsonArgsPrefix is the JSON separator before the arguments array with opening
	// bracket.
	jsonArgsPrefix = `","a":[`

	// jsonArgsPrefixNoOpen is the JSON separator before arguments without the array
	// bracket.
	jsonArgsPrefixNoOpen = `","a":`
	// jsonPayloadSuffix is the JSON closer for the payload object.
	jsonPayloadSuffix = `]}`

	// jsonPayloadSuffixZeroArgs is the JSON closer for payload with empty arguments.
	jsonPayloadSuffixZeroArgs = `","a":[]}`
	// hexDigits is a lookup table for hex encoding.
	hexDigits = "0123456789abcdef"
)

// EncodeActionPayloadBytes encodes an ActionPayload to a base64 URL-safe byte
// slice using zero-allocation encoding after pool warmup.
//
// This implementation works directly through the buffer pointer to prevent the
// slice header from escaping to the heap, achieving zero allocations when the
// buffer pool is warm.
//
// The returned buffer must be tracked by the caller for later release. Pass the
// result to DirectWriter.AppendPooledBytes() which will track and release it.
//
// Takes payload (templater_dto.ActionPayload) which is the action payload to
// encode.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes(payload templater_dto.ActionPayload) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON(bufferPointer, &payload)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes0 encodes an ActionPayload with zero arguments.
// This avoids the slice allocation that occurs with the variadic form.
//
// Takes function (string) which is the function name to encode.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes0(function string) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON0(bufferPointer, function)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes1 encodes an ActionPayload with one argument.
// This avoids the slice allocation that occurs with the variadic form.
//
// Takes function (string) which is the function name to encode.
// Takes argument0 (templater_dto.ActionArgument) which is the first argument.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes1(function string, argument0 templater_dto.ActionArgument) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON1(bufferPointer, function, &argument0)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes2 encodes an ActionPayload with two arguments.
// This avoids the slice allocation that occurs with the variadic form.
//
// Takes function (string) which is the function name to encode.
// Takes argument0 (templater_dto.ActionArgument) which is the first argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second argument.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes2(function string, argument0, argument1 templater_dto.ActionArgument) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON2(bufferPointer, function, &argument0, &argument1)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes3 encodes an ActionPayload with three arguments.
// This avoids the slice allocation that occurs with the variadic form.
//
// Takes function (string) which is the function name to encode.
// Takes argument0 (templater_dto.ActionArgument) which is the first argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second argument.
// Takes argument2 (templater_dto.ActionArgument) which is the third argument.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes3(function string, argument0, argument1, argument2 templater_dto.ActionArgument) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON3(bufferPointer, function, &argument0, &argument1, &argument2)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes4 encodes an ActionPayload with four arguments.
// This avoids the slice allocation that occurs with the variadic form.
//
// Takes function (string) which is the function name to encode.
// Takes argument0 (templater_dto.ActionArgument) which is the first argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second argument.
// Takes argument2 (templater_dto.ActionArgument) which is the third argument.
// Takes argument3 (templater_dto.ActionArgument) which is the fourth argument.
//
// Returns *[]byte which is a pooled buffer containing the base64-encoded
// payload, or nil on error.
func EncodeActionPayloadBytes4(function string, argument0, argument1, argument2, argument3 templater_dto.ActionArgument) *[]byte {
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON4(bufferPointer, function, &argument0, &argument1, &argument2, &argument3)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeStaticActionPayload encodes an ActionPayload to a base64 URL-safe
// string at code generation time. Unlike EncodeActionPayloadBytes*, this
// returns a plain string suitable for embedding directly in generated code as a
// static attribute value.
//
// This is used by the static emitter to pre-encode action payloads for events
// that are fully static (no dynamic arguments). The payload is computed once at
// code generation time rather than at each request.
//
// Takes payload (templater_dto.ActionPayload) which is the action payload to
// encode.
//
// Returns string which is the base64-encoded payload.
func EncodeStaticActionPayload(payload templater_dto.ActionPayload) string {
	var buffer []byte
	encodePayloadJSON(&buffer, &payload)
	encoded := make([]byte, base64.RawURLEncoding.EncodedLen(len(buffer)))
	base64.RawURLEncoding.Encode(encoded, buffer)
	return string(encoded)
}

// EncodeActionPayloadBytesArena is the arena-aware version of
// EncodeActionPayloadBytes.
//
// Takes arena (*ast_domain.RenderArena) which provides the byte buffer.
// Takes payload (templater_dto.ActionPayload) which contains the action data.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytesArena(arena *ast_domain.RenderArena, payload templater_dto.ActionPayload) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON(bufferPointer, &payload)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes0Arena is the arena-aware version of
// EncodeActionPayloadBytes0.
//
// Takes arena (*ast_domain.RenderArena) which provides the memory arena for
// buffer allocation.
// Takes function (string) which specifies the function name to encode.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytes0Arena(arena *ast_domain.RenderArena, function string) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON0(bufferPointer, function)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes1Arena is the arena-aware version of
// EncodeActionPayloadBytes1.
//
// Takes arena (*ast_domain.RenderArena) which provides the byte buffer pool.
// Takes function (string) which specifies the action function name.
// Takes argument0 (templater_dto.ActionArgument) which contains the
// action argument.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytes1Arena(arena *ast_domain.RenderArena, function string, argument0 templater_dto.ActionArgument) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON1(bufferPointer, function, &argument0)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes2Arena is the arena-aware version of
// EncodeActionPayloadBytes2.
//
// Takes arena (*ast_domain.RenderArena) which provides the byte buffer pool.
// Takes function (string) which specifies the action function name.
// Takes argument0 (templater_dto.ActionArgument) which is the first
// action argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second
// action argument.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytes2Arena(arena *ast_domain.RenderArena, function string, argument0, argument1 templater_dto.ActionArgument) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON2(bufferPointer, function, &argument0, &argument1)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes3Arena is the arena-aware version of
// EncodeActionPayloadBytes3.
//
// Takes arena (*ast_domain.RenderArena) which provides buffer allocation.
// Takes function (string) which specifies the action function name.
// Takes argument0 (templater_dto.ActionArgument) which is the first
// action argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second
// action argument.
// Takes argument2 (templater_dto.ActionArgument) which is the third
// action argument.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytes3Arena(arena *ast_domain.RenderArena, function string, argument0, argument1, argument2 templater_dto.ActionArgument) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON3(bufferPointer, function, &argument0, &argument1, &argument2)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// EncodeActionPayloadBytes4Arena is the arena-aware version of
// EncodeActionPayloadBytes4.
//
// Takes arena (*ast_domain.RenderArena) which provides the byte buffer pool.
// Takes function (string) which specifies the action function name.
// Takes argument0 (templater_dto.ActionArgument) which is the first
// action argument.
// Takes argument1 (templater_dto.ActionArgument) which is the second
// action argument.
// Takes argument2 (templater_dto.ActionArgument) which is the third
// action argument.
// Takes argument3 (templater_dto.ActionArgument) which is the fourth
// action argument.
//
// Returns *[]byte which contains the base64-encoded JSON payload.
func EncodeActionPayloadBytes4Arena(arena *ast_domain.RenderArena, function string, argument0, argument1, argument2, argument3 templater_dto.ActionArgument) *[]byte {
	bufferPointer := arena.GetByteBuf()
	*bufferPointer = (*bufferPointer)[:0]

	encodePayloadJSON4(bufferPointer, function, &argument0, &argument1, &argument2, &argument3)
	encodeBase64InPlace(bufferPointer)

	return bufferPointer
}

// encodePayloadJSON builds the JSON representation directly into the buffer.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append JSON data to.
// Takes payload (*templater_dto.ActionPayload) which contains the function
// name and arguments to encode.
func encodePayloadJSON(bufferPointer *[]byte, payload *templater_dto.ActionPayload) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, payload.Function)
	*bufferPointer = append(*bufferPointer, jsonArgsPrefixNoOpen...)

	encodeArgs(bufferPointer, payload.Args)
	*bufferPointer = append(*bufferPointer, '}')
}

// encodeArgs encodes the Args slice as JSON.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append the encoded
// output to.
// Takes arguments ([]templater_dto.ActionArgument) which contains the arguments to
// encode.
func encodeArgs(bufferPointer *[]byte, arguments []templater_dto.ActionArgument) {
	if arguments == nil {
		*bufferPointer = append(*bufferPointer, jsonNull...)
		return
	}

	if len(arguments) == 0 {
		*bufferPointer = append(*bufferPointer, jsonEmptyArray...)
		return
	}

	*bufferPointer = append(*bufferPointer, '[')
	for i := range arguments {
		if i > 0 {
			*bufferPointer = append(*bufferPointer, ',')
		}
		encodeArg(bufferPointer, &arguments[i])
	}
	*bufferPointer = append(*bufferPointer, ']')
}

// encodeArg encodes a single ActionArgument as JSON.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append to.
// Takes argument (*templater_dto.ActionArgument) which is the argument to encode.
func encodeArg(bufferPointer *[]byte, argument *templater_dto.ActionArgument) {
	*bufferPointer = append(*bufferPointer, `{"t":"`...)
	escapeJSONString(bufferPointer, argument.Type)
	*bufferPointer = append(*bufferPointer, '"')

	if argument.Value != nil {
		*bufferPointer = append(*bufferPointer, `,"v":`...)
		encodeValue(bufferPointer, argument.Value)
	}

	*bufferPointer = append(*bufferPointer, '}')
}

// encodeValue encodes a value as JSON using type-specific fast paths.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append the encoded
// value to.
// Takes v (any) which is the value to encode.
func encodeValue(bufferPointer *[]byte, v any) {
	switch value := v.(type) {
	case string:
		encodeString(bufferPointer, value)
	case int:
		*bufferPointer = strconv.AppendInt(*bufferPointer, int64(value), decimalBase)
	case int8, int16, int32, int64:
		encodeSignedInt(bufferPointer, value)
	case uint:
		*bufferPointer = strconv.AppendUint(*bufferPointer, uint64(value), decimalBase)
	case uint8, uint16, uint32, uint64:
		encodeUnsignedInt(bufferPointer, value)
	case float32:
		*bufferPointer = strconv.AppendFloat(*bufferPointer, float64(value), 'g', -1, float32BitSize)
	case float64:
		*bufferPointer = strconv.AppendFloat(*bufferPointer, value, 'g', -1, float64BitSize)
	case bool:
		*bufferPointer = strconv.AppendBool(*bufferPointer, value)
	case nil:
		*bufferPointer = append(*bufferPointer, jsonNull...)
	case map[string]any:
		encodeMap(bufferPointer, value)
	case []byte:
		encodeString(bufferPointer, string(value))
	case []any:
		encodeSlice(bufferPointer, value)
	default:
		encodeReflectValue(bufferPointer, value)
	}
}

// encodeSignedInt appends a signed integer value to the buffer.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append to.
// Takes v (any) which is an int8, int16, int32, or int64 value.
func encodeSignedInt(bufferPointer *[]byte, v any) {
	switch value := v.(type) {
	case int8:
		*bufferPointer = strconv.AppendInt(*bufferPointer, int64(value), decimalBase)
	case int16:
		*bufferPointer = strconv.AppendInt(*bufferPointer, int64(value), decimalBase)
	case int32:
		*bufferPointer = strconv.AppendInt(*bufferPointer, int64(value), decimalBase)
	case int64:
		*bufferPointer = strconv.AppendInt(*bufferPointer, value, decimalBase)
	}
}

// encodeUnsignedInt appends an unsigned integer value to the buffer.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append to.
// Takes v (any) which is a uint8, uint16, uint32, or uint64 value.
func encodeUnsignedInt(bufferPointer *[]byte, v any) {
	switch value := v.(type) {
	case uint8:
		*bufferPointer = strconv.AppendUint(*bufferPointer, uint64(value), decimalBase)
	case uint16:
		*bufferPointer = strconv.AppendUint(*bufferPointer, uint64(value), decimalBase)
	case uint32:
		*bufferPointer = strconv.AppendUint(*bufferPointer, uint64(value), decimalBase)
	case uint64:
		*bufferPointer = strconv.AppendUint(*bufferPointer, value, decimalBase)
	}
}

// encodeReflectValue handles encoding of non-primitive types using reflection
// for maps, slices, and arrays, falling back to string conversion for other
// types.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append to.
// Takes value (any) which is the value to encode.
func encodeReflectValue(bufferPointer *[]byte, value any) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		encodeReflectMap(bufferPointer, rv)
	case reflect.Slice, reflect.Array:
		encodeReflectSlice(bufferPointer, rv)
	default:
		encodeString(bufferPointer, convertValueToString(value))
	}
}

// encodeString encodes a string value with quotes.
//
// Takes bufferPointer (*[]byte) which is the buffer to append the
// encoded string to.
// Takes s (string) which is the value to encode.
func encodeString(bufferPointer *[]byte, s string) {
	*bufferPointer = append(*bufferPointer, '"')
	escapeJSONString(bufferPointer, s)
	*bufferPointer = append(*bufferPointer, '"')
}

// encodeMap encodes a map[string]any as a JSON object, recursively encoding
// each value.
//
// Takes bufferPointer (*[]byte) which is the buffer to append to.
// Takes m (map[string]any) which is the map to encode.
func encodeMap(bufferPointer *[]byte, m map[string]any) {
	*bufferPointer = append(*bufferPointer, '{')
	first := true
	for k, v := range m {
		if !first {
			*bufferPointer = append(*bufferPointer, ',')
		}
		first = false
		encodeString(bufferPointer, k)
		*bufferPointer = append(*bufferPointer, ':')
		encodeValue(bufferPointer, v)
	}
	*bufferPointer = append(*bufferPointer, '}')
}

// encodeSlice encodes a []any as a JSON array, recursively encoding each
// element.
//
// Takes bufferPointer (*[]byte) which is the buffer to append to.
// Takes s ([]any) which is the slice to encode.
func encodeSlice(bufferPointer *[]byte, s []any) {
	*bufferPointer = append(*bufferPointer, '[')
	for i, v := range s {
		if i > 0 {
			*bufferPointer = append(*bufferPointer, ',')
		}
		encodeValue(bufferPointer, v)
	}
	*bufferPointer = append(*bufferPointer, ']')
}

// encodeReflectMap encodes any map with string keys as a JSON object using
// reflection. This is the slow path for map types other than map[string]any.
//
// Takes bufferPointer (*[]byte) which is the buffer to append to.
// Takes rv (reflect.Value) which is the map value to encode.
func encodeReflectMap(bufferPointer *[]byte, rv reflect.Value) {
	*bufferPointer = append(*bufferPointer, '{')
	first := true
	iterator := rv.MapRange()
	for iterator.Next() {
		if !first {
			*bufferPointer = append(*bufferPointer, ',')
		}
		first = false
		encodeString(bufferPointer, iterator.Key().String())
		*bufferPointer = append(*bufferPointer, ':')
		encodeValue(bufferPointer, iterator.Value().Interface())
	}
	*bufferPointer = append(*bufferPointer, '}')
}

// encodeReflectSlice encodes any slice or array as a JSON array using
// reflection. This is the slow path for slice types other than []any.
//
// Takes bufferPointer (*[]byte) which is the buffer to append to.
// Takes rv (reflect.Value) which is the slice/array value to encode.
func encodeReflectSlice(bufferPointer *[]byte, rv reflect.Value) {
	*bufferPointer = append(*bufferPointer, '[')
	for i := range rv.Len() {
		if i > 0 {
			*bufferPointer = append(*bufferPointer, ',')
		}
		encodeValue(bufferPointer, rv.Index(i).Interface())
	}
	*bufferPointer = append(*bufferPointer, ']')
}

// convertValueToString converts an unknown value to string.
//
// Takes v (any) which is the value to convert.
//
// Returns string which is the string representation of the value.
func convertValueToString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ValueToString(v)
	}
}

// escapeJSONString appends a JSON-escaped string to the buffer.
// Handles all characters requiring escaping per RFC 8259.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append
// escaped bytes to.
// Takes s (string) which is the string to escape.
func escapeJSONString(bufferPointer *[]byte, s string) {
	for i := range len(s) {
		c := s[i]
		switch c {
		case '"':
			*bufferPointer = append(*bufferPointer, '\\', '"')
		case '\\':
			*bufferPointer = append(*bufferPointer, '\\', '\\')
		case '\n':
			*bufferPointer = append(*bufferPointer, '\\', 'n')
		case '\r':
			*bufferPointer = append(*bufferPointer, '\\', 'r')
		case '\t':
			*bufferPointer = append(*bufferPointer, '\\', 't')
		case '\b':
			*bufferPointer = append(*bufferPointer, '\\', 'b')
		case '\f':
			*bufferPointer = append(*bufferPointer, '\\', 'f')
		default:
			if c < controlCharMax {
				appendControlChar(bufferPointer, c)
			} else {
				*bufferPointer = append(*bufferPointer, c)
			}
		}
	}
}

// appendControlChar appends a control character as a \u00XX escape sequence.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append to.
// Takes c (byte) which is the control character to escape.
func appendControlChar(bufferPointer *[]byte, c byte) {
	*bufferPointer = append(*bufferPointer, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0x0f])
}

// encodeBase64InPlace performs base64 encoding in-place within the buffer.
//
// Takes bufferPointer (*[]byte) which points to the buffer to encode and resize.
func encodeBase64InPlace(bufferPointer *[]byte) {
	jsonLen := len(*bufferPointer)
	encodedLen := base64.RawURLEncoding.EncodedLen(jsonLen)
	totalLen := jsonLen + encodedLen

	if cap(*bufferPointer) < totalLen {
		expandBuffer(bufferPointer, jsonLen, totalLen)
	} else {
		*bufferPointer = (*bufferPointer)[:totalLen]
	}

	base64.RawURLEncoding.Encode((*bufferPointer)[jsonLen:], (*bufferPointer)[:jsonLen])

	copy(*bufferPointer, (*bufferPointer)[jsonLen:])
	*bufferPointer = (*bufferPointer)[:encodedLen]
}

// expandBuffer grows the buffer when more space is needed.
//
// This handles the rare case where the buffer must expand. The old
// buffer is not returned to the pool because bufferPointer was
// obtained from the pool and will be reassigned. Returning it
// would allow another goroutine to Get it, and the subsequent
// reassignment would corrupt their data. The old slice's
// underlying array will be garbage collected.
//
// Takes bufferPointer (*[]byte) which points to the buffer to expand.
// Takes jsonLen (int) which specifies how many bytes to copy from the original.
// Takes totalLen (int) which specifies the new buffer size.
func expandBuffer(bufferPointer *[]byte, jsonLen, totalLen int) {
	newBuf := make([]byte, totalLen)
	copy(newBuf, (*bufferPointer)[:jsonLen])
	*bufferPointer = newBuf
}

// encodePayloadJSON0 builds JSON for a payload with zero arguments.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append JSON data to.
// Takes function (string) which specifies the function name to encode.
func encodePayloadJSON0(bufferPointer *[]byte, function string) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, function)
	*bufferPointer = append(*bufferPointer, jsonPayloadSuffixZeroArgs...)
}

// encodePayloadJSON1 builds JSON for a payload with one argument.
//
// Takes bufferPointer (*[]byte) which points to the buffer to append the JSON to.
// Takes function (string) which is the function name to include in the payload.
// Takes argument0 (*templater_dto.ActionArgument) which is the argument to encode.
func encodePayloadJSON1(bufferPointer *[]byte, function string, argument0 *templater_dto.ActionArgument) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, function)
	*bufferPointer = append(*bufferPointer, jsonArgsPrefix...)
	encodeArg(bufferPointer, argument0)
	*bufferPointer = append(*bufferPointer, jsonPayloadSuffix...)
}

// encodePayloadJSON2 builds JSON for a payload with two arguments.
//
// Takes bufferPointer (*[]byte) which receives the encoded JSON output.
// Takes function (string) which specifies the function name in the payload.
// Takes argument0 (*templater_dto.ActionArgument) which provides the
// first argument.
// Takes argument1 (*templater_dto.ActionArgument) which provides the
// second argument.
func encodePayloadJSON2(bufferPointer *[]byte, function string, argument0, argument1 *templater_dto.ActionArgument) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, function)
	*bufferPointer = append(*bufferPointer, jsonArgsPrefix...)
	encodeArg(bufferPointer, argument0)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument1)
	*bufferPointer = append(*bufferPointer, jsonPayloadSuffix...)
}

// encodePayloadJSON3 builds JSON for a payload with three arguments.
//
// Takes bufferPointer (*[]byte) which receives the encoded JSON output.
// Takes function (string) which specifies the function name in the payload.
// Takes argument0 (*templater_dto.ActionArgument) which provides the
// first argument.
// Takes argument1 (*templater_dto.ActionArgument) which provides the
// second argument.
// Takes argument2 (*templater_dto.ActionArgument) which provides the
// third argument.
func encodePayloadJSON3(bufferPointer *[]byte, function string, argument0, argument1, argument2 *templater_dto.ActionArgument) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, function)
	*bufferPointer = append(*bufferPointer, jsonArgsPrefix...)
	encodeArg(bufferPointer, argument0)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument1)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument2)
	*bufferPointer = append(*bufferPointer, jsonPayloadSuffix...)
}

// encodePayloadJSON4 builds JSON for a payload with four arguments.
//
// Takes bufferPointer (*[]byte) which is the buffer to append the JSON to.
// Takes function (string) which is the function name to encode.
// Takes argument0 (*templater_dto.ActionArgument) which is the first argument.
// Takes argument1 (*templater_dto.ActionArgument) which is the second argument.
// Takes argument2 (*templater_dto.ActionArgument) which is the third argument.
// Takes argument3 (*templater_dto.ActionArgument) which is the fourth argument.
func encodePayloadJSON4(bufferPointer *[]byte, function string, argument0, argument1, argument2, argument3 *templater_dto.ActionArgument) {
	*bufferPointer = append(*bufferPointer, jsonPayloadPrefix...)
	escapeJSONString(bufferPointer, function)
	*bufferPointer = append(*bufferPointer, jsonArgsPrefix...)
	encodeArg(bufferPointer, argument0)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument1)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument2)
	*bufferPointer = append(*bufferPointer, ',')
	encodeArg(bufferPointer, argument3)
	*bufferPointer = append(*bufferPointer, jsonPayloadSuffix...)
}
