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

func TestSafeModeBoundedUnsafeAllowsInBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		want any
		name string
		code string
	}{
		{
			name: "SliceData then in-bounds Slice",
			code: `import "unsafe"
b := []int64{10, 20, 30}
p := unsafe.SliceData(b)
s := unsafe.Slice(p, 3)
s[0] + s[1] + s[2]`,
			want: int64(60),
		},
		{
			name: "Add within the origin window",
			code: `import "unsafe"
b := []int64{10, 20, 30}
p := unsafe.Pointer(unsafe.SliceData(b))
q := (*int64)(unsafe.Add(p, 8))
s := unsafe.Slice(q, 2)
s[0] + s[1]`,
			want: int64(50),
		},
		{
			name: "StringData then String round-trip",
			code: `import "unsafe"
src := "hello"
p := unsafe.StringData(src)
out := unsafe.String(p, 5)
out`,
			want: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fast, fastErr := NewService().Eval(context.Background(), tt.code)
			require.NoError(t, fastErr)
			require.Equal(t, tt.want, fast)

			safe, safeErr := NewService(WithSafeMode()).Eval(context.Background(), tt.code)
			require.NoError(t, safeErr, "in-bounds unsafe must still work in safe mode")
			require.Equal(t, tt.want, safe)
		})
	}
}

func TestSafeModeBoundedUnsafeRejectsEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
	}{
		{
			name: "Add beyond the origin window",
			code: `import "unsafe"
b := []int64{10, 20, 30}
p := unsafe.Pointer(unsafe.SliceData(b))
q := (*int64)(unsafe.Add(p, 8000))
unsafe.Slice(q, 1)[0]`,
		},
		{
			name: "Slice length overruns the backing array",
			code: `import "unsafe"
b := []int64{10, 20, 30}
p := unsafe.SliceData(b)
s := unsafe.Slice(p, 1000)
s[500]`,
		},
		{
			name: "String length overruns the buffer",
			code: `import "unsafe"
src := "hi"
p := unsafe.StringData(src)
out := unsafe.String(p, 1000)
len(out)`,
		},
		{
			name: "uintptr round-trip is rejected",
			code: `import "unsafe"
x := 5
u := uintptr(unsafe.Pointer(&x))
int(u) - int(u)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewService(WithSafeMode()).Eval(context.Background(), tt.code)
			require.Error(t, err, "safe mode must reject the escape")
			require.ErrorContains(t, err, "out of bounds")
		})
	}
}
