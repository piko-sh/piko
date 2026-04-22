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
	"strings"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
)

// extractCheckFlags holds the parsed flags for the check subcommand.
type extractCheckFlags struct {
	// manifest is the piko-symbols.yaml path to validate.
	manifest string

	// root is the project root directory to discover from.
	root string
}

// runExtractCheck verifies that the current manifest covers every
// required third-party import for the project. Missing entries
// produce exit code 1 so this command can gate CI.
//
// Takes arguments ([]string) which contains the subcommand arguments.
// Takes stdout (io.Writer) which receives summary output.
// Takes stderr (io.Writer) which receives drift reports.
//
// Returns int which is the exit code: 0 when aligned, 1 on drift or error.
func runExtractCheck(arguments []string, stdout, stderr io.Writer) int {
	flags, status := parseExtractCheckArgs(arguments, stderr)
	if status != parseOK {
		if status == parseHelp {
			extractCheckUsage(stdout)
			return 0
		}
		return 1
	}

	manifest, err := driver_symbols_extract.LoadManifest(flags.manifest)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error loading manifest: %s\n", err)
		return 1
	}

	result, err := driver_symbols_extract.Discover(context.Background(), driver_symbols_extract.DiscoverOptions{
		Root: flags.root,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error discovering project: %s\n", err)
		return 1
	}

	return reportCheckResult(stdout, stderr, flags.manifest, manifest, result)
}

// reportCheckResult computes the manifest-vs-project diff, surfaces
// warnings, and returns the right exit code. Splitting this out of
// runExtractCheck keeps the reporting logic testable with synthesised
// manifests and DiscoverResults, no filesystem or packages.Load
// involvement required.
//
// Takes stdout (io.Writer) which receives the up-to-date summary on
// success.
// Takes stderr (io.Writer) which receives warnings and the missing
// package list on drift.
// Takes manifestPath (string) which is the path shown in messages.
// Takes manifest (*driver_symbols_extract.Manifest) which is the
// declared set.
// Takes result (driver_symbols_extract.DiscoverResult) which is the
// discovered required set.
//
// Returns int which is the exit code: 0 when the manifest covers
// every required package, 1 when packages are missing.
func reportCheckResult(
	stdout, stderr io.Writer,
	manifestPath string,
	manifest *driver_symbols_extract.Manifest,
	result driver_symbols_extract.DiscoverResult,
) int {
	diff := driver_symbols_extract.Diff(manifest, result)

	for _, path := range result.SkippedCgo {
		_, _ = fmt.Fprintf(stderr, "warning: %s uses cgo and cannot be interpreted\n", path)
	}
	for _, path := range result.GenericCandidates {
		_, _ = fmt.Fprintf(stderr, "warning: %s exports generic types; a manual generic: block is required\n", path)
	}
	for _, path := range diff.Unused {
		_, _ = fmt.Fprintf(stderr, "warning: %s is declared in %s but not used by the project\n", path, manifestPath)
	}

	if len(diff.Missing) == 0 {
		_, _ = fmt.Fprintf(stdout, "%s is up-to-date (%d package(s)).\n", manifestPath, len(manifest.Packages))
		return 0
	}

	_, _ = fmt.Fprintf(stderr, "\n%s is missing %d package(s):\n", manifestPath, len(diff.Missing))
	for _, path := range diff.Missing {
		_, _ = fmt.Fprintf(stderr, "  - %s\n", path)
	}
	_, _ = fmt.Fprintf(stderr, "\nhint: add the missing paths to %s and run \"piko extract generate\"\n", manifestPath)
	return 1
}

// parseExtractCheckArgs parses the command-line flags for check.
//
// Takes arguments ([]string) which is the raw argument list.
// Takes stderr (io.Writer) which receives usage and diagnostics.
//
// Returns the parsed flags and a parseStatus describing the outcome.
func parseExtractCheckArgs(arguments []string, stderr io.Writer) (extractCheckFlags, parseStatus) {
	flags := extractCheckFlags{
		manifest: "piko-symbols.yaml",
		root:     ".",
	}

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if !strings.HasPrefix(argument, "-") {
			_, _ = fmt.Fprintf(stderr, "Unexpected argument: %s\n\n", argument)
			extractCheckUsage(stderr)
			return flags, parseError
		}

		switch argument {
		case "-h", "--help":
			return flags, parseHelp
		case "--manifest":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.manifest = arguments[i]
		case "--root":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.root = arguments[i]
		default:
			_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
			extractCheckUsage(stderr)
			return flags, parseError
		}
	}

	return flags, parseOK
}

// extractCheckUsage writes the usage information for the check subcommand.
//
// Takes w (io.Writer) which receives the usage text.
func extractCheckUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko extract check [flags]

Compare piko-symbols.yaml against the current project and exit
non-zero when the manifest is missing required packages.

Intended for CI: a green check means "piko extract generate" will
produce a registry that matches the project sources. Manifest
entries that aren't required by the project are reported as
warnings but do not fail the check.

Flags:
  --manifest <path>    Path to YAML manifest (default: piko-symbols.yaml)
  --root <dir>         Project root directory (default: .)
  -h, --help           Show this help message

`)
}
