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

package bootstrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeref(t *testing.T) {
	t.Run("string nil returns fallback", func(t *testing.T) {
		assert.Equal(t, "default", deref((*string)(nil), "default"))
	})
	t.Run("string non-nil returns value", func(t *testing.T) {
		assert.Equal(t, "hello", deref(new("hello"), "default"))
	})
	t.Run("string explicit empty overrides fallback", func(t *testing.T) {
		assert.Equal(t, "", deref(new(""), "default"))
	})

	t.Run("bool nil returns fallback", func(t *testing.T) {
		assert.Equal(t, true, deref((*bool)(nil), true))
	})
	t.Run("bool true returns true", func(t *testing.T) {
		assert.Equal(t, true, deref(new(true), false))
	})
	t.Run("bool explicit false overrides true fallback", func(t *testing.T) {
		assert.Equal(t, false, deref(new(false), true))
	})

	t.Run("int nil returns fallback", func(t *testing.T) {
		assert.Equal(t, 42, deref((*int)(nil), 42))
	})
	t.Run("int non-nil returns value", func(t *testing.T) {
		assert.Equal(t, 7, deref(new(7), 42))
	})
	t.Run("int explicit zero overrides fallback", func(t *testing.T) {
		assert.Equal(t, 0, deref(new(0), 42))
	})

	t.Run("float64 nil returns fallback", func(t *testing.T) {
		assert.InDelta(t, 0.05, deref((*float64)(nil), 0.05), 1e-9)
	})
	t.Run("float64 non-nil returns value", func(t *testing.T) {
		assert.InDelta(t, 1.0, deref(new(1.0), 0.05), 1e-9)
	})
	t.Run("float64 explicit zero overrides fallback", func(t *testing.T) {
		assert.InDelta(t, 0.0, deref(new(0.0), 0.05), 1e-9)
	})

	t.Run("duration nil returns fallback", func(t *testing.T) {
		assert.Equal(t, 5*time.Minute, deref((*time.Duration)(nil), 5*time.Minute))
	})
	t.Run("duration non-nil returns value", func(t *testing.T) {
		assert.Equal(t, 10*time.Second, deref(new(10*time.Second), 5*time.Minute))
	})
	t.Run("duration explicit zero overrides fallback", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), deref(new(time.Duration(0)), 5*time.Minute))
	})

	t.Run("int32 nil returns fallback", func(t *testing.T) {
		assert.Equal(t, int32(10), deref((*int32)(nil), int32(10)))
	})
	t.Run("int32 non-nil returns value", func(t *testing.T) {
		assert.Equal(t, int32(5), deref(new(int32(5)), int32(10)))
	})

	t.Run("int64 nil returns fallback", func(t *testing.T) {
		assert.Equal(t, int64(1048576), deref((*int64)(nil), int64(1048576)))
	})
	t.Run("int64 non-nil returns value", func(t *testing.T) {
		assert.Equal(t, int64(2048), deref(new(int64(2048)), int64(1048576)))
	})
}
