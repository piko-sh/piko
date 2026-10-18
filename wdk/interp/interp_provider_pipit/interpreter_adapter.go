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

package interp_provider_pipit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/safedisk"
	"pipit.sh/pipit"
)

const (
	// directoryPermission is the permission mode for directories created during bytecode
	// emission.
	directoryPermission = 0o750

	// filePermission is the permission mode for files written during bytecode emission.
	filePermission = 0o600

	// compiledDirectory is the emission subdirectory that receives bytecode, manifests and
	// compilation errors.
	compiledDirectory = "compiled"

	// sourceDirectory is the emission subdirectory that receives the compiled sources.
	sourceDirectory = "source"

	// fieldDirectory is the structured-logging key for directory paths.
	fieldDirectory = "directory"

	// fieldPath is the structured-logging key for file paths.
	fieldPath = "path"
)

// interpreterAdapter wraps *pipit.Interpreter to implement InterpreterPort.
type interpreterAdapter struct {
	// service is the underlying pipit interpreter.
	service *pipit.Interpreter

	// bytecodeEmissionDirectory is the root directory for emitting source and compiled
	// bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string
}

// emittedFile is one file written by bytecode emission.
type emittedFile struct {
	// name is the file name within the bytecode directory.
	name string

	// content is the file's bytes.
	content []byte
}

var (
	_ templater_domain.InterpreterPort = (*interpreterAdapter)(nil)
)

// CompileAndExecute compiles packages together and executes their init functions.
//
// Interpreter panics become errors. Compilation failures include all problems and
// available hints.
//
// Takes modulePath (string) which identifies the module.
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
//
// Returns error when compilation or init execution fails, or the interpreter panics.
func (a *interpreterAdapter) CompileAndExecute(ctx context.Context, modulePath string, packages map[string]map[string]string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = interpreterPanicError(ctx, "compiling and initialising a program", recovered)
		}
	}()

	if a.bytecodeEmissionDirectory != "" {
		a.emitSourceFiles(ctx, modulePath, packages)
	}

	compiledFileSet, compileErr := a.service.CompileProgram(ctx, modulePath, packages)

	if a.bytecodeEmissionDirectory != "" {
		a.emitCompilationOutput(ctx, packages, compiledFileSet, compileErr)
	}

	if compileErr != nil {
		return newCompileProblemsError(modulePath, compileErr)
	}

	if err := a.service.ExecuteInits(ctx, compiledFileSet); err != nil {
		return fmt.Errorf("initialising module %q: %w", modulePath, err)
	}

	return nil
}

// HasRegisteredPackage reports whether the given import path is available in the symbol
// registry.
//
// Takes importPath (string) which is the full package import path to look up.
//
// Returns bool which is true when the package is registered.
func (a *interpreterAdapter) HasRegisteredPackage(importPath string) bool {
	return a.service.HasRegisteredPackage(importPath)
}

// CompilesFile reports whether pipit compiles a project Go file, with the rules and build
// tags it applies when it compiles the program.
//
// Takes name (string) which is the file's name; only the base name counts.
// Takes source ([]byte) which is the file's content.
//
// Returns bool which is true when the file is compiled.
// Returns error when the file's build constraints are malformed.
func (a *interpreterAdapter) CompilesFile(name string, source []byte) (bool, error) {
	return a.service.MatchFile(name, source)
}

// emitSourceFiles writes source files to the emission directory for debugging. They are
// written before compilation starts so they survive a compiler crash.
//
// Takes modulePath (string) which identifies the module being compiled.
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
func (a *interpreterAdapter) emitSourceFiles(ctx context.Context, modulePath string, packages map[string]map[string]string) {
	_, l := logger_domain.From(ctx, log)
	sandbox, err := safedisk.NewSandbox(a.bytecodeEmissionDirectory, safedisk.ModeReadWrite)
	if err != nil {
		l.Warn("Bytecode emission could not open its directory",
			logger_domain.String(fieldDirectory, a.bytecodeEmissionDirectory), logger_domain.Error(err))
		return
	}
	defer func() { _ = sandbox.Close() }()

	for relativePath, files := range packages {
		packageDirectory := filepath.Join(sourceDirectory, sanitisePath(modulePath), sanitisePath(relativePath))
		if err := sandbox.MkdirAll(packageDirectory, directoryPermission); err != nil {
			l.Warn("Bytecode emission could not create a source directory",
				logger_domain.String(fieldDirectory, packageDirectory), logger_domain.Error(err))
			continue
		}

		for filename, source := range files {
			outputPath := filepath.Join(packageDirectory, filename)
			if err := sandbox.WriteFile(outputPath, []byte(source), filePermission); err != nil {
				l.Warn("Bytecode emission could not write a source file",
					logger_domain.String(fieldPath, outputPath), logger_domain.Error(err))
			}
		}
	}
}

// emitCompilationOutput writes a package-path manifest and either bytecode with its
// disassembly or the compilation error for post-mortem inspection. A program that cannot
// be packed gets its packing error in place of the bytecode.
//
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
// Takes compiledFileSet (*pipit.CompiledFileSet) which is the compiled program, or nil
// when compilation failed.
// Takes compileErr (error) which is the compilation failure, or nil.
func (a *interpreterAdapter) emitCompilationOutput(
	ctx context.Context,
	packages map[string]map[string]string,
	compiledFileSet *pipit.CompiledFileSet,
	compileErr error,
) {
	_, l := logger_domain.From(ctx, log)
	sandbox, err := safedisk.NewSandbox(a.bytecodeEmissionDirectory, safedisk.ModeReadWrite)
	if err != nil {
		l.Warn("Bytecode emission could not open its directory",
			logger_domain.String(fieldDirectory, a.bytecodeEmissionDirectory), logger_domain.Error(err))
		return
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.MkdirAll(compiledDirectory, directoryPermission); err != nil {
		l.Warn("Bytecode emission could not create the bytecode directory",
			logger_domain.String(fieldDirectory, compiledDirectory), logger_domain.Error(err))
		return
	}

	suffix, sortedPaths := bytecodeFileSuffix(packages)
	outputs := []emittedFile{{
		name:    fmt.Sprintf("bytecode-%s.txt", suffix),
		content: []byte(strings.Join(sortedPaths, "\n") + "\n"),
	}}
	if compiledFileSet != nil {
		outputs = append(outputs,
			packedBytecodeOutput(suffix, compiledFileSet),
			emittedFile{name: fmt.Sprintf("bytecode-%s.pkasm", suffix), content: []byte(pipit.DisassembleAssembly(compiledFileSet))},
		)
	}
	if compileErr != nil {
		outputs = append(outputs, emittedFile{name: fmt.Sprintf("bytecode-%s.error.txt", suffix), content: []byte(compileErr.Error() + "\n")})
	}

	for _, output := range outputs {
		outputPath := filepath.Join(compiledDirectory, output.name)
		if err := sandbox.WriteFile(outputPath, output.content, filePermission); err != nil {
			l.Warn("Bytecode emission could not write a file",
				logger_domain.String(fieldPath, outputPath), logger_domain.Error(err))
		}
	}
}

// packedBytecodeOutput packs a compiled program for emission, or records why it could not
// be packed so the failure sits next to the program's disassembly.
//
// Takes suffix (string) which names the compilation batch.
// Takes compiledFileSet (*pipit.CompiledFileSet) which is the compiled program.
//
// Returns emittedFile which holds the packed bytecode, or the packing error as text.
func packedBytecodeOutput(suffix string, compiledFileSet *pipit.CompiledFileSet) emittedFile {
	packed, err := pipit.PackCompiledFileSetToBytes(compiledFileSet)
	if err != nil {
		return emittedFile{name: fmt.Sprintf("bytecode-%s.pack-error.txt", suffix), content: []byte(err.Error() + "\n")}
	}
	return emittedFile{name: fmt.Sprintf("bytecode-%s.bin", suffix), content: packed}
}

// bytecodeFileSuffix builds a filename suffix from the relative package paths in a
// compilation batch.
//
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
//
// Returns string which is the sanitised path for single-package batches or a short hash
// for multi-package batches.
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

// sanitisePath replaces path separators with underscores to produce a safe filename
// component.
//
// Takes path (string) which is the filesystem path to sanitise.
//
// Returns string which is the sanitised path with separators replaced by underscores.
func sanitisePath(path string) string {
	return strings.ReplaceAll(path, "/", "_")
}
