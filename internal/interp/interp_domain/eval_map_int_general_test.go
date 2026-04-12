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

package interp_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalMapIntGeneralTypedFastPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect     any
		name       string
		source     string
		entrypoint string
	}{
		{
			name: "map int to pointer struct (lookup hit)",
			source: `package main

type node struct {
	value int64
}

func run() int64 {
	m := map[int64]*node{}
	m[42] = &node{value: 7}
	if found, ok := m[42]; ok {
		return found.value
	}
	return -1
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(7),
		},
		{
			name: "map int to pointer struct (lookup miss)",
			source: `package main

type node struct {
	value int64
}

func run() int64 {
	m := map[int64]*node{}
	m[42] = &node{value: 7}
	if _, ok := m[99]; !ok {
		return 100
	}
	return -1
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(100),
		},
		{
			name: "map int to interface (any)",
			source: `package main

func run() int64 {
	m := map[int]any{}
	m[1] = int64(10)
	m[2] = int64(20)
	v1 := m[1].(int64)
	v2 := m[2].(int64)
	return v1 + v2
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(30),
		},
		{
			name: "map int64 to interface (any)",
			source: `package main

func run() int64 {
	m := map[int64]any{}
	m[1] = int64(10)
	m[2] = int64(20)
	v1 := m[1].(int64)
	v2 := m[2].(int64)
	return v1 + v2
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(30),
		},
		{
			name: "map int to slice (general value type)",
			source: `package main

func run() int64 {
	m := map[int][]int{}
	m[1] = []int{10, 20, 30}
	s := m[1]
	return int64(s[0] + s[1] + s[2])
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(60),
		},
		{
			name: "map int to pointer struct reused across iterations",
			source: `package main

type node struct {
	value int64
}

func run() int64 {
	m := map[int64]*node{}
	var sum int64
	for i := int64(0); i < 100; i++ {
		m[i] = &node{value: i * 2}
	}
	for i := int64(0); i < 100; i++ {
		if found, ok := m[i]; ok {
			sum += found.value
		}
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(9900),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled, err := service.CompileFileSet(context.Background(), map[string]string{
				"main.go": tt.source,
			})
			require.NoError(t, err, "compile failure for %s", tt.name)
			result, err := service.ExecuteEntrypoint(context.Background(), compiled, tt.entrypoint)
			require.NoError(t, err, "execute failure for %s", tt.name)
			require.Equal(t, tt.expect, result, "expected %v, got %v for %s", tt.expect, result, tt.name)
		})
	}
}
