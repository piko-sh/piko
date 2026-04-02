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

func TestEvalFallthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "basic_fallthrough",
			code: `x := 1
r := 0
switch x {
case 1:
	r = 10
	fallthrough
case 2:
	r += 20
}
r`,
			expect: int64(30),
		},
		{
			name: "no_fallthrough",
			code: `x := 1
r := 0
switch x {
case 1:
	r = 10
case 2:
	r = 20
}
r`,
			expect: int64(10),
		},
		{
			name: "fallthrough_chain",
			code: `x := 1
r := 0
switch x {
case 1:
	r += 1
	fallthrough
case 2:
	r += 2
	fallthrough
case 3:
	r += 4
}
r`,
			expect: int64(7),
		},
		{
			name: "fallthrough_to_default",
			code: `x := 2
r := 0
switch x {
case 2:
	r = 10
	fallthrough
default:
	r += 5
}
r`,
			expect: int64(15),
		},
		{
			name: "fallthrough_skips_condition",
			code: `x := 1
r := 0
switch x {
case 1:
	r = 100
	fallthrough
case 99:
	r += 1
}
r`,
			expect: int64(101),
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
