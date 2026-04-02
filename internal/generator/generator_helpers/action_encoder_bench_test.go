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

//go:build bench

package generator_helpers

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func BenchmarkEncodeActionPayloadBytes_Simple(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args:     nil,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := EncodeActionPayloadBytes(payload)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkEncodeActionPayloadBytes_WithArgs(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleSubmit",
		Args: []templater_dto.ActionArgument{
			{Type: "string", Value: "test-value"},
			{Type: "int", Value: 42},
			{Type: "bool", Value: true},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := EncodeActionPayloadBytes(payload)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkEncodeActionPayloadBytes_LargePayload(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleComplexAction",
		Args: []templater_dto.ActionArgument{
			{Type: "string", Value: "this is a longer string value that might be typical in real usage"},
			{Type: "string", Value: "another-string-value-with-dashes"},
			{Type: "int", Value: 12345},
			{Type: "float", Value: 123.456},
			{Type: "bool", Value: false},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := EncodeActionPayloadBytes(payload)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkEncodeActionPayloadBytes_Parallel(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args: []templater_dto.ActionArgument{
			{Type: "string", Value: "value"},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bufferPointer := EncodeActionPayloadBytes(payload)
			if bufferPointer != nil {
				ast_domain.PutByteBuf(bufferPointer)
			}
		}
	})
}

func BenchmarkEncodeActionPayload_StdLib(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args:     nil,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			b.Fatal(err)
		}
		encoded := base64.RawURLEncoding.EncodeToString(jsonBytes)
		_ = encoded
	}
}

func BenchmarkEncodeActionPayload_HandRolled(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args:     nil,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := ast_domain.GetByteBuf()
		buffer := (*bufferPointer)[:0]

		buffer = append(buffer, `{"f":"`...)
		buffer = append(buffer, payload.Function...)
		buffer = append(buffer, '"')
		if len(payload.Args) > 0 {
			buffer = append(buffer, `,"a":[`...)
			for i, argument := range payload.Args {
				if i > 0 {
					buffer = append(buffer, ',')
				}
				buffer = append(buffer, `{"t":"`...)
				buffer = append(buffer, argument.Type...)
				buffer = append(buffer, '"')
				if argument.Value != nil {
					buffer = append(buffer, `,"v":`...)
					buffer = appendJSONValue(buffer, argument.Value)
				}
				buffer = append(buffer, '}')
			}
			buffer = append(buffer, ']')
		}
		buffer = append(buffer, '}')

		jsonLen := len(buffer)
		encodedLen := base64.RawURLEncoding.EncodedLen(jsonLen)
		totalLen := jsonLen + encodedLen
		if cap(buffer) < totalLen {
			newBuf := make([]byte, totalLen)
			copy(newBuf, buffer[:jsonLen])
			buffer = newBuf
			bufferPointer = &buffer
		} else {
			buffer = buffer[:totalLen]
		}
		base64.RawURLEncoding.Encode(buffer[jsonLen:], buffer[:jsonLen])
		copy(buffer, buffer[jsonLen:])
		buffer = buffer[:encodedLen]
		*bufferPointer = buffer

		ast_domain.PutByteBuf(bufferPointer)
	}
}

func appendJSONValue(buffer []byte, v any) []byte {
	switch value := v.(type) {
	case string:
		buffer = append(buffer, '"')
		buffer = appendEscapedString(buffer, value)
		buffer = append(buffer, '"')
	case int:
		buffer = strconv.AppendInt(buffer, int64(value), 10)
	case int64:
		buffer = strconv.AppendInt(buffer, value, 10)
	case float64:
		buffer = strconv.AppendFloat(buffer, value, 'g', -1, 64)
	case bool:
		buffer = strconv.AppendBool(buffer, value)
	default:

		jsonBytes, _ := json.Marshal(value)
		buffer = append(buffer, jsonBytes...)
	}
	return buffer
}

func appendEscapedString(buffer []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buffer = append(buffer, '\\', '"')
		case '\\':
			buffer = append(buffer, '\\', '\\')
		case '\n':
			buffer = append(buffer, '\\', 'n')
		case '\r':
			buffer = append(buffer, '\\', 'r')
		case '\t':
			buffer = append(buffer, '\\', 't')
		default:
			if c < 0x20 {
				buffer = append(buffer, '\\', 'u', '0', '0')
				buffer = append(buffer, "0123456789abcdef"[c>>4])
				buffer = append(buffer, "0123456789abcdef"[c&0xf])
			} else {
				buffer = append(buffer, c)
			}
		}
	}
	return buffer
}

func BenchmarkEncodeActionPayload_HandRolled_WithArgs(b *testing.B) {
	payload := templater_dto.ActionPayload{
		Function: "handleSubmit",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "test-value"},
			{Type: "s", Value: 42},
			{Type: "s", Value: true},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := ast_domain.GetByteBuf()
		buffer := (*bufferPointer)[:0]

		buffer = append(buffer, `{"f":"`...)
		buffer = append(buffer, payload.Function...)
		buffer = append(buffer, '"')
		if len(payload.Args) > 0 {
			buffer = append(buffer, `,"a":[`...)
			for i, argument := range payload.Args {
				if i > 0 {
					buffer = append(buffer, ',')
				}
				buffer = append(buffer, `{"t":"`...)
				buffer = append(buffer, argument.Type...)
				buffer = append(buffer, '"')
				if argument.Value != nil {
					buffer = append(buffer, `,"v":`...)
					buffer = appendJSONValue(buffer, argument.Value)
				}
				buffer = append(buffer, '}')
			}
			buffer = append(buffer, ']')
		}
		buffer = append(buffer, '}')

		jsonLen := len(buffer)
		encodedLen := base64.RawURLEncoding.EncodedLen(jsonLen)
		totalLen := jsonLen + encodedLen
		if cap(buffer) < totalLen {
			newBuf := make([]byte, totalLen)
			copy(newBuf, buffer[:jsonLen])
			buffer = newBuf
			bufferPointer = &buffer
		} else {
			buffer = buffer[:totalLen]
		}
		base64.RawURLEncoding.Encode(buffer[jsonLen:], buffer[:jsonLen])
		copy(buffer, buffer[jsonLen:])
		buffer = buffer[:encodedLen]
		*bufferPointer = buffer

		ast_domain.PutByteBuf(bufferPointer)
	}
}
