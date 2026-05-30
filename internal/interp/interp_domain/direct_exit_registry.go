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
	"piko.sh/piko/internal/interp/interp_domain/asm"
)

// directExitRegistrations enumerates every per-op direct-exit stub emitted into
// asm_vm_dispatch_direct_exits_*.s.
//
// Each entry feeds asmgen at code-generation time (the asm package owns the Plan-9 ASM
// body via DirectExitHandlerSpec emission, see asm/handlers_direct_exit.go). The matching
// Go forward declaration and asmJumpTable install entry live in
// vm_dispatch_direct_exits.go and are not derived from this list: those reference the
// handlerXxx Go function names directly so reflect.ValueOf().Pointer() resolves at link
// time.
//
// Returns []asm.DirectExitHandlerSpec, the canonical list.
func directExitRegistrations() []asm.DirectExitHandlerSpec {
	return []asm.DirectExitHandlerSpec{
		{Name: "handlerSetFieldExit", ExitReason: exitSetField},
		{Name: "handlerGetFieldExit", ExitReason: exitGetField},
		{Name: "handlerMapIndexExit", ExitReason: exitMapIndex},
		{Name: "handlerAppendExit", ExitReason: exitAppend},
		{Name: "handlerAppendByteFastExit", ExitReason: exitAppendByteFast},
	}
}

func init() { //nolint:gochecknoinits // one-shot registry registration; mirrors handler-table init pattern.
	for _, spec := range directExitRegistrations() {
		asm.RegisterDirectExit(spec)
	}
}
