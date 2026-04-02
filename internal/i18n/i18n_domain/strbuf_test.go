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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrBuf_WriteString(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteString("Hello")
	buffer.WriteString(", ")
	buffer.WriteString("World!")

	assert.Equal(t, "Hello, World!", buffer.String())
}

func TestStrBuf_WriteByte(t *testing.T) {
	buffer := NewStrBuf(64)
	_ = buffer.WriteByte('H')
	_ = buffer.WriteByte('i')

	assert.Equal(t, "Hi", buffer.String())
}

func TestStrBuf_WriteInt(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		value    int
	}{
		{name: "positive", value: 42, expected: "42"},
		{name: "zero", value: 0, expected: "0"},
		{name: "negative", value: -123, expected: "-123"},
		{name: "large", value: 1234567890, expected: "1234567890"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buffer := NewStrBuf(64)
			buffer.WriteInt(tc.value)
			assert.Equal(t, tc.expected, buffer.String())
		})
	}
}

func TestStrBuf_WriteInt64(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteInt64(9223372036854775807)

	assert.Equal(t, "9223372036854775807", buffer.String())
}

func TestStrBuf_WriteFloat(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		value    float64
	}{
		{name: "integer", value: 42.0, expected: "42"},
		{name: "decimal", value: 3.14, expected: "3.14"},
		{name: "negative", value: -1.5, expected: "-1.5"},
		{name: "zero", value: 0.0, expected: "0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buffer := NewStrBuf(64)
			buffer.WriteFloat(tc.value)
			assert.Equal(t, tc.expected, buffer.String())
		})
	}
}

func TestStrBuf_WriteFloatPrec(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteFloatPrec(3.14159, 2)

	assert.Equal(t, "3.14", buffer.String())
}

func TestStrBuf_WriteBool(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteBool(true)
	buffer.WriteString(" ")
	buffer.WriteBool(false)

	assert.Equal(t, "true false", buffer.String())
}

func TestStrBuf_WriteAny(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{name: "string", value: "hello", expected: "hello"},
		{name: "int", value: 42, expected: "42"},
		{name: "int64", value: int64(100), expected: "100"},
		{name: "float64", value: 3.14, expected: "3.14"},
		{name: "float32", value: float32(2.5), expected: "2.5"},
		{name: "bool true", value: true, expected: "true"},
		{name: "bool false", value: false, expected: "false"},
		{name: "bytes", value: []byte("data"), expected: "data"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buffer := NewStrBuf(64)
			buffer.WriteAny(tc.value)
			assert.Equal(t, tc.expected, buffer.String())
		})
	}
}

func TestStrBuf_Reset(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteString("Hello")
	assert.Equal(t, 5, buffer.Len())

	buffer.Reset()
	assert.Equal(t, 0, buffer.Len())
	assert.Equal(t, "", buffer.String())

	buffer.WriteString("World")
	assert.Equal(t, "World", buffer.String())
}

func TestStrBuf_Len(t *testing.T) {
	buffer := NewStrBuf(64)
	assert.Equal(t, 0, buffer.Len())

	buffer.WriteString("Hello")
	assert.Equal(t, 5, buffer.Len())
}

func TestStrBuf_Cap(t *testing.T) {
	buffer := NewStrBuf(128)
	assert.GreaterOrEqual(t, buffer.Cap(), 128)
}

func TestStrBuf_Bytes(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteString("Hello")

	bytes := buffer.Bytes()
	assert.Equal(t, []byte("Hello"), bytes)
}

func TestStrBuf_UnsafeString(t *testing.T) {
	buffer := NewStrBuf(64)
	buffer.WriteString("Hello, World!")

	s := buffer.UnsafeString()
	assert.Equal(t, "Hello, World!", s)
}

func TestStrBuf_UnsafeString_Empty(t *testing.T) {
	buffer := NewStrBuf(64)
	s := buffer.UnsafeString()
	assert.Equal(t, "", s)
}

func TestStrBuf_GrowsAutomatically(t *testing.T) {
	buffer := NewStrBuf(4)
	longString := "This is a much longer string that exceeds the initial capacity"
	buffer.WriteString(longString)

	assert.Equal(t, longString, buffer.String())
	assert.GreaterOrEqual(t, buffer.Cap(), len(longString))
}

func TestStrBufPool_GetPut(t *testing.T) {
	pool := NewStrBufPool(64)

	buf1 := pool.Get()
	require.NotNil(t, buf1)
	buf1.WriteString("test")

	pool.Put(buf1)

	buf2 := pool.Get()
	require.NotNil(t, buf2)
	assert.Equal(t, 0, buf2.Len())
}

func TestStrBufPool_Reuse(t *testing.T) {
	pool := NewStrBufPool(64)

	for range 10 {
		buffer := pool.Get()
		buffer.WriteString("iteration")
		pool.Put(buffer)
	}

	buffer := pool.Get()
	buffer.WriteString("final")
	assert.Equal(t, "final", buffer.String())
}

func BenchmarkStrBuf_WriteString(b *testing.B) {
	buffer := NewStrBuf(256)
	b.ResetTimer()

	for b.Loop() {
		buffer.Reset()
		buffer.WriteString("Hello, ")
		buffer.WriteString("World!")
	}
}

func BenchmarkStrBuf_WriteInt(b *testing.B) {
	buffer := NewStrBuf(64)
	b.ResetTimer()

	for b.Loop() {
		buffer.Reset()
		buffer.WriteInt(12345)
	}
}

func BenchmarkStrBuf_WriteAny(b *testing.B) {
	buffer := NewStrBuf(64)
	values := []any{"hello", 42, 3.14, true}
	b.ResetTimer()

	for b.Loop() {
		buffer.Reset()
		for _, v := range values {
			buffer.WriteAny(v)
		}
	}
}

func BenchmarkStrBufPool(b *testing.B) {
	pool := NewStrBufPool(64)
	b.ResetTimer()

	for b.Loop() {
		buffer := pool.Get()
		buffer.WriteString("test")
		pool.Put(buffer)
	}
}
