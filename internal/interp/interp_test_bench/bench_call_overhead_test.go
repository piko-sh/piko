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

//go:build bench

package interp_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func BenchmarkCallOverhead(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"no_args",
			`package main

func noop() int { return 1 }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += noop()
	}
	return s
}
`,
		},
		{
			"one_int_arg",
			`package main

func inc(x int) int { return x + 1 }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += inc(i)
	}
	return s
}
`,
		},
		{
			"two_int_args",
			`package main

func add(a, b int) int { return a + b }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += add(i, i+1)
	}
	return s
}
`,
		},
		{
			"four_int_args",
			`package main

func sum4(a, b, c, d int) int { return a + b + c + d }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += sum4(i, i+1, i+2, i+3)
	}
	return s
}
`,
		},
		{
			"mixed_int_string",
			`package main

func mixed(x int, s string) int { return x + len(s) }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += mixed(i, "hello")
	}
	return s
}
`,
		},
		{
			"mixed_int_string_float",
			`package main

func mixed3(x int, s string, f float64) int {
	return x + len(s) + int(f)
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += mixed3(i, "hello", 3.14)
	}
	return s
}
`,
		},
		{
			"general_arg_slice",
			`package main

func withSlice(s []int) int { return len(s) }

func run() int {
	sl := []int{1, 2, 3}
	s := 0
	for i := 0; i < 1000; i++ {
		s += withSlice(sl)
	}
	return s
}
`,
		},
		{
			"two_returns",
			`package main

func divmod(a, b int) (int, int) { return a / b, a % b }

func run() int {
	s := 0
	for i := 1; i < 1000; i++ {
		q, r := divmod(i*7, 3)
		s += q + r
	}
	return s
}
`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			service := interp_domain.NewService()
			cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": bm.source})
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err = service.ExecuteEntrypoint(context.Background(), cfs, "run")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCallDepthExecOnly(b *testing.B) {
	depths := []struct {
		name   string
		source string
	}{
		{
			"depth_10",
			`package main

func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}

func run() int { return recurse(10) }
`,
		},
		{
			"depth_100",
			`package main

func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}

func run() int { return recurse(100) }
`,
		},
		{
			"depth_1000",
			`package main

func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}

func run() int { return recurse(1000) }
`,
		},
	}

	for _, d := range depths {
		b.Run(d.name, func(b *testing.B) {
			service := interp_domain.NewService()
			cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": d.source})
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err = service.ExecuteEntrypoint(context.Background(), cfs, "run")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
