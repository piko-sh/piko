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

// dispatch_amd64.h -- amd64-specific macros for the threaded dispatch loop.
// Included by all vm_dispatch_*_amd64.s files.

// Register allocation (preserved across all handlers):
//   R15  - &DispatchContext
//   R14  - pc (instruction index)
//   R13  - codeLen
//   R12  - codeBase (pointer to Body[0])
//   R8   - intsBase (pointer to regs.Ints[0])
//   R9   - floatsBase (pointer to regs.Floats[0])
//   R10  - jumpTable base pointer
//   R11  - intConstsBase (pointer to IntConstants[0])
//
// Scratch (per-handler): DX, AX, BX, CX, SI, DI, X0, X1

// DISPATCH_NEXT is the threaded dispatch tail inlined at the end of each
// handler. Each handler's indirect JMP is a separate branch prediction
// site, achieving ~50% accuracy vs ~10% for a single switch statement.
//
// The label "dn" is function-local in Go's Plan 9 assembler, so the
// same name can appear in multiple TEXT blocks without conflict.
#define DISPATCH_NEXT() \
	CMPQ    R14, R13;                   \
	JL      dn;                         \
	MOVQ    R14, 16(R15);               \
	MOVQ    $EXIT_END_OF_CODE, 96(R15); \
	MOVQ    R14, 104(R15);              \
	RET;                                \
	dn:                                 \
	MOVL    (R12)(R14*4), DX;           \
	INCQ    R14;                        \
	MOVBLZX DL, CX;                     \
	JMP     (R10)(CX*8)

// DIV_BY_ZERO_EXIT stores the error and returns to Go.
#define DIV_BY_ZERO_EXIT() \
	MOVQ R14, 16(R15);               \
	MOVQ $EXIT_DIV_BY_ZERO, 96(R15); \
	MOVQ R14, 104(R15);              \
	RET
