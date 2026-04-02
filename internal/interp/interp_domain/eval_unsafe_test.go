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

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsafeOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"unsafe_string_data_nonempty", `
import "unsafe"
s := "hello"
p := unsafe.StringData(s)
p != nil`, true},
		{"unsafe_string_data_empty", `
import "unsafe"
s := ""
p := unsafe.StringData(s)
p == nil`, true},
		{"unsafe_string_roundtrip", `
import "unsafe"
s := "hi"
p := unsafe.StringData(s)
result := unsafe.String(p, 2)
result`, "hi"},
		{"unsafe_slice_from_array", `
import "unsafe"
arr := [3]int{10, 20, 30}
s := unsafe.Slice(&arr[0], 3)
s[0] + s[1] + s[2]`, int64(60)},
		{"unsafe_slice_data", `
import "unsafe"
s := []int{1, 2, 3}
p := unsafe.SliceData(s)
p != nil`, true},
		{"unsafe_slice_data_empty", `
import "unsafe"
s := make([]int, 0)
p := unsafe.SliceData(s)
p == nil`, true},
		{"unsafe_add_pointer", `
import "unsafe"
arr := [3]int{10, 20, 30}
p := unsafe.Pointer(&arr[0])
p2 := unsafe.Add(p, 8)
result := *(*int)(p2)
result`, int64(20)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestUnsafeOperationsGoDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"unsafe_string_roundtrip", `
import "unsafe"
s := "hi"
p := unsafe.StringData(s)
result := unsafe.String(p, 2)
result`, "hi"},
		{"unsafe_slice", `
import "unsafe"
arr := [3]int{10, 20, 30}
s := unsafe.Slice(&arr[0], 3)
s[2]`, int64(30)},
		{"unsafe_slice_data", `
import "unsafe"
s := []int{1, 2, 3}
p := unsafe.SliceData(s)
p != nil`, true},
		{"unsafe_add", `
import "unsafe"
arr := [3]int{10, 20, 30}
p := unsafe.Pointer(&arr[0])
p2 := unsafe.Add(p, 16)
result := *(*int)(p2)
result`, int64(30)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(WithForceGoDispatch())
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
