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

package asm

import "piko.sh/piko/wdk/asmgen"

// frameSizeZero is the TEXT frame size for handlers that use no local
// stack space.
const frameSizeZero = "$0"

// flagNoSplit marks a TEXT symbol as NOSPLIT, meaning it must not grow
// the goroutine stack and requires no stack-bound check.
const flagNoSplit = "NOSPLIT"

// dispatchBuildConstraint is the build constraint prefix shared by
// all dispatch handler files.
const dispatchBuildConstraint = "!safe && !(js && wasm)"

// dispatchOutputDir is the output directory for generated dispatch
// handler files, relative to the project root.
const dispatchOutputDir = "internal/interp/interp_domain"

// dispatchIncludes lists the standard .h includes for dispatch files.
// The architecture-specific include (dispatch_amd64.h / dispatch_arm64.h)
// is resolved by the generator based on the target architecture.
var dispatchIncludes = []string{"textflag.h", "asm_dispatch_offsets.h", "asm_dispatch_amd64.h"}

// FileGroups returns all FileGroup definitions for the interp_domain
// dispatch handlers.
//
// Returns []asmgen.FileGroup[BytecodeArchitecturePort] describing every .s file to generate.
func FileGroups() []asmgen.FileGroup[BytecodeArchitecturePort] {
	return []asmgen.FileGroup[BytecodeArchitecturePort]{
		{
			BaseName:        "asm_vm_dispatch_arith",
			OutputDir:       dispatchOutputDir,
			BuildConstraint: dispatchBuildConstraint,
			HeaderComment:   "Data movement, arithmetic, and bitwise handlers.",
			Includes:        dispatchIncludes,
			Handlers:        arithmeticHandlers(),
		},
		{
			BaseName:        "asm_vm_dispatch_cmp",
			OutputDir:       dispatchOutputDir,
			BuildConstraint: dispatchBuildConstraint,
			HeaderComment:   "Comparison, conversion, math intrinsic, and control flow handlers.",
			Includes:        dispatchIncludes,
			Handlers:        comparisonHandlers(),
		},
		{
			BaseName:              "asm_vm_dispatch_string",
			OutputDir:             dispatchOutputDir,
			BuildConstraint:       dispatchBuildConstraint,
			HeaderCommentFunction: stringHeaderComment,
			Includes:              dispatchIncludes,
			Handlers:              stringHandlers(),
		},
		{
			BaseName:        "asm_vm_dispatch_super",
			OutputDir:       dispatchOutputDir,
			BuildConstraint: dispatchBuildConstraint,
			HeaderComment:   "Fused superinstruction handlers.",
			Includes:        dispatchIncludes,
			Handlers:        superinstructionHandlers(),
		},
		{
			BaseName:        "asm_vm_dispatch_init",
			OutputDir:       dispatchOutputDir,
			BuildConstraint: dispatchBuildConstraint,
			HeaderComment:   "Dispatch loop initialisation, jump table setup, and exit handlers.",
			Includes:        dispatchIncludes,
			Handlers:        initialisationHandlers(),
		},
		{
			BaseName:        "asm_vm_dispatch_inline",
			OutputDir:       dispatchOutputDir,
			BuildConstraint: dispatchBuildConstraint,
			HeaderComment:   "Inline call and return handlers.",
			Includes:        dispatchIncludes,
			Handlers:        inlineCallHandlers(),
		},
	}
}
