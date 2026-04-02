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

// dispatch_arm64.h -- arm64-specific macros for the threaded dispatch loop.
// Included by all vm_dispatch_*_arm64.s files.

// Register allocation (preserved across all handlers):
//   R19  - &DispatchContext
//   R20  - pc (instruction index)
//   R21  - codeLen
//   R22  - codeBase (pointer to Body[0])
//   R23  - intsBase (pointer to regs.Ints[0])
//   R24  - floatsBase (pointer to regs.Floats[0])
//   R25  - jumpTable base pointer
//   R26  - intConstsBase (pointer to IntConstants[0])
//
// Scratch (per-handler): R0-R15, F0-F3
// Avoid: R28 (Go g register), R29 (FP), R30 (LR), R16-R18 (platform)

// DISPATCH_NEXT is the threaded dispatch tail inlined at the end of each
// handler. Each handler's indirect jump via the jump table is a separate
// branch prediction site, achieving ~50% accuracy vs ~10% for a single
// switch statement.
//
// The label "dn" is function-local in Go's Plan 9 assembler, so the
// same name can appear in multiple TEXT blocks without conflict.
#define DISPATCH_NEXT() \
	CMP   R21, R20;              \
	BLT   dn;                    \
	MOVD  R20, 16(R19);          \
	MOVD  $EXIT_END_OF_CODE, R0; \
	MOVD  R0, 96(R19);           \
	MOVD  R20, 104(R19);         \
	RET;                         \
	dn:                          \
	MOVWU (R22)(R20<<2), R0;     \
	ADD   $1, R20, R20;          \
	AND   $0xFF, R0, R1;         \
	MOVD  (R25)(R1<<3), R2;      \
	JMP   (R2)

// DIV_BY_ZERO_EXIT stores the error and returns to Go.
#define DIV_BY_ZERO_EXIT() \
	MOVD R20, 16(R19);          \
	MOVD $EXIT_DIV_BY_ZERO, R0; \
	MOVD R0, 96(R19);           \
	MOVD R20, 104(R19);         \
	RET
