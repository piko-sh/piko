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
	"piko.sh/piko/internal/json"
)

// extractDiscoverFlags holds the parsed flags for the discover
// subcommand.
type extractDiscoverFlags struct {
	// root is the project root directory.
	root string

	// outputFormat controls how the discovered packages are rendered:
	// "list", "yaml", or "json".
	outputFormat string

	// ignore is the comma-separated list of additional import paths
	// to exclude from the result.
	ignore string
}

// runExtractDiscover walks the project, computes the transitive
// import graph, filters out already-registered packages, and prints
// the remaining required packages.
//
// Takes arguments ([]string) which contains the subcommand arguments
// (without the leading "discover").
// Takes stdout (io.Writer) which receives the discovered list.
// Takes stderr (io.Writer) which receives warnings and diagnostics.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func runExtractDiscover(arguments []string, stdout, stderr io.Writer) int {
	flags, status := parseExtractDiscoverArgs(arguments, stderr)
	if status != parseOK {
		if status == parseHelp {
			extractDiscoverUsage(stdout)
			return 0
		}
		return 1
	}

	result, err := driver_symbols_extract.Discover(context.Background(), driver_symbols_extract.DiscoverOptions{
		Root:         flags.root,
		ExtraIgnored: splitIgnoreList(flags.ignore),
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	if err := renderDiscoverResult(stdout, stderr, flags.outputFormat, result); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}

// parseExtractDiscoverArgs parses the command-line flags for discover.
//
// Takes arguments ([]string) which is the raw argument list.
// Takes stderr (io.Writer) which receives usage and diagnostics.
//
// Returns the parsed flags and a parseStatus describing the outcome.
func parseExtractDiscoverArgs(arguments []string, stderr io.Writer) (extractDiscoverFlags, parseStatus) {
	flags := extractDiscoverFlags{
		root:         ".",
		outputFormat: "list",
	}

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if !strings.HasPrefix(argument, "-") {
			_, _ = fmt.Fprintf(stderr, "Unexpected argument: %s\n\n", argument)
			extractDiscoverUsage(stderr)
			return flags, parseError
		}

		switch argument {
		case "-h", "--help":
			return flags, parseHelp
		case "--root":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.root = arguments[i]
		case "--output":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.outputFormat = arguments[i]
		case "--ignore":
			if i+1 >= len(arguments) {
				_, _ = fmt.Fprintf(stderr, errFlagRequiresValueFmt, argument)
				return flags, parseError
			}
			i++
			flags.ignore = arguments[i]
		default:
			_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
			extractDiscoverUsage(stderr)
			return flags, parseError
		}
	}

	switch flags.outputFormat {
	case "list", "yaml", "json":
	default:
		_, _ = fmt.Fprintf(stderr, "Unknown output format: %s (expected list, yaml, or json)\n", flags.outputFormat)
		return flags, parseError
	}

	return flags, parseOK
}

// splitIgnoreList turns a comma-separated ignore list into a trimmed
// non-empty slice.
//
// Takes raw (string) which is the comma-separated input.
//
// Returns []string with whitespace trimmed and empty entries removed.
func splitIgnoreList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			trimmed = append(trimmed, part)
		}
	}
	return trimmed
}

// renderDiscoverResult prints the discovered result in the requested
// format. Warnings (cgo, generics) always go to stderr.
//
// Takes stdout (io.Writer) which receives the primary output.
// Takes stderr (io.Writer) which receives warnings.
// Takes format (string) which selects "list", "yaml", or "json".
// Takes result (driver_symbols_extract.DiscoverResult) which is the
// discovery output.
//
// Returns error when rendering fails.
func renderDiscoverResult(stdout, stderr io.Writer, format string, result driver_symbols_extract.DiscoverResult) error {
	switch format {
	case "list":
		for _, path := range result.RequiredImports {
			_, _ = fmt.Fprintln(stdout, path)
		}
	case "yaml":
		_, _ = fmt.Fprintln(stdout, "packages:")
		for _, path := range result.RequiredImports {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", path)
		}
	case "json":
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding json: %w", err)
		}
		if _, err := stdout.Write(encoded); err != nil {
			return fmt.Errorf("writing json: %w", err)
		}
		_, _ = fmt.Fprintln(stdout)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}

	for _, path := range result.SkippedCgo {
		_, _ = fmt.Fprintf(stderr, "warning: %s uses cgo and cannot be interpreted\n", path)
	}
	for _, path := range result.GenericCandidates {
		_, _ = fmt.Fprintf(stderr, "warning: %s exports generic types; add a manual generic: block in piko-symbols.yaml\n", path)
	}
	return nil
}

// extractDiscoverUsage writes the usage information for the discover
// subcommand.
//
// Takes w (io.Writer) which receives the usage text.
func extractDiscoverUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko extract discover [flags]

Walk the project's .pk files and Go source, compute the transitive
import closure, and print the third-party packages that need to be
registered with the interpreter.

Already-registered packages (Go standard library, piko-native types,
the project's own module) are filtered out automatically.

Flags:
  --root <dir>         Project root directory (default: .)
  --output <format>    Output format: list, yaml, or json (default: list)
  --ignore <paths>     Comma-separated import paths to exclude
  -h, --help           Show this help message

`)
}
