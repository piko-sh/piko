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

package cli

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/wdk/json"

	"piko.sh/piko/internal/interp/interp_adapters"
	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/interp/interp_schema"
)

// filePermission is the default permission mode for files written by the CLI.
const filePermission = 0o644

// bytecodeFlags holds parsed flags for the bytecode subcommand.
type bytecodeFlags struct {
	// save is the output file path for saving compiled bytecode.
	save string

	// descriptor is the path to the types descriptor JSON file.
	descriptor string

	// compact enables compact JSON output instead of pretty-printed.
	compact bool

	// asm enables human-readable assembly output.
	asm bool
}

// RunBytecode runs the bytecode subcommand, writing to os.Stdout and
// os.Stderr.
//
// Takes arguments ([]string) which contains the command-line arguments.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunBytecode(arguments []string) int {
	return RunBytecodeWithIO(arguments, os.Stdout, os.Stderr)
}

// RunBytecodeWithIO runs the bytecode subcommand with explicit output
// writers.
//
// Usage: piko bytecode <file.go> [flags]
//
// Compiles a Go source file using the interpreter's compiler and
// outputs the bytecode as JSON or saves it as a .bin file.
//
// Takes arguments ([]string) which contains the command-line arguments.
// Takes stdout (io.Writer) which receives the JSON output.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunBytecodeWithIO(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		bytecodeUsage(stderr)
		return 1
	}

	flags, filePath, ok := parseBytecodeArgs(arguments, stderr)
	if !ok {
		return 1
	}

	cleanedFilePath := filepath.Clean(filePath)
	source, err := os.ReadFile(cleanedFilePath) //nolint:gosec // user-specified path, sanitised
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error reading file: %v\n", err)
		return 1
	}

	service := interp_domain.NewService()
	providers := []interp_domain.SymbolProviderPort{driven_system_symbols.NewProvider()}

	if flags.descriptor != "" {
		descriptorPackages, descriptorError := loadTypesDescriptor(flags.descriptor, source)
		if descriptorError != nil {
			_, _ = fmt.Fprintf(stderr, "Error loading types descriptor: %v\n", descriptorError)
			return 1
		}
		providers = append(providers, &descriptorTypesProvider{packages: descriptorPackages})
	}

	service.UseSymbolProviders(providers...)

	fileName := filepath.Base(filePath)
	compiledFileSet, err := service.CompileFileSet(context.Background(), map[string]string{
		fileName: string(source),
	})
	if err != nil {
		printCompilationError(err, flags, filePath, stderr)
		return 1
	}

	if flags.save != "" {
		return saveBytecodeFile(compiledFileSet, flags.save, stderr)
	}

	if flags.asm {
		_, _ = fmt.Fprint(stdout, compiledFileSet.DisassembleAssembly())
		return 0
	}

	return printBytecodeInspection(compiledFileSet, flags, stdout, stderr)
}

// printCompilationError writes compilation error details and hints
// to stderr.
//
// Takes err (error) which is the compilation error to report.
// Takes flags (bytecodeFlags) which holds the current flag state.
// Takes filePath (string) which is the source file path for hints.
// Takes stderr (io.Writer) which receives the error output.
func printCompilationError(err error, flags bytecodeFlags, filePath string, stderr io.Writer) {
	_, _ = fmt.Fprintf(stderr, "Compilation error: %v\n", err)
	if !strings.Contains(err.Error(), "not found in symbol registry") {
		return
	}
	if flags.descriptor == "" {
		_, _ = fmt.Fprint(stderr, "\nThis file imports packages the CLI doesn't know about.\n")
		_, _ = fmt.Fprint(stderr, "Run 'piko extract' first, then pass the generated descriptor:\n\n")
		_, _ = fmt.Fprintf(stderr, "  piko bytecode --types <output>/gen_types_descriptor.json %s\n\n", filePath)
		_, _ = fmt.Fprint(stderr, "Run 'piko bytecode --help' for a full guide.\n")
	} else {
		_, _ = fmt.Fprint(stderr, "\nThe types descriptor does not include all required packages.\n")
		_, _ = fmt.Fprint(stderr, "Re-run 'piko extract' to regenerate it, then try again.\n")
	}
}

// parseBytecodeArgs separates flags from the file path argument.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns bytecodeFlags, the file path, and whether parsing succeeded.
func parseBytecodeArgs(arguments []string, stderr io.Writer) (bytecodeFlags, string, bool) {
	var flags bytecodeFlags
	var filePath string

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		switch {
		case argument == "--asm":
			flags.asm = true
		case argument == "--compact":
			flags.compact = true
		case argument == "--save":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprint(stderr, "--save requires an output file path\n\n")
				bytecodeUsage(stderr)
				return flags, "", false
			}
			i++
			flags.save = arguments[i]
		case strings.HasPrefix(argument, "--save="):
			flags.save = strings.TrimPrefix(argument, "--save=")
		case argument == "--types":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprint(stderr, "--types requires a descriptor file path\n\n")
				bytecodeUsage(stderr)
				return flags, "", false
			}
			i++
			flags.descriptor = arguments[i]
		case strings.HasPrefix(argument, "--types="):
			flags.descriptor = strings.TrimPrefix(argument, "--types=")
		case argument == "help" || argument == "-h" || argument == "--help":
			bytecodeUsage(stderr)
			return flags, "", false
		case strings.HasPrefix(argument, "-"):
			_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
			bytecodeUsage(stderr)
			return flags, "", false
		default:
			if filePath != "" {
				_, _ = fmt.Fprintf(stderr, "Unexpected argument: %s\n\n", argument)
				bytecodeUsage(stderr)
				return flags, "", false
			}
			filePath = argument
		}
	}

	if filePath == "" {
		_, _ = fmt.Fprint(stderr, "Missing file path\n\n")
		bytecodeUsage(stderr)
		return flags, "", false
	}

	return flags, filePath, true
}

// saveBytecodeFile packs the compiled file set into FlatBuffer format
// and writes it to the given path.
//
// Takes compiledFileSet which is the compiled bytecode to serialise.
// Takes outputPath (string) which is the destination file path.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns int which is 0 on success or 1 on failure.
func saveBytecodeFile(compiledFileSet *interp_domain.CompiledFileSet, outputPath string, stderr io.Writer) int {
	versionedData := interp_adapters.PackCompiledFileSetToBytes(compiledFileSet)

	if err := os.WriteFile(outputPath, versionedData, filePermission); err != nil { //nolint:gosec // user-specified path
		_, _ = fmt.Fprintf(stderr, "Error writing file: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stderr, "Bytecode saved to %s (%d bytes)\n", outputPath, len(versionedData))
	return 0
}

// printBytecodeInspection converts the compiled file set to an
// inspection JSON and prints it.
//
// Takes compiledFileSet which is the compiled bytecode to inspect.
// Takes flags (bytecodeFlags) which controls JSON formatting.
// Takes stdout (io.Writer) which receives the JSON output.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns int which is 0 on success or 1 on failure.
func printBytecodeInspection(compiledFileSet *interp_domain.CompiledFileSet, flags bytecodeFlags, stdout, stderr io.Writer) int {
	inspection := inspectCompiledFileSet(compiledFileSet)

	var output []byte
	var err error
	if flags.compact {
		output, err = json.Marshal(inspection)
	} else {
		output, err = json.MarshalIndent(inspection, "", "  ")
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error marshalling JSON: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, string(output))
	return 0
}

// inspectCompiledFileSet converts a compiled file set into a
// JSON-serialisable inspection summary, walking the function tree
// directly rather than going through FlatBuffer serialisation.
//
// Takes compiledFileSet (*interp_domain.CompiledFileSet) which is the
// compiled bytecode to inspect.
//
// Returns *interp_schema.BytecodeInspection which is the inspection
// summary.
func inspectCompiledFileSet(compiledFileSet *interp_domain.CompiledFileSet) *interp_schema.BytecodeInspection {
	inspection := &interp_schema.BytecodeInspection{
		Entrypoints: make(map[string]uint16),
	}

	if root := compiledFileSet.Root(); root != nil {
		inspection.Root = inspectFunction(root)
	}
	if varInit := compiledFileSet.VariableInitFunction(); varInit != nil {
		inspection.VarInit = inspectFunction(varInit)
	}

	maps.Copy(inspection.Entrypoints, compiledFileSet.Entrypoints())
	inspection.InitFunctions = compiledFileSet.InitFuncs()

	return inspection
}

var registerBankNames = [...]string{
	"int", "float", "string", "general", "bool", "uint", "complex",
}

// inspectFunction converts a compiled function into a
// JSON-serialisable inspection summary.
//
// Takes function (*interp_domain.CompiledFunction) which is the
// compiled function to inspect.
//
// Returns *interp_schema.FunctionInspection which is the inspection
// summary.
func inspectFunction(function *interp_domain.CompiledFunction) *interp_schema.FunctionInspection {
	inspection := &interp_schema.FunctionInspection{
		Name:         function.ExportName(),
		SourceFile:   function.ExportSourceFile(),
		NumRegisters: make(map[string]uint32),
		Instructions: len(function.Body()),
		Constants:    make(map[string]int),
		CallSites:    len(function.CallSites()),
		Upvalues:     len(function.UpvalueDescriptors()),
		IsVariadic:   function.ExportIsVariadic(),
	}

	for i, count := range function.NumRegistersSlice() {
		if count > 0 && i < len(registerBankNames) {
			inspection.NumRegisters[registerBankNames[i]] = count
		}
	}

	addNonZero(inspection.Constants, "int", len(function.IntConstants()))
	addNonZero(inspection.Constants, "float", len(function.FloatConstants()))
	addNonZero(inspection.Constants, "string", len(function.StringConstants()))
	addNonZero(inspection.Constants, "bool", len(function.BoolConstants()))
	addNonZero(inspection.Constants, "uint", len(function.UintConstants()))
	addNonZero(inspection.Constants, "complex", len(function.ComplexConstants()))
	addNonZero(inspection.Constants, "general", len(function.GeneralConstantDescriptors()))

	for _, child := range function.ExportFunctions() {
		inspection.Functions = append(inspection.Functions, inspectFunction(child))
	}

	return inspection
}

// addNonZero inserts a key-value pair into the map only if the value
// is greater than zero.
//
// Takes m (map[string]int) which is the target map.
// Takes key (string) which is the map key.
// Takes value (int) which is the value to insert if positive.
func addNonZero(m map[string]int, key string, value int) {
	if value > 0 {
		m[key] = value
	}
}

// compileOnlyPlaceholder is a sentinel reflect.Value for compile-only
// mode.
//
// The actual reflect.Value is only needed at runtime; for compilation,
// the symbol registry just needs to confirm the symbol exists. This
// value is never inspected, only the generalConstantDescriptor
// (package path + symbol name) matters for serialisation.
var compileOnlyPlaceholder = reflect.ValueOf(0)

// descriptorTypesProvider supplies pre-built types.Package objects and
// placeholder symbol entries loaded from a types descriptor file. The
// placeholder reflect.Value entries allow the compiler's symbol Lookup
// to succeed; actual runtime values are resolved when the bytecode is
// later loaded by the application.
type descriptorTypesProvider struct {
	// packages maps import paths to their loaded type packages.
	packages map[string]*types.Package
}

// Exports returns a symbol export table with placeholder entries for
// every exported symbol in the loaded packages. The compiler only
// needs Lookup to succeed (returning a valid reflect.Value); the
// actual value is only used at runtime.
//
// Returns interp_domain.SymbolExports with placeholder entries.
func (p *descriptorTypesProvider) Exports() interp_domain.SymbolExports {
	exports := make(interp_domain.SymbolExports, len(p.packages))
	for importPath, pkg := range p.packages {
		scope := pkg.Scope()
		names := scope.Names()
		symbols := make(map[string]reflect.Value, len(names))
		for _, name := range names {
			object := scope.Lookup(name)
			if object.Exported() {
				symbols[name] = compileOnlyPlaceholder
			}
		}
		if len(symbols) > 0 {
			exports[importPath] = symbols
		}
	}
	return exports
}

// TypesPackages returns the pre-built types.Package objects loaded
// from the descriptor.
//
// Returns map[string]*types.Package mapping import paths to their
// type packages.
func (p *descriptorTypesProvider) TypesPackages() map[string]*types.Package {
	return p.packages
}

// loadTypesDescriptor reads a types descriptor JSON file, combines
// its import paths with the source file's imports, and loads
// types.Package objects via golang.org/x/tools/go/packages.
//
// Takes descriptorPath (string) which is the path to the
// gen_types_descriptor.json file.
// Takes source ([]byte) which is the Go source being compiled, used
// to extract additional import paths not in the descriptor.
//
// Returns map[string]*types.Package which contains the loaded type
// packages.
// Returns error when the file cannot be read or packages cannot be
// loaded.
func loadTypesDescriptor(descriptorPath string, source []byte) (map[string]*types.Package, error) {
	cleanedDescriptorPath := filepath.Clean(descriptorPath)
	data, err := os.ReadFile(cleanedDescriptorPath) //nolint:gosec // user-specified path, sanitised
	if err != nil {
		return nil, fmt.Errorf("reading descriptor: %w", err)
	}

	descriptorPaths, err := driver_symbols_extract.ReadTypesDescriptor(data)
	if err != nil {
		return nil, err
	}

	allPaths := mergeImportPaths(descriptorPaths, extractSourceImports(source))
	if len(allPaths) == 0 {
		return nil, nil
	}

	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedName,
	}

	loaded, err := packages.Load(config, allPaths...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	result := make(map[string]*types.Package, len(loaded))
	for _, pkg := range loaded {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("loading %s: %w", pkg.PkgPath, pkg.Errors[0])
		}
		if pkg.Types != nil {
			result[pkg.PkgPath] = pkg.Types
		}
	}

	return result, nil
}

// extractSourceImports parses import paths from Go source code.
//
// Takes source ([]byte) which is the Go source to parse.
//
// Returns []string which lists the import paths found in the source.
func extractSourceImports(source []byte) []string {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "", source, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	importPaths := make([]string, 0, len(file.Imports))
	for _, importSpec := range file.Imports {
		importPath, unquoteError := strconv.Unquote(importSpec.Path.Value)
		if unquoteError == nil {
			importPaths = append(importPaths, importPath)
		}
	}

	return importPaths
}

// mergeImportPaths combines two slices of import paths, removing
// duplicates.
//
// Takes a ([]string) which is the first set of import paths.
// Takes b ([]string) which is the second set of import paths.
//
// Returns []string which contains the deduplicated union.
func mergeImportPaths(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	merged := make([]string, 0, len(a)+len(b))

	for _, path := range a {
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			merged = append(merged, path)
		}
	}
	for _, path := range b {
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			merged = append(merged, path)
		}
	}

	return merged
}

// bytecodeUsage prints the command-line help text for the bytecode
// command.
//
// Takes w (io.Writer) which receives the usage text.
func bytecodeUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `piko bytecode compiles a Go source file and outputs its bytecode.

Usage:
  piko bytecode <file.go> [flags]

Flags:
  --asm              Print human-readable bytecode assembly (.pkasm format)
  --compact          Compact JSON output (default: pretty-printed)
  --save <path>      Save compiled bytecode as a .bin file instead of printing JSON
  --types <path>     Load a types descriptor for project-specific imports

Compiling files with project imports:

  Generated .go files typically import project-specific packages (your domain
  types, services, etc.) that the CLI doesn't know about by default. To compile
  these files, you need a types descriptor - a JSON file listing the import
  paths that 'piko extract' has processed.

  1. Run 'piko extract' in your project (this also generates gen_types_descriptor.json):

       piko extract

  2. Pass the descriptor to 'piko bytecode' with --types:

       piko bytecode --types internal/symbols/gen_types_descriptor.json dist/.../generated.go

  The descriptor is written to the same output directory as your gen_*.go files.
  Its location depends on the 'output' field in your piko-symbols.yaml manifest.

  Without --types, only standard library imports are available.

Examples:
  piko bytecode generated.go                       # Stdlib-only file
  piko bytecode generated.go --asm                 # Human-readable assembly
  piko bytecode generated.go --compact             # Compact JSON output
  piko bytecode generated.go --save output.bin     # Save as FlatBuffer binary

  piko bytecode --types internal/symbols/gen_types_descriptor.json \
    dist/pages/my_page/generated.go                # File with project imports

  piko bytecode --types internal/symbols/gen_types_descriptor.json \
    --save output.bin dist/emails/welcome/generated.go

`)
}
