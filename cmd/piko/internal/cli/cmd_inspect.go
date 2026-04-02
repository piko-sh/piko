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
	"path/filepath"
	"strings"

	"piko.sh/piko/wdk/json"

	"piko.sh/piko/internal/interp/interp_schema"
	fbsCollection "piko.sh/piko/wdk/fbs/collection"
	fbsI18n "piko.sh/piko/wdk/fbs/i18n"
	fbsManifest "piko.sh/piko/wdk/fbs/manifest"
	fbsSearch "piko.sh/piko/wdk/fbs/search"
	"piko.sh/piko/wdk/safedisk"
)

// inspectHandler reads raw binary payload bytes and returns a
// JSON-serialisable value.
type inspectHandler struct {
	// unpack strips the version header and validates the schema hash.
	// For non-FBS types, this may be a no-op pass-through.
	unpack func(data []byte) ([]byte, error)

	// convert parses the raw payload into a JSON-serialisable struct.
	convert func(payload []byte) (any, error)
}

// inspectHandlers maps type names to their unpack+convert handlers.
var inspectHandlers = map[string]inspectHandler{
	"manifest": {
		unpack: fbsManifest.Unpack,
		convert: func(p []byte) (any, error) {
			return fbsManifest.ConvertManifest(p)
		},
	},
	"i18n": {
		unpack: fbsI18n.Unpack,
		convert: func(p []byte) (any, error) {
			return fbsI18n.ConvertI18n(p)
		},
	},
	"collection": {
		unpack: fbsCollection.Unpack,
		convert: func(p []byte) (any, error) {
			return fbsCollection.ConvertCollection(p)
		},
	},
	"search": {
		unpack: fbsSearch.Unpack,
		convert: func(p []byte) (any, error) {
			return fbsSearch.ConvertSearchIndex(p)
		},
	},
	"bytecode": {
		unpack:  interp_schema.Unpack,
		convert: func(p []byte) (any, error) { return interp_schema.ConvertBytecode(p) },
	},
	"wal": {
		unpack:  func(data []byte) ([]byte, error) { return data, nil },
		convert: parseWALEntries,
	},
}

// inspectFlags holds the parsed flags for the inspect subcommand.
type inspectFlags struct {
	// compact enables compact JSON output instead of pretty-printing.
	compact bool

	// effective shows only the final state per key after WAL replay.
	effective bool

	// parseValues parses JSON string values into native JSON objects.
	parseValues bool
}

// RunInspect runs the inspect subcommand, writing to os.Stdout and os.Stderr.
//
// Takes arguments ([]string) which contains the command-line arguments.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunInspect(arguments []string) int {
	return RunInspectWithIO(arguments, os.Stdout, os.Stderr)
}

// RunInspectWithIO runs the inspect subcommand with explicit output writers.
//
// Usage: piko inspect <type> <file> [flags]
//
// Flags may appear before, between, or after the positional arguments.
// The --effective flag is only valid for the wal type.
//
// Takes arguments ([]string) which contains the command-line arguments.
// Takes stdout (io.Writer) which receives the JSON output.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunInspectWithIO(arguments []string, stdout, stderr io.Writer) int {
	flags, positional, ok := parseInspectArgs(arguments, stderr)
	if !ok {
		return 1
	}

	typeName, handler, exitCode := resolveInspectHandler(positional, flags, stderr)
	if exitCode != 0 {
		return exitCode
	}

	parentDir := filepath.Dir(positional[1])

	factory, err := safedisk.NewCLIFactory(parentDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error creating sandbox factory: %v\n", err)
		return 1
	}

	result, exitCode := loadAndConvert(factory, typeName, positional[1], handler, stderr)
	if exitCode != 0 {
		return exitCode
	}

	result = applyWALTransforms(result, flags)
	return marshalAndPrint(result, flags, stdout, stderr)
}

// parseInspectArgs separates --name boolean flags from positional
// arguments, where flags may appear in any position.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
// Takes stderr (io.Writer) which receives error messages for unknown flags.
//
// Returns inspectFlags which contains the parsed flag values.
// Returns []string which contains the positional (non-flag) arguments.
// Returns bool which is false when an unknown flag was encountered.
func parseInspectArgs(arguments []string, stderr io.Writer) (inspectFlags, []string, bool) {
	var flags inspectFlags
	var positional []string

	for _, argument := range arguments {
		if strings.HasPrefix(argument, "-") {
			switch argument {
			case "--compact":
				flags.compact = true
			case "--effective":
				flags.effective = true
			case "--parse-values":
				flags.parseValues = true
			default:
				_, _ = fmt.Fprintf(stderr, "Unknown flag: %s\n\n", argument)
				inspectUsage(stderr)
				return flags, nil, false
			}
		} else {
			positional = append(positional, argument)
		}
	}

	return flags, positional, true
}

// resolveInspectHandler validates positional arguments, locates the handler for
// the given type, and checks WAL-only flag constraints.
//
// Takes positional ([]string) which contains the type name and file path.
// Takes flags (inspectFlags) which holds parsed flag values.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns string which is the resolved type name.
// Returns inspectHandler which is the handler for the type.
// Returns int which is 0 on success or 1 on validation failure.
func resolveInspectHandler(positional []string, flags inspectFlags, stderr io.Writer) (string, inspectHandler, int) {
	const minPositionalArgs = 2
	if len(positional) < minPositionalArgs {
		inspectUsage(stderr)
		return "", inspectHandler{}, 1
	}

	typeName := positional[0]
	handler, found := inspectHandlers[typeName]
	if !found {
		_, _ = fmt.Fprintf(stderr, "Unknown type: %s\n\n", typeName)
		inspectUsage(stderr)
		return "", inspectHandler{}, 1
	}

	if flags.effective && typeName != "wal" {
		_, _ = fmt.Fprint(stderr, "--effective is only valid for the wal type\n\n")
		inspectUsage(stderr)
		return "", inspectHandler{}, 1
	}
	if flags.parseValues && typeName != "wal" {
		_, _ = fmt.Fprint(stderr, "--parse-values is only valid for the wal type\n\n")
		inspectUsage(stderr)
		return "", inspectHandler{}, 1
	}

	return typeName, handler, 0
}

// loadAndConvert reads the file, unpacks the binary payload, and converts it
// to a JSON-serialisable value.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
// Takes typeName (string) which identifies the file type for error messages.
// Takes filePath (string) which is the path to the file to inspect.
// Takes handler (inspectHandler) which provides unpack and convert functions.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns any which is the converted result.
// Returns int which is 0 on success or 1 on failure.
func loadAndConvert(factory safedisk.Factory, typeName, filePath string, handler inspectHandler, stderr io.Writer) (any, int) {
	parentDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	sandbox, err := factory.Create("inspect-manifest", parentDir, safedisk.ModeReadOnly)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error opening directory: %v\n", err)
		return nil, 1
	}
	data, err := sandbox.ReadFile(fileName)
	_ = sandbox.Close()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error reading file: %v\n", err)
		return nil, 1
	}

	payload, err := handler.unpack(data)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error unpacking %s: %v\n", typeName, err)
		return nil, 1
	}

	result, err := handler.convert(payload)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error converting %s: %v\n", typeName, err)
		return nil, 1
	}

	return result, 0
}

// applyWALTransforms applies WAL-specific transformations (effective state,
// value parsing) to the inspect result when the corresponding flags are set.
//
// Takes result (any) which is the raw converted result.
// Takes flags (inspectFlags) which controls which transforms to apply.
//
// Returns any which is the transformed result.
func applyWALTransforms(result any, flags inspectFlags) any {
	if flags.effective {
		result = effectiveWALResult(result.(walInspectResult))
	}
	if flags.parseValues {
		result = parseWALValues(result.(walInspectResult))
	}
	return result
}

// marshalAndPrint serialises the result to JSON and writes it to stdout.
//
// Takes result (any) which is the value to serialise.
// Takes flags (inspectFlags) which controls compact vs pretty output.
// Takes stdout (io.Writer) which receives the JSON output.
// Takes stderr (io.Writer) which receives error messages.
//
// Returns int which is 0 on success or 1 on marshalling failure.
func marshalAndPrint(result any, flags inspectFlags, stdout, stderr io.Writer) int {
	var output []byte
	var err error
	if flags.compact {
		output, err = json.Marshal(result)
	} else {
		output, err = json.MarshalIndent(result, "", "  ")
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error marshalling JSON: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, string(output))
	return 0
}

// inspectUsage prints the command-line help text for inspect.
//
// Takes w (io.Writer) which receives the usage text.
func inspectUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `piko inspect reads binary files and outputs their contents as JSON.

Usage:
  piko inspect <type> <file> [flags]

Types:
  manifest     Inspect a manifest.bin file
  i18n         Inspect an i18n.bin file
  collection   Inspect a collection data.bin file
  search       Inspect a search index .bin file
  bytecode     Inspect a compiled bytecode .bin file
  wal          Inspect a cache WAL (.wal) file

Flags:
  --compact       Compact JSON output (default: pretty-printed)
  --effective     WAL only: show only the final state per key after replay
  --parse-values  WAL only: parse JSON string values into native JSON objects

Examples:
  piko inspect manifest dist/manifest.bin
  piko inspect i18n dist/i18n.bin
  piko inspect collection dist/collections/docs/data.bin
  piko inspect search dist/collections/docs/search.bin --compact
  piko inspect bytecode dist/pages/page_abc123/bytecode-def456.bin
  piko inspect wal .piko/wal/data.wal
  piko inspect wal .piko/wal/data.wal --effective
  piko inspect wal .piko/wal/data.wal --effective --parse-values

`)
}
