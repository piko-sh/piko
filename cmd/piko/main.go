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

package main

import (
	"fmt"
	"os"

	"piko.sh/piko/cmd/piko/internal/cli"
	"piko.sh/piko/cmd/piko/internal/wizard"
	sonicjson "piko.sh/piko/wdk/json/json_provider_sonic"

	_ "piko.sh/piko/wdk/logger/logger_integration_otel_grpc"
	_ "piko.sh/piko/wdk/logger/logger_integration_otel_http"
	_ "piko.sh/piko/wdk/logger/logger_otel_sdk"
)

// main is the entry point that dispatches subcommands or shows the welcome
// message.
func main() {
	sonicjson.New().Activate()

	if len(os.Args) > 1 {
		subcommand := os.Args[1]

		switch subcommand {
		case "new":
			os.Exit(wizard.Run())
		case "fmt":
			os.Exit(cli.RunFmt(os.Args[2:]))
		case "agents":
			os.Exit(cli.RunAgents(os.Args[2:]))
		case "extract":
			os.Exit(cli.RunExtract(os.Args[2:]))
		case "inspect":
			os.Exit(cli.RunInspect(os.Args[2:]))
		case "bytecode":
			os.Exit(cli.RunBytecode(os.Args[2:]))
		case "profile":
			os.Exit(cli.RunProfile(os.Args[2:]))
		case "get", "describe", "info", "watch", "diagnostics", "tui", "profiling", "watchdog":
			os.Exit(cli.RunCommand(subcommand, os.Args[2:]))
		case "version", "--version":
			fmt.Printf("piko %s\n", Version)
			os.Exit(0)
		case "help", "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			_, _ = fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
			printUsage()
			os.Exit(1)
		}
	}

	printMOTD()
}

// printMOTD writes a friendly welcome message with available commands.
func printMOTD() {
	fmt.Printf(`Piko %s - A framework for building server-rendered web applications in Go.

Getting Started:
  new           Create a new Piko project

Project Commands:
  fmt           Format Piko template files (.pk)
  extract       Extract Go package symbols for the bytecode interpreter
  inspect       Inspect FlatBuffers binary files
  bytecode      Inspect and analyse compiled bytecode files
  agents        Configure AI coding tools with Piko knowledge
  profile       Profile a live server under load (CPU, memory, mutex, blocking)

Monitoring Commands:
  get           Display resources from the monitoring server
  describe      Show detailed information (health, trace, task, workflow, artefact, dlq, resources, ratelimiter)
  info          Display system information (system, build, runtime, memory, gc, process)
  watch         Stream resource updates in real time
  profiling     Control on-demand runtime profiling (enable, disable, status, capture)
  watchdog      Manage watchdog diagnostic profiles (list, download, prune, status)
  diagnostics   Test connectivity to the monitoring server
  tui           Launch the interactive terminal UI

Run 'piko new' to create a project or 'piko help' for full usage details.
`, Version)
}

// printUsage writes detailed command line usage information to standard error.
func printUsage() {
	printUsageHeader()
	printUsageCommands()
	printUsageFlags()
	printUsageExamples()
}

// printUsageHeader writes the usage banner and syntax line.
func printUsageHeader() {
	_, _ = fmt.Fprintf(os.Stderr, `Piko CLI %s - A tool for creating and managing Piko projects

Usage:
  piko [subcommand] [flags]

`, Version)
}

// printUsageCommands writes the categorised command listing.
func printUsageCommands() {
	_, _ = fmt.Fprint(os.Stderr, `Project Commands:
  new           Create a new Piko project (interactive wizard)
  fmt           Format Piko template files (.pk)
  extract       Extract Go package symbols for the bytecode interpreter
  inspect       Inspect FlatBuffers binary files (manifest, i18n, collection, search, bytecode)
  bytecode      Inspect and analyse compiled bytecode files
  agents        Configure AI tools with Piko knowledge (Claude Code, Codex, Cursor, etc.)
  profile       Profile a live server under load (CPU, memory, mutex, blocking)

Monitoring Commands:
  get           Display resources (health, tasks, workflows, artefacts, variants, metrics, traces, resources, dlq, ratelimiter)
  describe      Show detailed information (health, trace, task, workflow, artefact, dlq, resources, ratelimiter)
  info          Display system information (system, build, runtime, memory, gc, process) (CPU, memory, goroutines, GC)
  watch         Stream resource updates in real time
  profiling     Control on-demand runtime profiling (enable, disable, status, capture)
  watchdog      Manage watchdog diagnostic profiles (list, download, prune, status)
  diagnostics   Test connectivity to the monitoring server
  tui           Launch the interactive terminal UI

Other:
  version       Show the Piko CLI version
  help          Show this help message

`)
}

// printUsageFlags writes the global flags section.
func printUsageFlags() {
	_, _ = fmt.Fprint(os.Stderr, `Global Flags (monitoring commands):
  -e, --endpoint    gRPC monitoring server address (default: 127.0.0.1:9091)
  -o, --output      Output format: table, wide, json (default: table)
  -n, --limit       Maximum number of items to return
  -t, --timeout     Connection and request timeout (default: 5s)
      --no-colour   Disable coloured output
      --raw         Disable coloured output (alias for --no-colour)
      --no-headers  Omit table headers from output

`)
}

// printUsageExamples writes the examples and subcommand help hints.
func printUsageExamples() {
	_, _ = fmt.Fprint(os.Stderr, `Examples:
  piko new                                 # Create a new Piko project
  piko fmt -w -r ./components              # Format all .pk files recursively
  piko get health                          # Show health status
  piko get health Liveness                 # Show only Liveness probe
  piko get tasks -n 10                     # Show 10 recent tasks
  piko get health -o wide                  # Show extra columns
  piko get health -o json                  # JSON output
  piko get tasks --no-headers              # Table without headers
  piko describe health                     # Detailed health info
  piko describe health Liveness            # Detail for one probe
  piko describe task <id>                  # Detailed task information
  piko get health --help                   # Resource-specific help
  piko info                                 # Show system info overview
  piko info memory                          # Show detailed memory stats
  piko watch health --interval 2s          # Stream health updates
  piko get dlq                             # Show dispatcher summaries
  piko get dlq email -n 10                 # Show email DLQ entries
  piko diagnostics                         # Test monitoring connectivity
  piko tui                                 # Launch the TUI

  piko agents install                       # Configure AI tools with Piko knowledge
  piko agents                               # Show available agents subcommands
  piko inspect manifest dist/manifest.bin  # Inspect a manifest binary
  piko bytecode inspect dist/pages/x/bytecode-y.bin  # Inspect bytecode

  piko profile http://localhost:8080/      # Profile a live server under load
  piko profile http://localhost:8080/ --focus "render"  # Focus on render functions

  piko profiling enable 30m               # Enable on-demand profiling for 30 minutes
  piko profiling status                   # Check profiling status
  piko profiling capture heap             # Capture a heap profile
  piko profiling capture cpu 30s          # Capture CPU profile for 30 seconds
  piko profiling disable                  # Disable profiling

  piko watchdog list                      # List captured profiles
  piko watchdog download --latest --type heap  # Download latest heap profile
  piko watchdog status                    # Show watchdog configuration
  piko watchdog prune                     # Remove all stored profiles

For subcommand-specific help:
  piko fmt -h
  piko inspect -h
  piko bytecode -h
  piko profile -h
  piko get health --help

`)
}
