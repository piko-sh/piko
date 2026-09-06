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

// SymbolExports maps package paths to symbol names and their reflected values. Acts as a
// type alias that lets the public API accept symbol exports without depending on a
// specific interpreter package.
type SymbolExports = map[string]map[string]reflect.Value

// InterpreterPort abstracts the Go interpreter that runs generated component code in
// interpreted mode (dev-i). It keeps the templater domain decoupled from the concrete
// interpreter, so the interpreter stays an optional dependency.
//
// The interpreter compiles every generated package as one program and runs its init
// functions, which register the template builders into the global FunctionRegistry.
type InterpreterPort interface {
	// CompileAndExecute compiles all packages as a single program and executes their init
	// functions, which register into the global FunctionRegistry.
	//
	// Takes ctx (context.Context) for cancellation and deadlines.
	// Takes modulePath (string) which identifies the module (e.g. "myproject/dist").
	// Takes packages (map[string]map[string]string) which maps relative package paths to
	// filename-to-source maps.
	//
	// Returns error when compilation or init execution fails.
	CompileAndExecute(ctx context.Context, modulePath string, packages map[string]map[string]string) error

	// HasRegisteredPackage reports whether the given import path is available in the symbol
	// registry. Packages that are registered do not need to be compiled from source.
	//
	// Takes importPath (string) which is the full Go import path to check.
	//
	// Returns true if the package is already available via the symbol registry.
	HasRegisteredPackage(importPath string) bool
}

// InterpreterPoolPort hands out interpreters that share one pre-warmed symbol registry.
// Loading symbols is expensive, so the pool builds it once and every interpreter it
// returns reuses it.
type InterpreterPoolPort interface {
	// LoadModules loads the module bundles queued on the provider into the pool's shared
	// symbol registry. It is called once, at startup, before the first Get; later calls
	// return the first call's result.
	//
	// Returns error when a module fails to load or ctx is cancelled.
	LoadModules(ctx context.Context) error

	// Get returns an interpreter ready for use with symbols pre-loaded.
	//
	// Returns InterpreterPort which is a ready-to-use interpreter.
	// Returns error when an interpreter cannot be created, including when queued modules
	// have not been loaded or failed to load.
	Get() (InterpreterPort, error)
}

// InterpreterProviderPort is the top-level interface for interpreter providers. It
// combines symbol registration with interpreter pool creation.
//
// Implementations are provided by optional modules such as
// piko.sh/piko/wdk/interp/interp_provider_pipit.
type InterpreterProviderPort interface {
	// RegisterSymbols adds additional symbol exports to the provider. These symbols are
	// loaded into the interpreters created by NewInterpreterPool.
	//
	// Takes exports (SymbolExports) which contains the additional symbols to register.
	RegisterSymbols(exports SymbolExports)

	// NewInterpreterPool creates a pool of pre-warmed interpreters.
	//
	// The golden interpreter is loaded once with the standard library, the piko runtime
	// symbols and any symbols added through RegisterSymbols. Each call returns an
	// independent pool with a fresh symbol registry; the caller loads queued modules into it
	// with LoadModules.
	//
	// Returns InterpreterPoolPort which provides the interpreters.
	NewInterpreterPool() InterpreterPoolPort
}
