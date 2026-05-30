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
	"strings"
	"testing"
)

func buildOptimiseBenchFunction() *CompiledFunction {
	builder := newBytecodeBuilder()
	builder.intRegisters(8).returnInt()
	for i := range 16 {
		builder.emit(opAddInt, uint8(i%4), uint8((i+1)%4), uint8((i+2)%4))
	}
	for i := range 8 {
		builder.emit(opMulInt, uint8(i%4), uint8((i+1)%4), uint8((i+2)%4))
	}
	builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	return builder.build()
}

func BenchmarkOptimisePipeline(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		function := buildOptimiseBenchFunction()
		b.StartTimer()
		if err := function.optimise(ctx); err != nil {
			b.Fatalf("optimise: %v", err)
		}
	}
}

func buildInlinerBenchFunction() *CompiledFunction {
	callee := makeDoubleCallee()
	return &CompiledFunction{
		name: "caller",
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2Return),
				1,
			),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments:    []varLocation{{register: 5, kind: registerInt}},
				returns:      []varLocation{{register: 6, kind: registerInt}},
			},
		},
		resultKinds:  []registerKind{registerInt},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 7},
		functions:    []*CompiledFunction{callee},
	}
}

func BenchmarkInliner(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		function := buildInlinerBenchFunction()
		b.StartTimer()
		if err := runBytecodeInliner(ctx, function); err != nil {
			b.Fatalf("inliner: %v", err)
		}
	}
}

func BenchmarkEscapeAnalysis(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		function := buildOptimiseBenchFunction()
		b.StartTimer()
		if err := runEscapeAnalysisPass(ctx, function); err != nil {
			b.Fatalf("escape analysis: %v", err)
		}
	}
}

func BenchmarkGCMarkCompact(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		vm := &VM{
			globals:   &globalStore{},
			arena:     newRegisterArena(),
			callStack: []callFrame{},
		}
		arena := vm.arena
		liveBuffer := arena.AllocStringBytes(128)
		copy(liveBuffer, strings.Repeat("x", 128))
		vm.globals.strings = []string{string(liveBuffer)}
		for range 4 {
			arena.growByteSlab(initialByteSlabSize)
		}
		liveInts := arena.AllocIntBacking(32)
		for i := range liveInts {
			liveInts[i] = int64(i)
		}
		for range 2 {
			arena.growIntBackingSlab(initialIntBackingSize)
		}
		vm.callStack = []callFrame{{
			registers: Registers{
				slicesInt: [][]int64{liveInts},
			},
		}}
		vm.framePointer = 0
		b.StartTimer()
		arena.MinorGC(vm)
	}
}
