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

package interp_provider_piko

import (
	"context"

	"piko.sh/piko/internal/interp/interp_domain"
)

// FileSetInterpreterPort is the optional interface an interpreter implements to support
// compile-once, run-named-entrypoint style invocation of multi-package Go programs.
//
// Distinct from BatchInterpreterPort.CompileAndExecute (which compiles and runs only init
// functions) by exposing the intermediate CompiledFileSet to the caller so a specific
// function can be invoked, repeated invocations are amortised, or only the named
// entrypoint runs.
type FileSetInterpreterPort interface {
	InterpreterPort

	// CompileProgram compiles a multi-package Go program rooted at modulePath.
	//
	// The packages map is keyed by relative package path, each value a filename-to-source
	// map. The compiled program is then used for subsequent ExecuteEntrypoint calls.
	//
	// Takes modulePath (string) which is the module root the program is compiled under.
	// Takes packages (map[string]map[string]string) which maps each package path to its
	// filename-to-source map.
	//
	// Returns *interp_domain.CompiledFileSet which is the compiled program.
	// Returns error when compilation fails.
	CompileProgram(ctx context.Context, modulePath string, packages map[string]map[string]string) (*interp_domain.CompiledFileSet, error)

	// ExecuteEntrypoint runs the named function in cfs and returns its result.
	//
	// Variable initialisers and init functions run before the entrypoint, in package-init
	// order.
	//
	// Takes cfs (*interp_domain.CompiledFileSet) which is the compiled program to run.
	// Takes entrypoint (string) which names the function to invoke.
	//
	// Returns any which is the value the entrypoint returned.
	// Returns error when the entrypoint is missing or fails.
	ExecuteEntrypoint(ctx context.Context, cfs *interp_domain.CompiledFileSet, entrypoint string) (any, error)
}

// CompiledFileSet is the public alias for the compiled multi-package program type.
// Re-exported so external callers can declare fields of it without importing the internal
// interp_domain package directly.
type CompiledFileSet = interp_domain.CompiledFileSet
