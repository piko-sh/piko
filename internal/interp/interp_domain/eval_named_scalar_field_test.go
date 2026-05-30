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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestF1NamedScalarStructFieldFastPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect           any
		name             string
		source           string
		entrypoint       string
		fastPathOpName   string
		slowPathOpName   string
		expectFastPath   bool
		expectSlowGetOp  bool
		shouldHaveFastOp bool
	}{
		{
			name: "named_int_field_no_methods",
			source: `package main

type Counter int

type Box struct {
	c Counter
}

func run() int64 {
	b := &Box{c: Counter(42)}
	return int64(b.c)
}

func main() {}
`,
			entrypoint:       "run",
			expect:           int64(42),
			fastPathOpName:   "GET_STRUCT_FIELD_INT_T0",
			shouldHaveFastOp: true,
		},
		{
			name: "named_bool_field_no_methods",
			source: `package main

type Flag bool

type Box struct {
	f Flag
}

func run() bool {
	b := &Box{f: Flag(true)}
	return bool(b.f)
}

func main() {}
`,
			entrypoint:       "run",
			expect:           true,
			fastPathOpName:   "GET_STRUCT_FIELD_BOOL_T0",
			shouldHaveFastOp: true,
		},
		{
			name: "named_uint_field_no_methods",
			source: `package main

type Tag uint64

type Box struct {
	t Tag
}

func run() uint64 {
	b := &Box{t: Tag(99)}
	return uint64(b.t)
}

func main() {}
`,
			entrypoint:       "run",
			expect:           uint64(99),
			fastPathOpName:   "GET_STRUCT_FIELD_UINT_T0",
			shouldHaveFastOp: true,
		},
		{
			name: "named_int_field_compound_assign",
			source: `package main

type Counter int

type Box struct {
	c Counter
}

func run() int64 {
	b := &Box{c: Counter(10)}
	b.c++
	b.c += Counter(5)
	return int64(b.c)
}

func main() {}
`,
			entrypoint:     "run",
			expect:         int64(16),
			fastPathOpName: "GET_STRUCT_FIELD_INT_T0",
		},
		{
			name: "named_string_with_methods_stays_slow",
			source: `package main

type ID string

func (i ID) Length() int { return len(string(i)) }

type Box struct {
	id ID
}

func run() int64 {
	b := &Box{id: ID("hello")}
	return int64(b.id.Length())
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(5),

			slowPathOpName: "GET_FIELD",
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

			if tt.fastPathOpName != "" {
				dump := compiled.DisassembleAssembly()
				if !strings.Contains(dump, tt.fastPathOpName) {
					t.Errorf("expected %s in bytecode for %s\nbytecode:\n%s", tt.fastPathOpName, tt.name, dump)
				}
			}
			if tt.slowPathOpName != "" {
				dump := compiled.DisassembleAssembly()
				if !strings.Contains(dump, tt.slowPathOpName) {
					t.Errorf("expected %s in bytecode for %s\nbytecode:\n%s", tt.slowPathOpName, tt.name, dump)
				}
			}

			result, err := service.ExecuteEntrypoint(context.Background(), compiled, tt.entrypoint)
			require.NoError(t, err, "execute failure for %s", tt.name)
			require.Equal(t, tt.expect, result, "expected %v, got %v for %s", tt.expect, result, tt.name)
		})
	}
}
