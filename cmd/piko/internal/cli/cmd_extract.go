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
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/wdk/interp/interp_piko_symbols"
	"pipit.sh/pipit"
	"pipit.sh/pipit/sdk/extract"
)

// RunExtract runs the `piko extract` command, which generates and checks the symbol
// tables the interpreter (dev-i) uses to import native Go packages.
//
// Takes arguments ([]string) which contains the command-line arguments following
// "extract".
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunExtract(arguments []string) int {
	return RunExtractWithIO(arguments, os.Stdout, os.Stderr)
}

// RunExtractWithIO runs the `piko extract` command with explicit output writers.
//
// The command itself is pipit's; piko supplies its own name, manifest defaults, the .pk
// source scanner and the packages it already ships tables for.
//
// Takes arguments ([]string) which contains the command-line arguments following
// "extract".
// Takes stdout (io.Writer) which receives normal output messages.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunExtractWithIO(arguments []string, stdout, stderr io.Writer) int {
	return extract.RunCommand(context.Background(), arguments, stdout, stderr, pikoExtractConfig())
}

// pikoExtractConfig adapts pipit's extract command to piko projects.
//
// Returns extract.Config which scans the conventional .pk directories and treats the
// standard library and piko's runtime API as already provided.
func pikoExtractConfig() extract.Config {
	return extract.Config{
		AlreadyProvided:     providedSymbolPaths,
		ToolName:            "piko extract",
		DefaultManifest:     "piko-symbols.yaml",
		InitPackage:         "piko_symbols",
		InitDirectory:       "internal/piko_symbols",
		SourceDirs:          []string{"pages", "partials", "components", "emails", "pdfs", "pk", "actions"},
		Scanners:            []extract.SourceScanner{pkScanner{}},
		IgnoreProjectModule: false,
	}
}

// pkScanner reads the Go imports of a .pk component's script block.
type pkScanner struct{}

// Match reports whether name is a .pk component file.
//
// Takes name (string) which is the file's base name.
//
// Returns bool which is true for names ending in .pk.
func (pkScanner) Match(name string) bool {
	return strings.HasSuffix(name, ".pk")
}

// Imports parses a .pk file and returns the imports of its Go script block.
//
// Takes path (string) which is the root-relative path to the .pk file.
// Takes data ([]byte) which is the file content.
//
// Returns []string which contains the Go import paths, or nil when there is no script.
// Returns error when the file cannot be parsed.
func (pkScanner) Imports(ctx context.Context, path string, data []byte) ([]string, error) {
	_, sources, parseErr := annotator_domain.ParsePK(ctx, data, path)
	if parseErr != nil && !annotator_domain.IsParseSoftError(parseErr) {
		return nil, fmt.Errorf("parsing %s: %w", path, parseErr)
	}
	if sources.ScriptSource == "" {
		return nil, nil
	}

	imports, err := extract.ParseGoImports(path, []byte(sources.ScriptSource))
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(imports, isComponentImport), nil
}

// providedSymbolPaths lists the import paths for which piko already ships symbol tables,
// covering the vendored standard library and piko's own runtime API.
//
// Returns []string which are the import paths, in no particular order.
func providedSymbolPaths() []string {
	return slices.Concat(pipit.StandardLibraryPaths(), slices.Collect(maps.Keys(interp_piko_symbols.Symbols)))
}

// isComponentImport reports whether an import path names a .pk component rather than a Go
// package.
//
// Takes importPath (string) which is the import path to check.
//
// Returns bool which is true when the path ends in .pk.
func isComponentImport(importPath string) bool {
	return strings.HasSuffix(importPath, ".pk")
}
