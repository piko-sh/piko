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

package interp_provider_piko

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/wasm/wasm_domain"
)

// WASMInterpreterFactory creates Piko bytecode interpreter instances
// for WASM. It implements wasm_domain.InterpreterFactoryPort.
type WASMInterpreterFactory struct{}

var _ wasm_domain.InterpreterFactoryPort = (*WASMInterpreterFactory)(nil)

// NewWASMInterpreterFactory creates a new WASM interpreter factory.
//
// Returns *WASMInterpreterFactory which creates Piko bytecode
// interpreter services.
func NewWASMInterpreterFactory() *WASMInterpreterFactory {
	return &WASMInterpreterFactory{}
}

// NewInterpreter creates a new Piko bytecode interpreter service
// wrapped in a wasmServiceWrapper that supports batch compilation.
//
// Returns any which is a *wasmServiceWrapper wrapping a
// *interp_domain.Service configured for WASM use.
func (*WASMInterpreterFactory) NewInterpreter() any {
	return &wasmServiceWrapper{
		Service: interp_domain.NewService(),
	}
}

// wasmServiceWrapper extends *interp_domain.Service with a
// CompileAndExecuteWASM method that accepts the source layout used
// by the WASM interpreter adapter.
type wasmServiceWrapper struct {
	*interp_domain.Service
}

// GetService returns the underlying interpreter service. This is used
// by WASMSymbolAdapter.Use to load symbols into the service when the
// type assertion to *interp_domain.Service fails due to the wrapper.
//
// Returns *interp_domain.Service which is the wrapped service.
func (w *wasmServiceWrapper) GetService() *interp_domain.Service {
	return w.Service
}

// CompileAndExecuteWASM compiles all packages as a single program and
// executes their init functions. This is the primary compilation path
// for the Piko bytecode interpreter in WASM.
//
// The init functions typically call templater_domain.RegisterASTFunc
// to register template builders in the global FunctionRegistry.
//
// Takes ctx (context.Context) for cancellation and deadlines.
// Takes mainCode (string) which is the main generated Go source code.
// Takes packagePath (string) which is the import path for the main
// package.
// Takes dependencies (map[string]string) which maps dependency import
// paths to their generated source code.
//
// Returns error when compilation or init execution fails.
func (w *wasmServiceWrapper) CompileAndExecuteWASM(ctx context.Context, mainCode, packagePath string, dependencies map[string]string) error {
	modulePath := extractModulePath(packagePath)
	modulePrefix := modulePath + "/"

	packages := make(map[string]map[string]string, len(dependencies)+1)

	packages[strings.TrimPrefix(packagePath, modulePrefix)] = map[string]string{"main.go": mainCode}

	for depPath, depCode := range dependencies {
		packages[strings.TrimPrefix(depPath, modulePrefix)] = map[string]string{"generated.go": depCode}
	}

	cfs, err := w.Service.CompileProgram(ctx, modulePath, packages)
	if err != nil {
		return fmt.Errorf("compiling program: %w", err)
	}

	if err := w.Service.ExecuteInits(ctx, cfs); err != nil {
		return fmt.Errorf("executing init functions: %w", err)
	}

	return nil
}

// extractModulePath extracts the module path from a full package path.
//
// For paths like "playground/internal/pages/home", it returns
// "playground". For paths without a slash, it returns the path
// as-is.
//
// Takes packagePath (string) which is the full import path.
//
// Returns string which is the module root portion of the path.
func extractModulePath(packagePath string) string {
	if idx := strings.Index(packagePath, "/"); idx >= 0 {
		return packagePath[:idx]
	}
	return packagePath
}
