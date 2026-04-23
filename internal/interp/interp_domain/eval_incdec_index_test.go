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

func TestIncDecOnIndexExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "map_value_inc",
			source: `package main

func run() int {
	counts := map[string]int{"a": 0}
	counts["a"]++
	counts["a"]++
	counts["b"]++
	return counts["a"] + counts["b"]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
		},
		{
			name: "map_value_dec",
			source: `package main

func run() int {
	counts := map[string]int{"x": 5}
	counts["x"]--
	counts["x"]--
	return counts["x"]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
		},
		{
			name: "slice_element_inc",
			source: `package main

func run() int {
	values := []int{10, 20, 30}
	values[1]++
	values[2]++
	values[2]++
	return values[0] + values[1] + values[2]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(63),
		},
		{
			name: "map_missing_key_inc_creates_entry",
			source: `package main

func run() int {
	counts := map[string]int{}
	counts["fresh"]++
	counts["fresh"]++
	counts["fresh"]++
	return counts["fresh"]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err, "source:\n%s", tt.source)
			require.Equal(t, tt.expect, result)
		})
	}
}
