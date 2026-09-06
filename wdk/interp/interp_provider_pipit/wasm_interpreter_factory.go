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

//go:build js && wasm

package interp_provider_pipit

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"piko.sh/piko/internal/wasm/wasm_domain"
	"pipit.sh/pipit"
)

// WASMInterpreterFactory creates pipit interpreters for the browser playground.
//
// A golden interpreter carrying the WASM-safe symbol tables (see wasmSymbolProviders) is
// built on first use, so the WASM module is ready before any symbol work happens, and
// every request gets a clone of it. Cloning costs far less than rebuilding the symbol
// registry per request.
type WASMInterpreterFactory struct {
	// golden returns the golden interpreter, building it on first use.
	golden func() *pipit.Interpreter
}

// wasmProgramInterpreter compiles and runs one generated program in a pipit interpreter.
type wasmProgramInterpreter struct {
	// interpreter is the pipit interpreter the program runs in.
	interpreter *pipit.Interpreter
}

var (
	_ wasm_domain.InterpreterFactoryPort = (*WASMInterpreterFactory)(nil)

	_ wasm_domain.ProgramInterpreterPort = (*wasmProgramInterpreter)(nil)
)

// NewWASMInterpreterFactory creates a new WASM interpreter factory.
//
// Returns *WASMInterpreterFactory which creates pipit interpreters with the WASM-safe
// symbol tables loaded.
func NewWASMInterpreterFactory() *WASMInterpreterFactory {
	return &WASMInterpreterFactory{
		golden: sync.OnceValue(func() *pipit.Interpreter {
			return pipit.NewInterpreter(pipit.WithSymbolProvider(wasmSymbolProviders()...))
		}),
	}
}

// NewInterpreter returns an interpreter for one program, cloned from the golden
// interpreter.
//
// Clones share the golden interpreter's symbol registry, so a package compiled by one
// request stays registered for the next. Every request supplies all of its packages as
// source, so each is recompiled, and the caller clears the page's template builder before
// running so a builder left by an earlier request is never served.
//
// Returns wasm_domain.ProgramInterpreterPort which compiles and runs one program.
func (f *WASMInterpreterFactory) NewInterpreter() wasm_domain.ProgramInterpreterPort {
	return &wasmProgramInterpreter{interpreter: f.golden().Clone()}
}

// CompileAndExecute compiles the main package and its dependencies as one program and
// runs their init functions, which register the template builders. A panic inside the
// interpreter is recovered and returned as an error.
//
// Takes mainCode (string) which is the generated Go source of the main package.
// Takes packagePath (string) which is the import path of the main package.
// Takes dependencies (map[string]string) which maps each dependency's import path to its
// generated source.
//
// Returns error when compilation or init execution fails, or the interpreter panics.
func (w *wasmProgramInterpreter) CompileAndExecute(ctx context.Context, mainCode, packagePath string, dependencies map[string]string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = interpreterPanicError(ctx, "compiling and initialising a playground program", recovered)
		}
	}()

	modulePath := extractModulePath(packagePath)
	modulePrefix := modulePath + "/"

	packages := make(map[string]map[string]string, len(dependencies)+1)

	packages[strings.TrimPrefix(packagePath, modulePrefix)] = map[string]string{"main.go": mainCode}

	for dependencyPath, dependencyCode := range dependencies {
		packages[strings.TrimPrefix(dependencyPath, modulePrefix)] = map[string]string{"generated.go": dependencyCode}
	}

	compiledFileSet, err := w.interpreter.CompileProgram(ctx, modulePath, packages)
	if err != nil {
		return fmt.Errorf("interpreting module %q: %w", modulePath, err)
	}

	if err := w.interpreter.ExecuteInits(ctx, compiledFileSet); err != nil {
		return fmt.Errorf("initialising module %q: %w", modulePath, err)
	}

	return nil
}

// extractModulePath extracts the module path from a full package path.
//
// For paths like "playground/internal/pages/home", it returns "playground". For paths
// without a slash, it returns the path as-is.
//
// Takes packagePath (string) which is the full import path.
//
// Returns string which is the module root portion of the path.
func extractModulePath(packagePath string) string {
	if module, _, ok := strings.Cut(packagePath, "/"); ok {
		return module
	}
	return packagePath
}
