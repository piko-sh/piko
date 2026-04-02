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
	"os"
	"slices"
	"strings"

	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safedisk"
)

// monitoringConnection abstracts the gRPC connection so that handlers can be
// tested with mock clients.
type monitoringConnection interface {
	// HealthClient returns the health service client for this connection.
	//
	// Returns pb.HealthServiceClient which provides access to health check operations.
	HealthClient() pb.HealthServiceClient

	// MetricsClient returns the gRPC client for the metrics service.
	//
	// Returns pb.MetricsServiceClient which provides access to metrics operations.
	MetricsClient() pb.MetricsServiceClient

	// OrchestratorClient returns the client for the orchestrator inspector service.
	//
	// Returns pb.OrchestratorInspectorServiceClient which provides access to the
	// orchestrator inspection API.
	OrchestratorClient() pb.OrchestratorInspectorServiceClient

	// RegistryClient returns the registry inspector service client.
	//
	// Returns pb.RegistryInspectorServiceClient which provides access to the
	// registry inspection API.
	RegistryClient() pb.RegistryInspectorServiceClient

	// DispatcherClient returns the client for the dispatcher inspector service.
	//
	// Returns pb.DispatcherInspectorServiceClient which provides access to
	// dispatcher inspection operations.
	DispatcherClient() pb.DispatcherInspectorServiceClient

	// RateLimiterClient returns the rate limiter inspector service client.
	//
	// Returns pb.RateLimiterInspectorServiceClient which provides access to rate
	// limiter inspection operations.
	RateLimiterClient() pb.RateLimiterInspectorServiceClient

	// ProviderInfoClient returns the client for accessing provider information.
	//
	// Returns pb.ProviderInfoServiceClient which provides access to provider info.
	ProviderInfoClient() pb.ProviderInfoServiceClient

	// Close releases any resources held by the source.
	//
	// Returns error when the source cannot be closed cleanly.
	Close() error
}

// CommandContext carries IO writers and shared state through the command chain.
type CommandContext struct {
	// Conn is the gRPC connection to the monitoring server; nil for commands
	// that do not require a connection.
	Conn monitoringConnection

	// Factory creates sandboxes for filesystem access.
	Factory safedisk.Factory

	// Opts holds the parsed global flags.
	Opts *GlobalOptions

	// Stdout is the writer for standard output.
	Stdout io.Writer

	// Stderr is the writer for error output.
	Stderr io.Writer
}

// command represents a single CLI subcommand.
type command struct {
	// run executes the command logic.
	run func(ctx context.Context, cc *CommandContext, arguments []string) error

	// name is the command identifier used in the CLI.
	name string

	// usage is a short usage line shown in help.
	usage string

	// description explains what the command does.
	description string

	// needsConnection indicates whether this command requires a gRPC connection.
	needsConnection bool

	// longRunning indicates the command runs indefinitely until interrupted.
	// When true, the context has no timeout deadline.
	longRunning bool
}

// commands maps command names to their definitions.
var commands = map[string]*command{
	"get":         {name: "get", usage: "piko get <resource> [flags]", description: "Display resources", needsConnection: true, run: runGet},
	"describe":    {name: "describe", usage: "piko describe <resource> [id]", description: "Show detailed information", needsConnection: true, run: runDescribe},
	"info":        {name: "info", usage: "piko info [category] [flags]", description: "Display system information", needsConnection: true, run: runInfo},
	"watch":       {name: "watch", usage: "piko watch <resource> [flags]", description: "Stream resource updates", needsConnection: true, longRunning: true, run: runWatch},
	"diagnostics": {name: "diagnostics", usage: "piko diagnostics [flags]", description: "Test connectivity to monitoring server", needsConnection: false, run: runDiagnosticsCmd},
	"tui":         {name: "tui", usage: "piko tui [flags]", description: "Launch the interactive terminal UI", needsConnection: false, run: runTUICmd},
}

// RunCommand dispatches a CLI subcommand using os.Stdout and os.Stderr.
//
// Takes subcommand (string) which identifies the command to run.
// Takes arguments ([]string) which contains the remaining arguments after the
// subcommand.
//
// Returns int which is the exit code: 0 for success, 1 for errors.
func RunCommand(subcommand string, arguments []string) int {
	return RunCommandWithIO(subcommand, arguments, os.Stdout, os.Stderr)
}

// RunCommandWithIO dispatches a CLI subcommand with explicit IO writers.
//
// Takes subcommand (string) which identifies the command to run.
// Takes arguments ([]string) which contains the remaining arguments after the
// subcommand.
// Takes stdout (io.Writer) which receives standard output.
// Takes stderr (io.Writer) which receives error output.
//
// Returns int which is the exit code: 0 for success, 1 for errors.
func RunCommandWithIO(subcommand string, arguments []string, stdout, stderr io.Writer) int {
	command, ok := commands[subcommand]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "Unknown command: %s\n", subcommand)
		printCommandHelp(stderr)
		return 1
	}

	opts, remaining := parseGlobalFlags(arguments)

	var ctx context.Context
	var cleanup func()
	if command.longRunning {
		var cancel context.CancelCauseFunc
		ctx, cancel = context.WithCancelCause(context.Background())
		cleanup = func() { cancel(errors.New("long-running command finished")) }
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(context.Background(), opts.Timeout,
			fmt.Errorf("command exceeded %s timeout", opts.Timeout))
		cleanup = func() { cancel() }
	}
	defer cleanup()

	_, l := logger.From(ctx, log)

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: creating sandbox factory: %v\n", err)
		return 1
	}

	cc := &CommandContext{
		Factory: factory,
		Opts:    opts,
		Stdout:  stdout,
		Stderr:  stderr,
	}

	if command.needsConnection {
		conn, err := connect(factory, opts)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		defer func() { _ = conn.Close() }()
		cc.Conn = conn
		l.Debug("Connected to monitoring server",
			logger.String("endpoint", opts.Endpoint))
	}

	if err := command.run(ctx, cc, remaining); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

// buildResourceList derives a sorted, comma-separated list of resource names
// from a dispatch map's keys.
//
// Takes m (map[string]V) which is the dispatch map to extract keys from.
//
// Returns string which is the sorted, comma-separated resource names.
func buildResourceList[V any](m map[string]V) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return strings.Join(keys, ", ")
}

// printCommandHelp prints usage information for all available commands.
//
// Takes w (io.Writer) which receives the help text.
func printCommandHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `
Monitoring Commands:
  get           Display resources (%s)
  describe      Show detailed information (%s)
  info          Display system information (system, build, runtime, memory, gc, process)
  watch         Stream resource updates (%s)
  diagnostics   Test connectivity to the monitoring server
  tui           Launch the interactive terminal UI

Global Flags:
  -e, --endpoint    gRPC monitoring server address (default: 127.0.0.1:9091)
  -o, --output      Output format: table, wide, json (default: table)
  -n, --limit       Maximum number of items to return
  -t, --timeout     Connection and request timeout (default: 5s)
      --no-colour   Disable coloured output
      --raw         Disable coloured output (alias for --no-colour)
      --no-headers  Omit table headers from output

Examples:
  piko get health                       # Show health status
  piko get health Liveness              # Show only Liveness probe
  piko get tasks -n 10                  # Show 10 recent tasks
  piko get health -o wide               # Show extra columns
  piko get health -o json               # JSON output
  piko get tasks --no-headers           # Table without headers (for scripting)
  piko describe health                  # Detailed health info for all probes
  piko describe health Liveness         # Detailed info for Liveness only
  piko describe task <id>               # Detailed task information
  piko get health --help                # Resource-specific help
  piko info                             # Show system info overview
  piko info memory                      # Show detailed memory stats
  piko info -o json                     # JSON output for all stats
  piko watch health --interval 2s       # Stream health updates
  piko diagnostics -e localhost:9091    # Test monitoring connectivity

`, getResourceList, describeResourceList, watchResourceList)
}
