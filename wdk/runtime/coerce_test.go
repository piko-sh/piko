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

package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoerceInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   int
		wantOk bool
	}{
		{"int", int(42), 42, true},
		{"int64", int64(42), 42, true},
		{"int32", int32(42), 42, true},
		{"int16", int16(42), 42, true},
		{"int8", int8(42), 42, true},
		{"float64", float64(42.9), 42, true},
		{"float32", float32(42.9), 42, true},
		{"string fails", "42", 0, false},
		{"nil fails", nil, 0, false},
		{"bool fails", true, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceInt(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   int64
		wantOk bool
	}{
		{"int64", int64(42), 42, true},
		{"int", int(42), 42, true},
		{"float64", float64(42.0), 42, true},
		{"string fails", "42", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceInt64(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   float64
		wantOk bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"float32", float32(3.14), float64(float32(3.14)), true},
		{"int", int(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"string fails", "3.14", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceFloat64(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceFloat32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   float32
		wantOk bool
	}{
		{"float32", float32(3.14), 3.14, true},
		{"float64", float64(3.14), float32(3.14), true},
		{"int", int(42), 42.0, true},
		{"string fails", "3.14", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceFloat32(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   []string
		wantOk bool
	}{
		{
			"[]string",
			[]string{"a", "b"},
			[]string{"a", "b"},
			true,
		},
		{
			"[]any with strings",
			[]any{"a", "b", "c"},
			[]string{"a", "b", "c"},
			true,
		},
		{
			"[]any with mixed types fails",
			[]any{"a", 42},
			nil,
			false,
		},
		{
			"empty []any",
			[]any{},
			[]string{},
			true,
		},
		{
			"string fails",
			"not a slice",
			nil,
			false,
		},
		{
			"nil fails",
			nil,
			nil,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceStringSlice(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceTime(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		input  any
		want   time.Time
		wantOk bool
	}{
		{
			"time.Time",
			fixedTime,
			fixedTime,
			true,
		},
		{
			"RFC3339 string",
			"2026-03-15T00:00:00Z",
			fixedTime,
			true,
		},
		{
			"date-only string",
			"2026-03-15",
			fixedTime,
			true,
		},
		{
			"invalid string fails",
			"not-a-date",
			time.Time{},
			false,
		},
		{
			"int fails",
			42,
			time.Time{},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CoerceTime(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCoerceSignedInt(t *testing.T) {
	t.Parallel()

	t.Run("int to int32", func(t *testing.T) {
		t.Parallel()
		got, ok := CoerceSignedInt[int32](int(42))
		assert.True(t, ok)
		assert.Equal(t, int32(42), got)
	})

	t.Run("float64 to int16", func(t *testing.T) {
		t.Parallel()
		got, ok := CoerceSignedInt[int16](float64(100))
		assert.True(t, ok)
		assert.Equal(t, int16(100), got)
	})

	t.Run("string fails", func(t *testing.T) {
		t.Parallel()
		_, ok := CoerceSignedInt[int32]("42")
		assert.False(t, ok)
	})
}

func TestCoerceUnsignedInt(t *testing.T) {
	t.Parallel()

	t.Run("int to uint32", func(t *testing.T) {
		t.Parallel()
		got, ok := CoerceUnsignedInt[uint32](int(42))
		assert.True(t, ok)
		assert.Equal(t, uint32(42), got)
	})

	t.Run("negative int fails", func(t *testing.T) {
		t.Parallel()
		_, ok := CoerceUnsignedInt[uint32](int(-1))
		assert.False(t, ok)
	})

	t.Run("uint64 to uint16", func(t *testing.T) {
		t.Parallel()
		got, ok := CoerceUnsignedInt[uint16](uint64(255))
		assert.True(t, ok)
		assert.Equal(t, uint16(255), got)
	})

	t.Run("string fails", func(t *testing.T) {
		t.Parallel()
		_, ok := CoerceUnsignedInt[uint32]("42")
		assert.False(t, ok)
	})
}

func TestMetadataGet(t *testing.T) {
	t.Parallel()

	m := map[string]any{
		"Title":       "Hello",
		"Slug":        "hello",
		"ReadingTime": 5,
		"author":      map[string]any{"name": "Jane"},
	}

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()
		v, ok := MetadataGet(m, "Title")
		assert.True(t, ok)
		assert.Equal(t, "Hello", v)
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		t.Parallel()
		v, ok := MetadataGet(m, "title")
		assert.True(t, ok)
		assert.Equal(t, "Hello", v)
	})

	t.Run("case-insensitive match for camelCase key", func(t *testing.T) {
		t.Parallel()
		v, ok := MetadataGet(m, "readingtime")
		assert.True(t, ok)
		assert.Equal(t, 5, v)
	})

	t.Run("custom field exact match", func(t *testing.T) {
		t.Parallel()
		v, ok := MetadataGet(m, "author")
		assert.True(t, ok)
		assert.NotNil(t, v)
	})

	t.Run("missing key returns false", func(t *testing.T) {
		t.Parallel()
		_, ok := MetadataGet(m, "nonexistent")
		assert.False(t, ok)
	})

	t.Run("nil map returns false", func(t *testing.T) {
		t.Parallel()
		_, ok := MetadataGet(nil, "Title")
		assert.False(t, ok)
	})
}
