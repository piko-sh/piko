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
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
	"piko.sh/piko/wdk/safedisk"
)

// extractFilePerms is the file permission used when writing
// extracted Go source files.
const extractFilePerms = 0o644

// extractGenerateFlags holds the parsed flags for the extract
// generate subcommand.
type extractGenerateFlags struct {
	// manifest is the path to the YAML manifest file.
	manifest string

	// output overrides the output directory from the manifest.
	output string

	// packageName overrides the package name from the manifest.
	packageName string

	// dryRun prints what would be generated without writing files.
	dryRun bool
}

// extractEmitContext bundles the parameters needed by the
// package file emission helpers, reducing argument counts on
// the individual functions.
type extractEmitContext struct {
	// genericConfigs maps import paths to their package-level
	// extraction configuration for generic handling.
	genericConfigs map[string]driver_symbols_extract.PackageConfig

	// manifest holds the loaded extraction manifest describing
	// which packages to extract and where to write output.
	manifest *driver_symbols_extract.Manifest

	// sandbox scopes file writes to the output directory.
	sandbox safedisk.Sandbox

	// stdout receives normal progress messages.
	stdout io.Writer

	// stderr receives error and diagnostic messages.
	stderr io.Writer

	// flags holds the parsed command-line flags.
	flags extractGenerateFlags
}

// runExtractGenerate runs the extract generate subcommand, which reads a
// piko-symbols manifest and produces the reflect-based Go symbol files that
// the interpreter uses to resolve external package imports.
//
// Takes arguments ([]string) which contains the subcommand arguments (without
// the leading "generate").
// Takes stdout (io.Writer) which receives normal progress messages.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func runExtractGenerate(arguments []string, stdout, stderr io.Writer) int {
	flags, status := parseExtractGenerateArgs(arguments, stderr)
	if status != parseOK {
		if status == parseHelp {
			extractGenerateUsage(stdout)
			return 0
		}
		return 1
	}

	manifest, err := loadAndApplyManifest(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Extracting %d packages...\n", len(manifest.Packages))

	genericConfigs := manifest.GenericConfigs()
	packages, err := driver_symbols_extract.Extract(manifest.ImportPaths(), genericConfigs)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error extracting packages: %s\n", err)
		return 1
	}

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error creating sandbox factory: %s\n", err)
		return 1
	}

	var sandbox safedisk.Sandbox
	if !flags.dryRun {
		sandbox, err = factory.Create("extract-output", manifest.Output, safedisk.ModeReadWrite)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error creating sandbox for %s: %s\n", manifest.Output, err)
			return 1
		}
		defer func() { _ = sandbox.Close() }()
	}

	ec := extractEmitContext{
		manifest:       manifest,
		genericConfigs: genericConfigs,
		flags:          flags,
		sandbox:        sandbox,
		stdout:         stdout,
		stderr:         stderr,
	}
	if err := emitPackageFiles(packages, ec); err != nil {
		return 1
	}

	printExtractGenerateSummary(flags, stdout)
	return 0
}

// parseExtractGenerateArgs separates flags from positional arguments.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
// Takes stderr (io.Writer) which receives error and usage output.
//
// Returns extractGenerateFlags which holds the parsed flag values.
// Returns parseStatus which discriminates success, help, and error
// outcomes.
func parseExtractGenerateArgs(arguments []string, stderr io.Writer) (extractGenerateFlags, parseStatus) {
	flags := extractGenerateFlags{
		manifest: "piko-symbols.yaml",
	}

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if !strings.HasPrefix(argument, "-") {
			_, _ = fmt.Fprintf(stderr, "Unexpected argument: %s\n\n", argument)
			extractGenerateUsage(stderr)
			return flags, parseError
		}

		switch argument {
		case "-h", "--help":
			return flags, parseHelp
		case "--manifest", "-manifest":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.manifest = arguments[i]
		case "--output", "-output":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.output = arguments[i]
		case "--package", "-package":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.packageName = arguments[i]
		case "--dry-run":
			flags.dryRun = true
		default:
			_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
			extractGenerateUsage(stderr)
			return flags, parseError
		}
	}

	return flags, parseOK
}

// loadAndApplyManifest loads the extraction manifest and applies flag overrides.
//
// Takes flags (extractGenerateFlags) which holds the parsed command-line flags
// including optional output and package name overrides.
//
// Returns *driver_symbols_extract.Manifest which is the loaded manifest with
// overrides applied.
// Returns error when the manifest cannot be loaded.
func loadAndApplyManifest(flags extractGenerateFlags) (*driver_symbols_extract.Manifest, error) {
	manifest, err := driver_symbols_extract.LoadManifest(flags.manifest)
	if err != nil {
		return nil, err
	}

	if flags.output != "" {
		manifest.Output = flags.output
	}
	if flags.packageName != "" {
		manifest.Package = flags.packageName
	}
	return manifest, nil
}

// emitPackageFiles generates and writes Go source files for each extracted
// package, including the types loader when generics are present.
//
// Takes packages ([]driver_symbols_extract.ExtractedPackage) which are the
// extracted packages to generate files for.
// Takes ec (extractEmitContext) which bundles manifest, configs, flags, sandbox,
// and output writers.
//
// Returns error when any generation or write step fails.
func emitPackageFiles(
	packages []driver_symbols_extract.ExtractedPackage,
	ec extractEmitContext,
) error {
	if err := writeRegisterFile(ec.manifest, ec.flags, ec.sandbox, ec.stdout, ec.stderr); err != nil {
		return err
	}

	for i := range packages {
		if err := emitSinglePackage(&packages[i], ec); err != nil {
			return err
		}
	}

	typesLoaderPaths := collectAllImportPaths(packages)
	if len(typesLoaderPaths) > 0 {
		if err := writeTypesLoader(typesLoaderPaths, ec.manifest, ec.flags, ec.sandbox, ec.stdout, ec.stderr); err != nil {
			_, _ = fmt.Fprintf(ec.stderr, "Error: %s\n", err)
			return err
		}
		if err := writeTypesDescriptor(typesLoaderPaths, ec.manifest, ec.flags, ec.sandbox, ec.stdout, ec.stderr); err != nil {
			_, _ = fmt.Fprintf(ec.stderr, "Error: %s\n", err)
			return err
		}
	}
	return nil
}

// writeRegisterFile emits gen_register.go, which declares the
// package-level Symbols map that every other gen_*.go file populates
// in its init(). Without this file the generated package fails to
// compile.
//
// Takes manifest (*driver_symbols_extract.Manifest) which supplies
// the destination package name and output directory.
// Takes flags (extractGenerateFlags) which indicates dry-run mode.
// Takes sandbox (safedisk.Sandbox) which scopes the file write to
// the output directory.
// Takes stdout (io.Writer) which receives progress messages.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns error when generation or writing fails.
func writeRegisterFile(manifest *driver_symbols_extract.Manifest, flags extractGenerateFlags, sandbox safedisk.Sandbox, stdout, stderr io.Writer) error {
	source, err := driver_symbols_extract.GenerateRegisterFile(manifest.Package)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error generating register file: %s\n", err)
		return err
	}
	filename := "gen_register.go"
	outPath := filepath.Join(manifest.Output, filename)
	if flags.dryRun {
		_, _ = fmt.Fprintf(stdout, "  register -> %s (%d bytes)\n", outPath, len(source))
		return nil
	}
	if err := sandbox.WriteFile(filename, source, extractFilePerms); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error writing %s: %s\n", outPath, err)
		return err
	}
	_, _ = fmt.Fprintf(stdout, "  register -> %s\n", outPath)
	return nil
}

// emitSinglePackage generates and writes (or dry-run prints) the source file
// for one extracted package.
//
// Takes pkg (driver_symbols_extract.ExtractedPackage) which is the package to
// process.
// Takes ec (extractEmitContext) which bundles manifest, configs, flags, sandbox,
// and output writers.
//
// Returns error when generation or writing fails.
func emitSinglePackage(
	pkg *driver_symbols_extract.ExtractedPackage,
	ec extractEmitContext,
) error {
	generatorConfig := ec.genericConfigs[pkg.ImportPath]
	source, err := driver_symbols_extract.GenerateFile(*pkg, ec.manifest.Package, generatorConfig)
	if err != nil {
		_, _ = fmt.Fprintf(ec.stderr, "Error generating %s: %s\n", pkg.ImportPath, err)
		return err
	}

	if source == nil {
		_, _ = fmt.Fprintf(ec.stdout, "  %s: no extractable symbols, skipping\n", pkg.ImportPath)
		return nil
	}

	filename := driver_symbols_extract.OutputFileName(pkg.ImportPath)
	outPath := filepath.Join(ec.manifest.Output, filename)

	if ec.flags.dryRun {
		_, _ = fmt.Fprintf(ec.stdout, "  %s -> %s (%d bytes)\n", pkg.ImportPath, outPath, len(source))
		return nil
	}

	if err := ec.sandbox.WriteFile(filename, source, extractFilePerms); err != nil {
		_, _ = fmt.Fprintf(ec.stderr, "Error writing %s: %s\n", outPath, err)
		return err
	}

	_, _ = fmt.Fprintf(ec.stdout, "  %s -> %s\n", pkg.ImportPath, outPath)
	return nil
}

// collectAllImportPaths returns the import paths of all extracted packages.
// These are loaded via go/importer at init time to provide complete
// types.Package objects (including untyped constants, type aliases, and
// interface embeddings) that cannot be synthesised from reflect values alone.
//
// Takes packages ([]driver_symbols_extract.ExtractedPackage) which are the
// extracted packages to collect paths from.
//
// Returns []string which lists all package import paths.
func collectAllImportPaths(packages []driver_symbols_extract.ExtractedPackage) []string {
	paths := make([]string, 0, len(packages))
	for i := range packages {
		paths = append(paths, packages[i].ImportPath)
	}
	return paths
}

// printExtractGenerateSummary prints the final status line for the extract
// generate command.
//
// Takes flags (extractGenerateFlags) which determines whether this was a
// dry run.
// Takes stdout (io.Writer) which receives the summary message.
func printExtractGenerateSummary(flags extractGenerateFlags, stdout io.Writer) {
	if flags.dryRun {
		_, _ = fmt.Fprint(stdout, "\nDry run complete. No files were written.\n")
	} else {
		_, _ = fmt.Fprint(stdout, "\nDone.\n")
	}
}

// writeTypesLoader generates and writes the types loader Go
// source files. These provide pre-built types.Package objects
// for all registered packages, ensuring the type checker has
// complete information (including untyped constants, type
// aliases, and interface embeddings).
//
// Takes genericPaths ([]string) which lists the import paths
// of packages to load.
// Takes manifest (*driver_symbols_extract.Manifest) which
// provides the output directory and package name.
// Takes flags (extractGenerateFlags) which holds the parsed
// command-line flags.
// Takes sandbox (safedisk.Sandbox) which scopes file writes
// to the output directory.
// Takes stdout (io.Writer) which receives progress messages.
//
// Returns error when generation or writing of the loader
// files fails.
func writeTypesLoader(genericPaths []string, manifest *driver_symbols_extract.Manifest, flags extractGenerateFlags, sandbox safedisk.Sandbox, stdout, _ io.Writer) error {
	source, err := driver_symbols_extract.GenerateTypesLoaderFile(genericPaths, manifest.Package)
	if err != nil {
		return fmt.Errorf("generating types_loader: %w", err)
	}
	filename := "gen_types_loader.go"
	outPath := filepath.Join(manifest.Output, filename)
	if flags.dryRun {
		_, _ = fmt.Fprintf(stdout, "  types_loader -> %s (%d bytes)\n", outPath, len(source))
		return nil
	}
	if err := sandbox.WriteFile(filename, source, extractFilePerms); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	_, _ = fmt.Fprintf(stdout, "  types_loader -> %s\n", outPath)

	wasmSource, err := driver_symbols_extract.GenerateTypesLoaderWASMFile(genericPaths, manifest.Package)
	if err != nil {
		return fmt.Errorf("generating types_loader wasm stub: %w", err)
	}
	wasmFilename := "gen_types_loader_javascript.go"
	wasmOutPath := filepath.Join(manifest.Output, wasmFilename)
	if flags.dryRun {
		_, _ = fmt.Fprintf(stdout, "  types_loader_wasm -> %s (%d bytes)\n", wasmOutPath, len(wasmSource))
		return nil
	}
	if err := sandbox.WriteFile(wasmFilename, wasmSource, extractFilePerms); err != nil {
		return fmt.Errorf("writing %s: %w", wasmOutPath, err)
	}
	_, _ = fmt.Fprintf(stdout, "  types_loader_wasm -> %s\n", wasmOutPath)
	return nil
}

// writeTypesDescriptor generates and writes the types descriptor JSON file.
// This file lists all extracted import paths so that piko bytecode can load
// the corresponding types.Package objects at compile time.
//
// Takes importPaths ([]string) which lists the import paths to include.
// Takes manifest (*driver_symbols_extract.Manifest) which provides the output
// directory.
// Takes flags (extractGenerateFlags) which holds the parsed command-line flags.
// Takes sandbox (safedisk.Sandbox) which scopes file writes to the output
// directory.
// Takes stdout (io.Writer) which receives progress messages.
//
// Returns error when generation or writing of the descriptor fails.
func writeTypesDescriptor(importPaths []string, manifest *driver_symbols_extract.Manifest, flags extractGenerateFlags, sandbox safedisk.Sandbox, stdout, _ io.Writer) error {
	descriptor, err := driver_symbols_extract.GenerateTypesDescriptorFile(importPaths)
	if err != nil {
		return fmt.Errorf("generating types descriptor: %w", err)
	}

	filename := "gen_types_descriptor.json"
	outPath := filepath.Join(manifest.Output, filename)

	if flags.dryRun {
		_, _ = fmt.Fprintf(stdout, "  types_descriptor -> %s (%d bytes)\n", outPath, len(descriptor))
		return nil
	}

	if err := sandbox.WriteFile(filename, descriptor, extractFilePerms); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	_, _ = fmt.Fprintf(stdout, "  types_descriptor -> %s\n", outPath)
	return nil
}

// extractGenerateUsage writes the usage information for the extract generate
// subcommand.
//
// Takes w (io.Writer) which receives the usage text.
func extractGenerateUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko extract generate [flags]

Generate reflect-based Go symbol files from a piko-symbols manifest.

Reads a YAML manifest specifying which packages to extract and writes
Go source files containing reflect.Value symbol tables that the
interpreter loads at runtime.

Flags:
  --manifest <path>    Path to YAML manifest file (default: piko-symbols.yaml)
  --output <directory> Override output directory from manifest
  --package <name>     Override output package name from manifest
  --dry-run            Print what would be generated without writing files
  -h, --help           Show this help message

Manifest format (YAML):

  package: driven_system_symbols
  output: internal/interp/interp_adapters/driven_system_symbols
  packages:
    - fmt
    - strings
    - encoding/json

`)
}
