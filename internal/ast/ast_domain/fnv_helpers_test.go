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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendFNVString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantSame bool
	}{
		{
			name:     "empty string",
			input:    "",
			wantLen:  8,
			wantSame: true,
		},
		{
			name:     "simple string",
			input:    "hello",
			wantLen:  8,
			wantSame: true,
		},
		{
			name:     "string with HTML chars",
			input:    "<script>alert('xss')</script>",
			wantLen:  8,
			wantSame: true,
		},
		{
			name:     "very long string",
			input:    "this is a very long string that would be problematic as a key because it is so long that it would make the DOM huge",
			wantLen:  8,
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result1 := AppendFNVString(nil, tt.input)
			assert.Len(t, result1, tt.wantLen, "output should be exactly 8 hex chars")

			for _, c := range result1 {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
					"output should only contain hex chars, got %c", c)
			}

			if tt.wantSame {
				result2 := AppendFNVString(nil, tt.input)
				assert.Equal(t, result1, result2, "same input should produce same output")
			}
		})
	}
}

func TestAppendFNVFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		wantLen int
	}{
		{
			name:    "simple float",
			input:   3.14,
			wantLen: 8,
		},
		{
			name:    "precision problem float",
			input:   0.1 + 0.2,
			wantLen: 8,
		},
		{
			name:    "zero",
			input:   0.0,
			wantLen: 8,
		},
		{
			name:    "negative",
			input:   -42.5,
			wantLen: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendFNVFloat(nil, tt.input)
			assert.Len(t, result, tt.wantLen, "output should be exactly 8 hex chars")

			result2 := AppendFNVFloat(nil, tt.input)
			assert.Equal(t, result, result2, "same input should produce same output")
		})
	}
}

func TestAppendFNVAny(t *testing.T) {
	tests := []struct {
		input   any
		name    string
		wantLen int
	}{
		{
			name:    "nil",
			input:   nil,
			wantLen: 0,
		},
		{
			name:    "string",
			input:   "hello",
			wantLen: 8,
		},
		{
			name:    "int",
			input:   42,
			wantLen: 8,
		},
		{
			name:    "struct",
			input:   struct{ Name string }{Name: "test"},
			wantLen: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendFNVAny(nil, tt.input)
			assert.Len(t, result, tt.wantLen)

			if tt.wantLen > 0 {

				result2 := AppendFNVAny(nil, tt.input)
				assert.Equal(t, result, result2, "same input should produce same output")
			}
		})
	}
}

func TestAppendFNVString_DifferentInputsDifferentOutput(t *testing.T) {
	result1 := AppendFNVString(nil, "hello")
	result2 := AppendFNVString(nil, "world")

	assert.NotEqual(t, result1, result2, "different inputs should produce different outputs")
}

func TestAppendFNVString_AppendsToBuf(t *testing.T) {
	prefix := []byte("prefix:")
	result := AppendFNVString(prefix, "hello")

	assert.True(t, len(result) > len(prefix), "result should be longer than prefix")
	assert.Equal(t, "prefix:", string(result[:7]), "prefix should be preserved")
	assert.Len(t, result, 7+8, "total length should be prefix + 8 hex chars")
}

func TestWriteHex32ToBuf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		sum  uint32
	}{
		{
			name: "zero produces all zeroes",
			sum:  0,
			want: "00000000",
		},
		{
			name: "0xFF produces known hex",
			sum:  0xFF,
			want: "000000ff",
		},
		{
			name: "0xDEADBEEF produces known hex",
			sum:  0xDEADBEEF,
			want: "deadbeef",
		},
		{
			name: "max uint32 produces all f's",
			sum:  0xFFFFFFFF,
			want: "ffffffff",
		},
		{
			name: "0x12345678 produces expected hex",
			sum:  0x12345678,
			want: "12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buffer := make([]byte, fnvHexBufSize)
			writeHex32ToBuf(buffer, tt.sum)
			assert.Equal(t, tt.want, string(buffer))
		})
	}
}

func TestFNVHexBuf_BytesAndRelease(t *testing.T) {
	t.Run("get buffer and verify Bytes returns 8 bytes", func(t *testing.T) {
		buffer := GetFNVStringBuf("hello")
		data := buffer.Bytes()
		require.NotNil(t, data, "Bytes() should return non-nil before Release")
		assert.Len(t, data, fnvHexBufSize, "Bytes() should return exactly 8 bytes")

		for _, c := range data {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"expected hex character, got %c", c)
		}

		buffer.Release()
		assert.Nil(t, buffer.Bytes(), "Bytes() should return nil after Release")
	})

	t.Run("double Release does not panic", func(t *testing.T) {
		buffer := GetFNVStringBuf("test")
		assert.NotPanics(t, func() {
			buffer.Release()
			buffer.Release()
		})
	})

	t.Run("uninitialised FNVHexBuf returns nil from Bytes", func(t *testing.T) {
		var buffer FNVHexBuf
		assert.Nil(t, buffer.Bytes(), "uninitialised buffer should return nil from Bytes()")
	})

	t.Run("Release on uninitialised FNVHexBuf does not panic", func(t *testing.T) {
		var buffer FNVHexBuf
		assert.NotPanics(t, func() {
			buffer.Release()
		})
	})
}

func TestGetFNVStringBuf(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "hello produces consistent output",
			input: "hello",
		},
		{
			name:  "empty string produces valid buffer",
			input: "",
		},
		{
			name:  "string with special characters",
			input: "<div class=\"test\">",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := GetFNVStringBuf(tt.input)
			defer buffer.Release()

			data := buffer.Bytes()
			require.NotNil(t, data, "Bytes() should be non-nil")
			assert.Len(t, data, fnvHexBufSize, "should be 8 bytes")

			appendResult := AppendFNVString(nil, tt.input)
			assert.Equal(t, string(appendResult), string(data),
				"GetFNVStringBuf and AppendFNVString should produce identical output")
		})
	}
}

func TestGetFNVFloatBuf(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{
			name:  "zero",
			input: 0.0,
		},
		{
			name:  "pi",
			input: 3.14,
		},
		{
			name:  "negative",
			input: -1.0,
		},
		{
			name:  "large float",
			input: 1e15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := GetFNVFloatBuf(tt.input)
			defer buffer.Release()

			data := buffer.Bytes()
			require.NotNil(t, data, "Bytes() should be non-nil")
			assert.Len(t, data, fnvHexBufSize, "should be 8 bytes")

			appendResult := AppendFNVFloat(nil, tt.input)
			assert.Equal(t, string(appendResult), string(data),
				"GetFNVFloatBuf and AppendFNVFloat should produce identical output")
		})
	}
}

func TestGetFNVAnyBuf(t *testing.T) {
	tests := []struct {
		input   any
		name    string
		wantNil bool
	}{
		{
			name:    "nil returns empty buffer",
			input:   nil,
			wantNil: true,
		},
		{
			name:  "string value",
			input: "str",
		},
		{
			name:  "int value",
			input: 42,
		},
		{
			name:  "bool value",
			input: true,
		},
		{
			name:  "struct with Stringer",
			input: NodeElement,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := GetFNVAnyBuf(tt.input)
			defer buffer.Release()

			if tt.wantNil {
				assert.Nil(t, buffer.Bytes(),
					"nil input should produce nil Bytes()")
				return
			}

			data := buffer.Bytes()
			require.NotNil(t, data, "Bytes() should be non-nil for non-nil input")
			assert.Len(t, data, fnvHexBufSize, "should be 8 bytes")

			appendResult := AppendFNVAny(nil, tt.input)
			assert.Equal(t, string(appendResult), string(data),
				"GetFNVAnyBuf and AppendFNVAny should produce identical output")
		})
	}
}
