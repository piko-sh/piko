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
	"os"
	"slices"
	"strings"
)

// parseStatus discriminates between successful parse, help request, and
// error so each subcommand can map them to the right process exit code.
type parseStatus int

const (
	// parseOK means parsing completed successfully and the command
	// should run.
	parseOK parseStatus = iota

	// parseHelp means the user asked for help and the command should
	// exit with status 0 without running.
	parseHelp

	// parseError means parsing failed; the caller should exit with a
	// non-zero status.
	parseError
)

// errFlagRequiresValueFmt is the format string used when a required
// flag value is missing from the command line.
const errFlagRequiresValueFmt = "%s requires a value\n"

// extractSubcommandHandler is the contract implemented by every
// `piko extract` subcommand: parse its own flags, do its job, and
// return a process exit code.
type extractSubcommandHandler func(arguments []string, stdout, stderr io.Writer) int

// extractSubcommands maps the subcommand name to its handler. Adding
// a new subcommand is a matter of implementing a handler with the
// extractSubcommandHandler signature and registering it here.
var extractSubcommands = map[string]extractSubcommandHandler{
	"generate": runExtractGenerate,
	"discover": runExtractDiscover,
	"init":     runExtractInit,
	"check":    runExtractCheck,
}

// RunExtract runs the `piko extract` command, dispatching to the named
// subcommand. With no arguments it prints the help and exits 0 so
// users can discover the available subcommands.
//
// Takes arguments ([]string) which contains the command-line arguments
// following "extract".
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunExtract(arguments []string) int {
	return RunExtractWithIO(arguments, os.Stdout, os.Stderr)
}

// RunExtractWithIO runs the `piko extract` command with explicit output
// writers, which makes it straightforward to test and embed the
// dispatcher in other contexts.
//
// Takes arguments ([]string) which contains the command-line arguments
// following "extract".
// Takes stdout (io.Writer) which receives normal output messages.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunExtractWithIO(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		extractUsage(stdout)
		return 0
	}

	if arguments[0] == "-h" || arguments[0] == "--help" {
		extractUsage(stdout)
		return 0
	}

	handler, ok := extractSubcommands[arguments[0]]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "Unknown subcommand: %s\n\n", arguments[0])
		extractUsage(stderr)
		return 1
	}
	return handler(arguments[1:], stdout, stderr)
}

// extractUsage writes the usage information for the extract command and
// lists the available subcommands in a stable order.
//
// Takes w (io.Writer) which receives the usage text.
func extractUsage(w io.Writer) {
	var builder strings.Builder
	builder.WriteString(`Usage: piko extract <subcommand> [flags]

Work with the symbol registry that the interpreter (dev-i) uses to
resolve external package imports.

Subcommands:
`)

	names := make([]string, 0, len(extractSubcommands))
	for name := range extractSubcommands {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		builder.WriteString("  ")
		builder.WriteString(name)
		builder.WriteString("    ")
		builder.WriteString(extractSubcommandSummary(name))
		builder.WriteString("\n")
	}

	builder.WriteString(`
Run "piko extract <subcommand> --help" for details on any subcommand.
`)

	_, _ = fmt.Fprint(w, builder.String())
}

// extractSubcommandSummary returns a one-line summary for use in the
// parent `piko extract` help output.
//
// Takes name (string) which is the subcommand name.
//
// Returns string describing the subcommand briefly.
func extractSubcommandSummary(name string) string {
	switch name {
	case "generate":
		return "Generate reflect-based symbol files from piko-symbols.yaml"
	case "discover":
		return "Walk the project and report packages that need registering"
	case "init":
		return "Create a piko-symbols.yaml by discovering project imports"
	case "check":
		return "Verify piko-symbols.yaml against the current project imports"
	default:
		return ""
	}
}
