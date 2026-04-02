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

func TestEvalMapCommaOk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "present_key_value",
			code:   "m := map[string]int{\"a\": 42}\nv, _ := m[\"a\"]\nv",
			expect: int64(42),
		},
		{
			name:   "absent_key_value",
			code:   "m := map[string]int{\"a\": 42}\nv, _ := m[\"b\"]\nv",
			expect: int64(0),
		},
		{
			name: "present_key_with_ok",
			code: `func check() int {
	m := map[string]int{"x": 99}
	v, ok := m["x"]
	if ok {
		return v
	}
	return -1
}
check()`,
			expect: int64(99),
		},
		{
			name: "absent_key_with_ok",
			code: `func check() int {
	m := map[string]int{"x": 99}
	_, ok := m["y"]
	if ok {
		return 1
	}
	return 0
}
check()`,
			expect: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}
