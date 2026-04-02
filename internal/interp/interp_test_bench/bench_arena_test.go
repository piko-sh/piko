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

func BenchmarkRegisterAllocation(b *testing.B) {
	numRegs := [interp_domain.NumRegisterKinds]uint32{8, 4, 4, 8}

	b.Run("direct_make", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {

			for range 32 {
				_ = interp_domain.NewRegistersForBench(numRegs)
			}
		}
	})

	b.Run("arena", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			arena := interp_domain.GetRegisterArena()
			for range 32 {
				_ = arena.AllocRegisters(numRegs)
			}
			interp_domain.PutRegisterArena(arena)
		}
	})
}

func BenchmarkCallDepth(b *testing.B) {
	depths := []struct {
		name string
		code string
	}{
		{
			"depth_10",
			`func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}
recurse(10)`,
		},
		{
			"depth_100",
			`func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}
recurse(100)`,
		},
		{
			"depth_1000",
			`func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1) + 1
}
recurse(1000)`,
		},
	}

	for _, d := range depths {
		b.Run(d.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				service := interp_domain.NewService()
				_, err := service.Eval(context.Background(), d.code)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
