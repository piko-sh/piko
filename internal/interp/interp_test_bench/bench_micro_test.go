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
	"reflect"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func BenchmarkMicro(b *testing.B) {
	b.Run("arena_alloc_restore_single", func(b *testing.B) {
		numRegs := [interp_domain.NumRegisterKinds]uint32{4, 2, 2, 4, 2, 2, 0}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			arena := interp_domain.GetRegisterArena()
			sp := arena.Save()
			_ = arena.AllocRegisters(numRegs)
			arena.Restore(sp)
			interp_domain.PutRegisterArena(arena)
		}
	})

	b.Run("arena_alloc_restore_10_frames", func(b *testing.B) {
		numRegs := [interp_domain.NumRegisterKinds]uint32{4, 2, 2, 4, 2, 2, 0}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			arena := interp_domain.GetRegisterArena()
			saves := [10]interp_domain.ArenaSavePoint{}
			for i := range 10 {
				saves[i] = arena.Save()
				_ = arena.AllocRegisters(numRegs)
			}
			for i := 9; i >= 0; i-- {
				arena.Restore(saves[i])
			}
			interp_domain.PutRegisterArena(arena)
		}
	})

	b.Run("reflect_valueof_int", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reflect.ValueOf(int64(42))
		}
	})

	b.Run("reflect_valueof_string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reflect.ValueOf("hello")
		}
	})

	b.Run("reflect_value_int_extract", func(b *testing.B) {
		v := reflect.ValueOf(int64(42))
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.Int()
		}
	})

	b.Run("reflect_value_string_extract", func(b *testing.B) {
		v := reflect.ValueOf("hello")
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.String()
		}
	})

	b.Run("reflect_value_interface", func(b *testing.B) {
		v := reflect.ValueOf(int64(42))
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.Interface()
		}
	})

	b.Run("type_assert_func", func(b *testing.B) {
		var untypedFunction any = func(s string) string { return s }
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			f, ok := untypedFunction.(func(string) string)
			_, _ = f, ok
		}
	})

	b.Run("int_array_copy_2", func(b *testing.B) {
		src := [8]int64{1, 2, 3, 4, 5, 6, 7, 8}
		dst := [8]int64{}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			dst[0] = src[0]
			dst[1] = src[1]
		}
		_ = dst
	})

	b.Run("int_array_copy_4", func(b *testing.B) {
		src := [8]int64{1, 2, 3, 4, 5, 6, 7, 8}
		dst := [8]int64{}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			dst[0] = src[0]
			dst[1] = src[1]
			dst[2] = src[2]
			dst[3] = src[3]
		}
		_ = dst
	})
}
