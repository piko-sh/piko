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
	"flag"
	"strings"
	"time"
)

const (
	// defaultEndpoint is the default gRPC monitoring server address.
	defaultEndpoint = "127.0.0.1:9091"

	// defaultTimeout is the default connection and request timeout.
	defaultTimeout = 5 * time.Second

	// defaultOutputFormat is the default output format.
	defaultOutputFormat = "table"
)

// GlobalOptions holds command-line flags shared across all CLI commands.
type GlobalOptions struct {
	// Endpoint is the gRPC monitoring server address.
	Endpoint string

	// Output is the output format: "table", "wide", or "json".
	Output string

	// CertsDir is the path to a certificate directory for TLS connections.
	// When set, expects an opinionated layout: ca.pem (required), and
	// optionally client.pem + client-key.pem for mTLS.
	CertsDir string

	// Timeout is the connection and request timeout.
	Timeout time.Duration

	// Limit is the maximum number of items to return. Zero means the handler
	// should use its own default.
	Limit int

	// NoColour disables coloured output.
	NoColour bool

	// NoHeaders omits table headers from output.
	NoHeaders bool
}

var (
	// globalValueFlags lists global flags that take a value argument.
	globalValueFlags = map[string]bool{
		"-e": true, "--endpoint": true,
		"-o": true, "--output": true,
		"-t": true, "--timeout": true,
		"-n": true, "--limit": true,
		"--certs": true,
	}

	// globalBoolFlags lists global flags that are boolean (no value argument).
	globalBoolFlags = map[string]bool{
		"--no-colour":  true,
		"--raw":        true,
		"--no-headers": true,
	}
)

// parseGlobalFlags parses global flags from the argument list, returning
// the options and the remaining unparsed arguments. Global flags may appear
// anywhere in the argument list, interspersed with positional arguments.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
//
// Returns *GlobalOptions which holds the parsed flag values.
// Returns []string which contains the remaining positional arguments.
func parseGlobalFlags(arguments []string) (*GlobalOptions, []string) {
	globalArgs, remaining := separateGlobalFlags(arguments)

	opts := &GlobalOptions{}
	fs := flag.NewFlagSet("piko", flag.ContinueOnError)

	fs.StringVar(&opts.Endpoint, "endpoint", defaultEndpoint, "gRPC monitoring server address")
	fs.StringVar(&opts.Endpoint, "e", defaultEndpoint, "gRPC monitoring server address (shorthand)")
	fs.StringVar(&opts.Output, "output", defaultOutputFormat, "Output format: table, wide, json")
	fs.StringVar(&opts.Output, "o", defaultOutputFormat, "Output format (shorthand)")
	fs.DurationVar(&opts.Timeout, "timeout", defaultTimeout, "Connection and request timeout")
	fs.DurationVar(&opts.Timeout, "t", defaultTimeout, "Connection and request timeout (shorthand)")
	fs.BoolVar(&opts.NoColour, "no-colour", false, "Disable coloured output")
	fs.BoolVar(&opts.NoColour, "raw", false, "Disable coloured output (alias for --no-colour)")
	fs.BoolVar(&opts.NoHeaders, "no-headers", false, "Omit table headers from output")
	fs.IntVar(&opts.Limit, "limit", 0, "Maximum number of items to return")
	fs.IntVar(&opts.Limit, "n", 0, "Maximum number of items to return (shorthand)")
	fs.StringVar(&opts.CertsDir, "certs", "", "Path to certificate directory for TLS")

	if err := fs.Parse(globalArgs); err != nil {
		return opts, remaining
	}

	if opts.Limit < 0 {
		opts.Limit = 0
	}

	return opts, remaining
}

// separateGlobalFlags extracts known global flags from anywhere in arguments.
// Flags can therefore be interspersed with positional arguments.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
//
// Returns globalArgs ([]string) which contains only the global flag arguments.
// Returns remaining ([]string) which contains non-global arguments.
func separateGlobalFlags(arguments []string) (globalArgs, remaining []string) {
	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]

		if isGlobalEqualsFlag(argument) {
			globalArgs = append(globalArgs, argument)
			continue
		}

		if globalValueFlags[argument] {
			globalArgs = append(globalArgs, argument)
			if i+1 < len(arguments) {
				i++
				globalArgs = append(globalArgs, arguments[i])
			}
			continue
		}

		if globalBoolFlags[argument] {
			globalArgs = append(globalArgs, argument)
			continue
		}

		remaining = append(remaining, argument)
	}
	return globalArgs, remaining
}

// isGlobalEqualsFlag reports whether arg is a --flag=value form of a known
// global flag.
//
// Takes arg (string) which is a single command-line argument.
//
// Returns bool which is true when the argument matches a global flag in
// equals form.
func isGlobalEqualsFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}
	eqIndex := strings.Index(arg, "=")
	if eqIndex <= 0 {
		return false
	}
	prefix := arg[:eqIndex]
	return globalValueFlags[prefix] || globalBoolFlags[prefix]
}
