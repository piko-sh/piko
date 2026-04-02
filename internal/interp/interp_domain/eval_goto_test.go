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

func TestEvalGotoAndLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "forward_goto",
			code: `func f() int {
	x := 1
	goto skip
	x = 99
	skip:
	return x
}
f()`,
			expect: int64(1),
		},
		{
			name: "backward_goto_loop",
			code: `func f() int {
	x := 0
	again:
	x++
	if x < 5 {
		goto again
	}
	return x
}
f()`,
			expect: int64(5),
		},
		{
			name: "labelled_break",
			code: `func count() int {
	sum := 0
	outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if j == 2 {
				break outer
			}
			sum++
		}
	}
	return sum
}
count()`,
			expect: int64(2),
		},
		{
			name: "labelled_continue",
			code: `func count() int {
	sum := 0
	outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer
			}
			sum++
		}
	}
	return sum
}
count()`,
			expect: int64(3),
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
