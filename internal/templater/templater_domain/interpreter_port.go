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

package templater_domain

import (
	"context"
	"reflect"
)

// SymbolExports maps package paths to symbol names and their reflected values.
// Acts as a type alias that lets the public API accept symbol exports without
// depending on a specific interpreter package.
type SymbolExports = map[string]map[string]reflect.Value

// InterpreterPort abstracts the Go interpreter used for JIT compilation.
// Allows the templater domain to remain decoupled from the concrete
// interpreter implementation, enabling the interpreter to be an optional
// dependency.
type InterpreterPort interface {
	// Eval evaluates Go source code and returns the result.
	//
	// Takes ctx (context.Context) for cancellation and deadlines.
	// Takes code (string) which contains the Go source code to evaluate.
	//
	// Returns any which is the result of evaluating the code.
	// Returns error when evaluation fails.
	Eval(ctx context.Context, code string) (any, error)

	// SetBuildContext sets the build context for import resolution.
	// The buildCtx parameter should be a go/build.Context.
	//
	// Takes buildCtx (any) which provides the build context settings.
	SetBuildContext(buildCtx any)

	// SetSourcecodeFilesystem sets the virtual filesystem for source files.
	// This is used for resolving imports from in-memory generated code.
	//
	// Takes fs (any) which provides the virtual filesystem implementation.
	SetSourcecodeFilesystem(fs any)

	// RegisterPackageAlias creates an alias for a package path, allowing
	// the interpreter to resolve imports using the alias.
	//
	// Takes canonical (string) which is the full package path.
	// Takes alias (string) which is the short alias to register.
	//
	// Returns error when alias registration fails.
	RegisterPackageAlias(canonical, alias string) error

	// Reset clears the interpreter state so it can be used again.
	// Call this before returning the interpreter to a pool.
	Reset()

	// Clone creates a copy of the interpreter with loaded symbols.
	// The cloned interpreter shares the symbol table but has independent
	// execution state.
	//
	// Returns InterpreterPort which is the cloned interpreter.
	Clone() InterpreterPort
}

// InterpreterPoolPort provides pooled interpreters for efficient reuse.
// Pre-warming interpreters with symbols is expensive, so pooling amortises
// this cost across multiple compilations.
type InterpreterPoolPort interface {
	// Get retrieves an interpreter from the pool.
	// The returned interpreter is ready for use with symbols pre-loaded.
	//
	// Returns InterpreterPort which is a ready-to-use interpreter.
	// Returns error when the pool is exhausted or an interpreter cannot
	// be created.
	Get() (InterpreterPort, error)

	// Put returns an interpreter to the pool after resetting it.
	// The interpreter's state is cleared before being returned to the pool.
	//
	// Takes i (InterpreterPort) which is the interpreter to return.
	Put(i InterpreterPort)
}

// BatchInterpreterPort extends InterpreterPort with batch compilation.
//
// Interpreters that compile all packages at once implement this to
// bypass the incremental Eval() loop. The internal Piko bytecode
// interpreter uses this path for efficient multi-package
// compilation.
type BatchInterpreterPort interface {
	InterpreterPort

	// CompileAndExecute compiles all packages as a single program and
	// executes their init functions, which register into the global
	// FunctionRegistry.
	//
	// Takes ctx (context.Context) for cancellation and deadlines.
	// Takes modulePath (string) which identifies the module
	// (e.g. "myproject/dist").
	// Takes packages (map[string]map[string]string) which maps relative
	// package paths to filename-to-source maps.
	//
	// Returns error when compilation or init execution fails.
	CompileAndExecute(ctx context.Context, modulePath string, packages map[string]map[string]string) error

	// HasRegisteredPackage reports whether the given import path is
	// available in the symbol registry. Packages that are registered
	// do not need to be compiled from source.
	//
	// Takes importPath (string) which is the full Go import path to
	// check.
	//
	// Returns true if the package is already available via the symbol
	// registry.
	HasRegisteredPackage(importPath string) bool
}

// InterpreterProviderPort is the top-level interface for interpreter providers.
// It combines symbol management with interpreter pool creation.
//
// Implementations are provided by optional modules such as
// piko.sh/piko/wdk/interp/interp_provider_piko.
type InterpreterProviderPort interface {
	// NewSymbolProvider creates a symbol provider with stdlib symbols loaded.
	// The symbol provider can be used to register additional symbols before
	// creating an interpreter pool.
	//
	// Returns SymbolProviderPort which is ready for symbol registration.
	NewSymbolProvider() SymbolProviderPort

	// NewInterpreterPool creates a pool of pre-warmed interpreters.
	// The golden interpreter is pre-loaded with the provided symbols, and
	// each interpreter retrieved from the pool is a clone of the golden.
	//
	// Takes symbols (SymbolProviderPort) which provides the symbols to
	// pre-load into the golden interpreter.
	//
	// Returns InterpreterPoolPort which provides pooled interpreters.
	NewInterpreterPool(symbols SymbolProviderPort) InterpreterPoolPort

	// RegisterSymbols adds additional symbol exports to the provider.
	// These symbols will be included when NewSymbolProvider is called.
	//
	// Takes exports (SymbolExports) which contains the additional symbols
	// to register.
	RegisterSymbols(exports SymbolExports)
}
