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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
	"piko.sh/piko/wdk/safedisk"
)

// extractInitFlags holds the parsed flags for the init subcommand.
type extractInitFlags struct {
	// root is the project root directory (default ".").
	root string

	// output is the manifest file path to write.
	output string

	// packageName is the generated-symbols package name to embed in
	// the manifest.
	packageName string

	// generatedDir is the output directory for "piko extract
	// generate" recorded in the manifest.
	generatedDir string

	// force overwrites an existing manifest when true.
	force bool
}

// runExtractInit discovers the project's required third-party
// packages and writes them to a piko-symbols.yaml manifest.
//
// Takes arguments ([]string) which contains the subcommand arguments.
// Takes stdout (io.Writer) which receives progress messages.
// Takes stderr (io.Writer) which receives warnings and errors.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func runExtractInit(arguments []string, stdout, stderr io.Writer) int {
	flags, status := parseExtractInitArgs(arguments, stderr)
	if status != parseOK {
		if status == parseHelp {
			extractInitUsage(stdout)
			return 0
		}
		return 1
	}

	sandbox, filename, err := openManifestSandbox(flags.output)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	defer func() { _ = sandbox.Close() }()

	result, err := driver_symbols_extract.Discover(context.Background(), driver_symbols_extract.DiscoverOptions{
		Root: flags.root,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	return writeInitManifest(stdout, stderr, sandbox, filename, flags, result)
}

// writeInitManifest performs the I/O and reporting half of
// `piko extract init`, separated from the orchestration in
// runExtractInit so it can be unit-tested against a safedisk
// MockSandbox with a synthesised DiscoverResult.
//
// Takes stdout (io.Writer) which receives the summary line.
// Takes stderr (io.Writer) which receives warnings and errors.
// Takes sandbox (safedisk.Sandbox) which scopes the write.
// Takes filename (string) which is the manifest file name inside the
// sandbox.
// Takes flags (extractInitFlags) which supply the rendered manifest
// header fields and the --force policy.
// Takes result (driver_symbols_extract.DiscoverResult) which supplies
// the packages to write and any warnings to surface.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func writeInitManifest(
	stdout, stderr io.Writer,
	sandbox safedisk.Sandbox,
	filename string,
	flags extractInitFlags,
	result driver_symbols_extract.DiscoverResult,
) int {
	if !flags.force {
		if _, statErr := sandbox.Stat(filename); statErr == nil {
			_, _ = fmt.Fprintf(stderr, "Error: %s already exists; use --force to overwrite\n", flags.output)
			return 1
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(stderr, "Error: %s\n", statErr)
			return 1
		}
	}

	contents := renderInitManifest(flags, result.RequiredImports)
	if err := sandbox.WriteFile(filename, []byte(contents), extractFilePerms); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error writing %s: %s\n", flags.output, err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Wrote %s with %d package(s).\n", flags.output, len(result.RequiredImports))
	for _, path := range result.SkippedCgo {
		_, _ = fmt.Fprintf(stderr, "warning: %s uses cgo and cannot be interpreted (not written to manifest)\n", path)
	}
	for _, path := range result.GenericCandidates {
		_, _ = fmt.Fprintf(stderr, "warning: %s exports generic types; edit the manifest to add a generic: block\n", path)
	}
	return 0
}

// parseExtractInitArgs parses the command-line flags for init.
//
// Takes arguments ([]string) which is the raw argument list.
// Takes stderr (io.Writer) which receives usage and diagnostics.
//
// Returns the parsed flags and a parseStatus describing the outcome.
func parseExtractInitArgs(arguments []string, stderr io.Writer) (extractInitFlags, parseStatus) {
	flags := extractInitFlags{
		root:         ".",
		output:       "piko-symbols.yaml",
		packageName:  "piko_symbols",
		generatedDir: "internal/piko_symbols",
	}

	valueTargets := map[string]*string{
		"--root":    &flags.root,
		"--output":  &flags.output,
		"--package": &flags.packageName,
		"--dir":     &flags.generatedDir,
	}

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if !strings.HasPrefix(argument, "-") {
			_, _ = fmt.Fprintf(stderr, "Unexpected argument: %s\n\n", argument)
			extractInitUsage(stderr)
			return flags, parseError
		}

		switch argument {
		case "-h", "--help":
			return flags, parseHelp
		case "--force":
			flags.force = true
		default:
			target, ok := valueTargets[argument]
			if !ok {
				_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
				extractInitUsage(stderr)
				return flags, parseError
			}
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			*target = arguments[i]
		}
	}

	return flags, parseOK
}

// renderInitManifest produces a YAML string for a new piko-symbols
// manifest with the discovered package list. The format matches what
// LoadManifest accepts (package, output, packages sequence).
//
// Takes flags (extractInitFlags) which supplies the header fields.
// Takes packages ([]string) which is the sorted discovered list.
//
// Returns string with trailing newline.
func renderInitManifest(flags extractInitFlags, packages []string) string {
	var builder strings.Builder
	builder.WriteString("# Generated by `piko extract init`. Edit freely.\n")
	_, _ = fmt.Fprintf(&builder, "package: %s\n", flags.packageName)
	_, _ = fmt.Fprintf(&builder, "output: %s\n", flags.generatedDir)
	builder.WriteString("packages:\n")
	for _, path := range packages {
		_, _ = fmt.Fprintf(&builder, "  - %s\n", path)
	}
	return builder.String()
}

// openManifestSandbox opens a safedisk sandbox scoped to the parent
// directory of the manifest output path. Using a sandbox keeps both
// the existence check and the write path-traversal-safe when the
// caller supplies an absolute or relative path.
//
// Takes outputPath (string) which is the destination manifest path.
//
// Returns the sandbox, the cleaned filename inside the sandbox, and
// any error encountered while opening the sandbox.
func openManifestSandbox(outputPath string) (safedisk.Sandbox, string, error) {
	directory, filename := filepath.Split(outputPath)
	if directory == "" {
		directory = "."
	}
	directory = filepath.Clean(directory)

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		return nil, "", fmt.Errorf("creating sandbox factory: %w", err)
	}
	sandbox, err := factory.Create("extract-init", directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, "", fmt.Errorf("creating sandbox for %s: %w", directory, err)
	}
	return sandbox, filename, nil
}

// extractInitUsage writes the usage information for the init subcommand.
//
// Takes w (io.Writer) which receives the usage text.
func extractInitUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko extract init [flags]

Discover the project's required third-party imports and write them to
a piko-symbols.yaml manifest. Refuses to overwrite an existing file
unless --force is given.

After writing the manifest, run "piko extract generate" to produce
the reflect-based symbol files.

Flags:
  --root <dir>         Project root directory (default: .)
  --output <path>      Manifest path to write (default: piko-symbols.yaml)
  --package <name>     Generated-symbols package name
                       (default: piko_symbols)
  --dir <path>         Generated-symbols output directory recorded in
                       the manifest (default: internal/piko_symbols)
  --force              Overwrite an existing manifest
  -h, --help           Show this help message

`)
}
