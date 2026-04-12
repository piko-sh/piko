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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

import (
	"context"
	"testing"
)

const (
	dispatchBenchInstructionCount = 1000
)

func BenchmarkDispatchOpNopMainTier(b *testing.B) {
	builder := newBytecodeBuilder()
	builder.intRegisters(1).returnInt()
	for range dispatchBenchInstructionCount {
		builder.emit(opNop, 0, 0, 0)
	}
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := service.Execute(ctx, compiled)
		if err != nil {
			b.Fatalf("execute: %v", err)
		}
	}
}

func BenchmarkDispatchOpNopTier3Form(b *testing.B) {
	builder := newBytecodeBuilder()
	builder.intRegisters(1).returnInt()
	for range dispatchBenchInstructionCount {
		builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), uint8(subOpTier3Nop))
	}
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := service.Execute(ctx, compiled)
		if err != nil {
			b.Fatalf("execute: %v", err)
		}
	}
}

func BenchmarkDispatchTier0AddInt(b *testing.B) {
	builder := newBytecodeBuilder()
	builder.intRegisters(3).returnInt()
	builder.emit(opDrillTier1, uint8(subOpLoadIntConstSmall), 1, 1)
	builder.emit(opDrillTier1, uint8(subOpLoadIntConstSmall), 2, 1)
	for range dispatchBenchInstructionCount {
		builder.emit(opAddInt, 1, 1, 2)
	}
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	compiled := builder.build()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := service.Execute(ctx, compiled)
		if err != nil {
			b.Fatalf("execute: %v", err)
		}
	}
}
