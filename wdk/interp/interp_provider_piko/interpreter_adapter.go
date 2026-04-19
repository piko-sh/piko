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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/interp/interp_adapters"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// directoryPermission is the permission mode for directories created
	// during bytecode emission.
	directoryPermission = 0o750

	// filePermission is the permission mode for files written during
	// bytecode emission.
	filePermission = 0o600

	// slogKeyError is the structured-logging key for error values.
	slogKeyError = "error"

	// slogKeyPath is the structured-logging key for file paths.
	slogKeyPath = "path"

	// slogKeyDirectory is the structured-logging key for directory paths.
	slogKeyDirectory = "directory"
)

// interpreterAdapter wraps *interp_domain.Service to implement
// InterpreterPort and BatchInterpreterPort.
type interpreterAdapter struct {
	// service is the underlying bytecode interpreter service.
	service *interp_domain.Service

	// bytecodeEmissionDirectory is the root directory for emitting
	// source and compiled bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string
}

var _ templater_domain.InterpreterPort = (*interpreterAdapter)(nil)
var _ templater_domain.BatchInterpreterPort = (*interpreterAdapter)(nil)

// Eval evaluates Go source code and returns the result.
//
// Takes ctx (context.Context) for cancellation and deadlines.
// Takes code (string) which contains the Go source code to evaluate.
//
// Returns any which is the result of evaluating the code.
// Returns error when evaluation fails.
func (a *interpreterAdapter) Eval(ctx context.Context, code string) (any, error) {
	return a.service.Eval(ctx, code)
}

// SetBuildContext is a no-op for the Piko bytecode interpreter.
// Import resolution is handled through the SymbolRegistry and
// CompileProgram, not through go/build.Context.
//
// Takes buildCtx (any) which is ignored.
func (*interpreterAdapter) SetBuildContext(any) {}

// SetSourcecodeFilesystem is a no-op for the Piko bytecode interpreter.
// Source code is provided directly to CompileProgram rather than read
// from a virtual filesystem.
//
// Takes fs (any) which is ignored.
func (*interpreterAdapter) SetSourcecodeFilesystem(any) {}

// RegisterPackageAlias is a no-op for the Piko bytecode interpreter.
// In the batch compilation model, all packages are compiled together
// and import resolution is handled internally by CompileProgram.
//
// Takes canonical (string) which is the full package path.
// Takes alias (string) which is the short alias.
//
// Returns nil always.
func (*interpreterAdapter) RegisterPackageAlias(string, string) error {
	return nil
}

// Reset clears the interpreter state for reuse.
// This should be called before returning the interpreter to a pool.
func (a *interpreterAdapter) Reset() {
	a.service.Reset()
}

// Clone creates a copy of the interpreter with loaded symbols.
// The cloned interpreter shares the symbol table but has independent
// execution state.
//
// Returns templater_domain.InterpreterPort which is the cloned
// interpreter.
func (a *interpreterAdapter) Clone() templater_domain.InterpreterPort {
	return &interpreterAdapter{
		service: a.service.Clone(),
	}
}

// CompileAndExecute compiles all packages as a single program and
// executes their init functions. This is the primary compilation path
// for the Piko bytecode interpreter.
//
// The init functions typically call templater_domain.RegisterASTFunc to
// register template builders in the global FunctionRegistry.
//
// Takes ctx (context.Context) for cancellation and deadlines.
// Takes modulePath (string) which identifies the module.
// Takes packages (map[string]map[string]string) which maps relative
// package paths to filename-to-source maps.
//
// Returns error when compilation or init execution fails.
func (a *interpreterAdapter) CompileAndExecute(ctx context.Context, modulePath string, packages map[string]map[string]string) error {
	if a.bytecodeEmissionDirectory != "" {
		a.emitSourceFiles(modulePath, packages)
		a.service.SetCompilationSnapshot(func(snapshot *interp_domain.CompiledFileSet) {
			a.emitBytecode(packages, snapshot)
		})
	}

	compiledFileSet, err := a.service.CompileProgram(ctx, modulePath, packages)
	if err != nil {
		return fmt.Errorf("compiling program: %w", err)
	}

	if err := a.service.ExecuteInits(ctx, compiledFileSet); err != nil {
		return fmt.Errorf("executing init functions: %w", err)
	}

	return nil
}

// HasRegisteredPackage reports whether the given import path is
// available in the symbol registry.
//
// Takes importPath (string) which is the full package import path
// to look up.
//
// Returns bool which is true when the package is registered.
func (a *interpreterAdapter) HasRegisteredPackage(importPath string) bool {
	return a.service.HasRegisteredPackage(importPath)
}

// emitSourceFiles writes source files to the emission directory for
// debugging.
//
// Takes modulePath (string) which identifies the module being
// compiled.
// Takes packages (map[string]map[string]string) which maps
// relative package paths to filename-to-source maps.
func (a *interpreterAdapter) emitSourceFiles(modulePath string, packages map[string]map[string]string) {
	sandbox, err := safedisk.NewSandbox(a.bytecodeEmissionDirectory, safedisk.ModeReadWrite)
	if err != nil {
		slog.Warn("bytecode emission: failed to open emission sandbox", slogKeyDirectory, a.bytecodeEmissionDirectory, slogKeyError, err)
		return
	}
	defer func() { _ = sandbox.Close() }()

	for relativePath, files := range packages {
		packageDirectory := filepath.Join("source", sanitisePath(modulePath), sanitisePath(relativePath))
		if err := sandbox.MkdirAll(packageDirectory, directoryPermission); err != nil {
			slog.Warn("bytecode emission: failed to create source directory", slogKeyDirectory, packageDirectory, slogKeyError, err)
			continue
		}

		for filename, source := range files {
			outputPath := filepath.Join(packageDirectory, filename)
			if err := sandbox.WriteFile(outputPath, []byte(source), filePermission); err != nil {
				slog.Warn("bytecode emission: failed to write source file", slogKeyPath, outputPath, slogKeyError, err)
			}
		}
	}
}

// emitBytecode serialises the compiled file set to disk as a
// FlatBuffer binary for post-mortem inspection.
//
// Takes packages (map[string]map[string]string) which maps
// relative package paths to filename-to-source maps.
// Takes compiledFileSet (*interp_domain.CompiledFileSet) which
// is the compiled bytecode to serialise.
func (a *interpreterAdapter) emitBytecode(packages map[string]map[string]string, compiledFileSet *interp_domain.CompiledFileSet) {
	sandbox, err := safedisk.NewSandbox(a.bytecodeEmissionDirectory, safedisk.ModeReadWrite)
	if err != nil {
		slog.Warn("bytecode emission: failed to open emission sandbox", slogKeyDirectory, a.bytecodeEmissionDirectory, slogKeyError, err)
		return
	}
	defer func() { _ = sandbox.Close() }()

	bytecodeDirectory := "compiled"
	if err := sandbox.MkdirAll(bytecodeDirectory, directoryPermission); err != nil {
		slog.Warn("bytecode emission: failed to create bytecode directory", slogKeyDirectory, bytecodeDirectory, slogKeyError, err)
		return
	}

	suffix, sortedPaths := bytecodeFileSuffix(packages)

	data := interp_adapters.PackCompiledFileSetToBytes(compiledFileSet)
	filename := fmt.Sprintf("bytecode-%s.bin", suffix)
	outputPath := filepath.Join(bytecodeDirectory, filename)

	if err := sandbox.WriteFile(outputPath, data, filePermission); err != nil {
		slog.Warn("bytecode emission: failed to write bytecode file", slogKeyPath, outputPath, slogKeyError, err)
	}

	manifestFilename := fmt.Sprintf("bytecode-%s.txt", suffix)
	manifestPath := filepath.Join(bytecodeDirectory, manifestFilename)
	manifest := strings.Join(sortedPaths, "\n") + "\n"
	if err := sandbox.WriteFile(manifestPath, []byte(manifest), filePermission); err != nil {
		slog.Warn("bytecode emission: failed to write manifest file", slogKeyPath, manifestPath, slogKeyError, err)
	}

	asmFilename := fmt.Sprintf("bytecode-%s.pkasm", suffix)
	asmPath := filepath.Join(bytecodeDirectory, asmFilename)
	asmContent := compiledFileSet.DisassembleAssembly()
	if err := sandbox.WriteFile(asmPath, []byte(asmContent), filePermission); err != nil {
		slog.Warn("bytecode emission: failed to write assembly file", slogKeyPath, asmPath, slogKeyError, err)
	}
}

// bytecodeFileSuffix builds a filename suffix from the relative
// package paths in a compilation batch.
//
// Takes packages (map[string]map[string]string) which maps
// relative package paths to filename-to-source maps.
//
// Returns string which is the sanitised path for single-package
// batches or a short hash for multi-package batches.
// Returns []string which is the sorted list of sanitised paths.
func bytecodeFileSuffix(packages map[string]map[string]string) (string, []string) {
	paths := make([]string, 0, len(packages))
	for _, relativePath := range slices.Sorted(maps.Keys(packages)) {
		if relativePath == "" {
			paths = append(paths, "_root")
		} else {
			paths = append(paths, sanitisePath(relativePath))
		}
	}

	if len(paths) == 1 {
		return paths[0], paths
	}

	hash := sha256.Sum256([]byte(strings.Join(paths, "\n")))
	return fmt.Sprintf("batch-%s-%dpkgs", hex.EncodeToString(hash[:8]), len(paths)), paths
}

// sanitisePath replaces path separators with underscores to produce
// a safe filename component.
//
// Takes path (string) which is the filesystem path to sanitise.
//
// Returns string which is the sanitised path with separators
// replaced by underscores.
func sanitisePath(path string) string {
	return strings.ReplaceAll(path, "/", "_")
}
