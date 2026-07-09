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
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	sonicjson "piko.sh/piko/wdk/json/json_provider_sonic"

	"piko.sh/piko/cmd/pikopls/internal/lsp/gopls_bridge"
	"piko.sh/piko/cmd/pikopls/internal/lsp/lsp_adapters"
	"piko.sh/piko/cmd/pikopls/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/wdk/logger"
)

// stdrwc combines stdin and stdout into a single read-write-closer for JSON-RPC streams.
// It implements io.ReadWriteCloser.
type stdrwc struct {
	io.Reader

	io.WriteCloser
}

// deadlineWriteCloser applies a per-write deadline to the stdio output file so a wedged
// editor cannot block the server indefinitely. SetWriteDeadline is best-effort (it
// applies to pollable pipes and is ignored otherwise), so a write to a non-pollable
// target degrades to the prior blocking behaviour rather than erroring.
type deadlineWriteCloser struct {
	// file is the stdio output file the deadline is applied to before each write.
	file *os.File

	// timeout is the per-write deadline; a non-positive value disables the deadline.
	timeout time.Duration
}

const (
	// defaultDriverMode is the default driver mode used when the PIKO_LSP_DRIVER environment
	// variable is not set.
	defaultDriverMode = "stdio"

	// defaultTCPHost is the default host address for TCP connections.
	defaultTCPHost = "127.0.0.1"

	// defaultTCPPort is the default port number for TCP connections.
	defaultTCPPort = 4389

	// defaultPprofPort is the default port for the pprof HTTP server.
	defaultPprofPort = 6060

	// stdioWriteTimeout bounds a single write to the editor over stdio.
	stdioWriteTimeout = 15 * time.Second
)

var (
	// flagTCP holds the --tcp flag that switches from stdio to TCP transport.
	flagTCP = flag.Bool("tcp", false, "Use TCP mode instead of stdio")

	// flagPort holds the --port flag for the TCP listening port.
	flagPort = flag.Int("port", defaultTCPPort, "TCP port to listen on (used with --tcp)")

	// flagHost holds the --host flag for the TCP bind address.
	flagHost = flag.String("host", defaultTCPHost, "TCP host to bind to (used with --tcp)")

	// flagPprof holds the --pprof flag that enables the profiling HTTP server.
	flagPprof = flag.Bool("pprof", false, "Enable pprof profiling server")

	// flagPprofPort holds the --pprof-port flag for the profiling server port.
	flagPprofPort = flag.Int("pprof-port", defaultPprofPort, "Port for the pprof HTTP server (used with --pprof)")

	// flagFormatting holds the --formatting flag that enables document formatting.
	flagFormatting = flag.Bool("formatting", false, "Enable document formatting capabilities")

	// flagFileLogging holds the --file-logging flag that enables logging to a file.
	flagFileLogging = flag.Bool("file-logging", false, "Enable file logging to /tmp/pikopls-<pid>.log")

	// flagGoplsBridge holds the --gopls-bridge flag that enables delegating Go script blocks
	// to a child gopls process. It is off by default; editors opt in (VS Code via
	// initialisationOptions, Zed via this flag).
	flagGoplsBridge = flag.Bool("gopls-bridge", false, "Delegate Go script blocks to a child gopls process for Go intelligence")

	// flagGoplsPath holds the --gopls-path flag overriding gopls discovery.
	flagGoplsPath = flag.String("gopls-path", "", "Path to the gopls executable (defaults to PATH and Go bin discovery)")

	// flagGoplsDisable holds the --gopls-disable flag, the operator's hard veto on the
	// Go-block bridge. When set, no client opt-in can spawn gopls.
	flagGoplsDisable = flag.Bool("gopls-disable", false, "Hard-disable the Go-block gopls bridge regardless of any client request")

	// flagGoplsMaxChildren caps concurrent gopls child processes; 0 keeps the default.
	flagGoplsMaxChildren = flag.Int("gopls-max-children", 0, "Maximum concurrent gopls child processes (0 uses the default); raise it for large multi-module workspaces")

	// flagGoplsMaxOverlays caps open Go-block overlays per gopls child; 0 keeps the default.
	flagGoplsMaxOverlays = flag.Int("gopls-max-overlays", 0, "Maximum open Go-block overlays per gopls child (0 uses the default)")
)

// lspServices holds the services needed for LSP operation.
type lspServices struct {
	// container provides access to dependency-injected services.
	container *bootstrap.Container

	// pathsConfig supplies workspace path settings to the LSP server.
	pathsConfig *config.PathsConfig

	// docCache stores parsed document data for the LSP server.
	docCache *lsp_domain.DocumentCache

	// lspReader provides file system access for the LSP server.
	lspReader annotator_domain.FSReaderPort
}

// main starts the Piko LSP server.
func main() {
	sonicjson.New().Activate()
	flag.Parse()
	validatePortFlags()

	driverMode, tcpAddr := getDriverConfig()

	ctx := context.Background()

	if *flagFileLogging {
		cleanStaleLSPLogs()
		logFile := fmt.Sprintf("/tmp/pikopls-%d.log", os.Getpid())
		logger.AddFileOutputOnly(ctx, "lsp-log", logFile, logger.WithLevel(slog.LevelDebug), logger.WithJSON())
		getLog().Info("Piko LSP server starting...", logger.String("driver", driverMode), logger.String("tcpAddr", tcpAddr), logger.String("logFile", logFile))
	} else {
		getLog().Info("Piko LSP server starting...", logger.String("driver", driverMode), logger.String("tcpAddr", tcpAddr), logger.String("fileLogging", "disabled"))
	}

	if *flagPprof {
		startPprofServer(*flagPprofPort)
	}

	service := initialiseLSP()
	driver := createDriver(driverMode, tcpAddr, service)

	runServer(ctx, driverMode, driver)
}

// startPprofServer starts the pprof HTTP server on the given port. The server provides
// profiling endpoints at /_piko/debug/pprof/*.
//
// Takes port (int) which specifies the port number to listen on.
//
// Runs the server in a separate goroutine. The goroutine runs until the server stops or
// fails.
func startPprofServer(port int) {
	profilerConfig := profiler.Config{
		Port:                 port,
		BindAddress:          profiler.DefaultBindAddress,
		BlockProfileRate:     profiler.DefaultBlockProfileRate,
		MutexProfileFraction: profiler.DefaultMutexProfileFraction,
	}

	profiler.SetRuntimeRates(profilerConfig)

	if warning := profiler.CheckBuildFlags(); warning != "" {
		getLog().Warn(warning)
	}

	server, err := profiler.StartServer(profilerConfig)
	if err != nil {
		getLog().Error("Failed to start pprof server", logger.Error(err))
		return
	}
	server.SetErrorHandler(func(err error) {
		getLog().Error("Pprof server error", logger.Error(err))
	})

	addr := profiler.ServerAddress(profilerConfig)
	getLog().Info("Starting pprof server", logger.String("address", fmt.Sprintf("http://%s%s/debug/pprof/", addr, profiler.BasePath)))
}

// getDriverConfig reads driver configuration from command-line flags first, then falls
// back to environment variables.
//
// Returns driverMode (string) which specifies the LSP driver mode to use.
// Returns tcpAddr (string) which specifies the TCP address for connections.
func getDriverConfig() (driverMode, tcpAddr string) {
	if *flagTCP {
		driverMode = "tcp"
		tcpAddr = net.JoinHostPort(*flagHost, strconv.Itoa(*flagPort))
		return driverMode, tcpAddr
	}

	driverMode = os.Getenv("PIKO_LSP_DRIVER")
	if driverMode == "" {
		driverMode = defaultDriverMode
	}

	tcpAddr = os.Getenv("PIKO_LSP_TCP_ADDR")
	if tcpAddr == "" {
		tcpAddr = net.JoinHostPort(defaultTCPHost, strconv.Itoa(defaultTCPPort))
	}

	return driverMode, tcpAddr
}

// envTruthy reports whether an environment variable is set to a truthy value (1, true,
// yes, or on, case-insensitive).
//
// Takes key (string) which names the environment variable to read.
//
// Returns true when the variable holds a recognised truthy value.
func envTruthy(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return slices.Contains([]string{"1", "true", "yes", "on"}, value)
}

// initialiseLSP bootstraps the DI container and LSP-specific components.
//
// Returns lspServices which contains the initialised container, config provider, document
// cache, and LSP file reader.
func initialiseLSP() lspServices {
	deps := &bootstrap.Dependencies{
		AppRouter: chi.NewRouter(),
	}

	container, err := bootstrap.ConfigAndContainer(context.Background(), deps)
	if err != nil {
		fatalf("Failed to bootstrap container: %v", err)
	}

	container.SetCompilerDebugLogsEnabled(false)

	docCache := lsp_domain.NewDocumentCache()
	osReader := lsp_adapters.NewOsFSReader()
	lspReader, err := lsp_adapters.NewLspFSReader(docCache, osReader)
	if err != nil {
		fatalf("Failed to create LSP file reader: %v", err)
	}

	container.SetCoordinatorDiagnosticOutputOverride(coordinator_adapters.NewSilentDiagnosticOutput())
	container.SetFSReaderOverride(lspReader)
	container.SetRenderRegistryOverride(lsp_adapters.NewNoopRenderRegistry())

	return lspServices{
		container:   container,
		pathsConfig: &container.GetServerConfig().Paths,
		docCache:    docCache,
		lspReader:   lspReader,
	}
}

// createDriver creates the appropriate LSP driver based on the mode.
//
// Takes driverMode (string) which specifies the driver type ("stdio" or "tcp").
// Takes tcpAddr (string) which specifies the TCP address when using TCP mode.
// Takes service (lspServices) which provides the required service dependencies.
//
// Returns lsp_domain.LSPServerPort which is the configured LSP server driver.
func createDriver(driverMode, tcpAddr string, service lspServices) lsp_domain.LSPServerPort {
	coordinatorService, err := service.container.GetCoordinatorService()
	if err != nil {
		fatalf("Failed to get coordinator service: %v", err)
	}
	resolver, err := service.container.GetResolver()
	if err != nil {
		fatalf("Failed to get resolver service: %v", err)
	}
	typeInspectorMgr, err := service.container.GetTypeInspectorManager()
	if err != nil {
		fatalf("Failed to get type inspector manager: %v", err)
	}

	goplsManager, goplsBridgeEnabled := buildGoplsManager()

	switch driverMode {
	case "tcp":
		getLog().Info("Creating TCP driver adapter", logger.String("address", tcpAddr))
		driver, driverErr := lsp_adapters.NewTCPAdapter(lsp_adapters.TCPAdapterDeps{
			Addr:                 tcpAddr,
			CoordinatorService:   coordinatorService,
			Resolver:             resolver,
			TypeInspectorManager: typeInspectorMgr,
			DocCache:             service.docCache,
			LSPReader:            service.lspReader,
			PathsConfig:          service.pathsConfig,
			GoplsManager:         goplsManager,
			GoplsBridgeEnabled:   goplsBridgeEnabled,
			FormattingEnabled:    *flagFormatting,
		})
		if driverErr != nil {
			fatalf("Failed to create TCP driver adapter: %v", driverErr)
		}
		return driver
	case "stdio":
		getLog().Info("Creating STDIO driver adapter")
		driver, driverErr := lsp_adapters.NewStdioAdapter(lsp_adapters.StdioAdapterDeps{
			CoordinatorService:   coordinatorService,
			Resolver:             resolver,
			TypeInspectorManager: typeInspectorMgr,
			DocCache:             service.docCache,
			LSPReader:            service.lspReader,
			PathsConfig:          service.pathsConfig,
			GoplsManager:         goplsManager,
			GoplsBridgeEnabled:   goplsBridgeEnabled,
			FormattingEnabled:    *flagFormatting,
		})
		if driverErr != nil {
			fatalf("Failed to create STDIO driver adapter: %v", driverErr)
		}
		return driver
	default:
		fatalf("Unknown driver mode: %s (must be 'stdio' or 'tcp')", driverMode)
		return nil
	}
}

// buildGoplsManager resolves the gopls bridge configuration and builds the shared
// Manager.
//
// The Manager is always allowed to discover gopls lazily; whether a connection uses the
// bridge is decided per-connection by the returned default (the process --gopls-bridge
// flag, overridable by initializationOptions.goBridge), so a client such as VS Code can
// opt in even when the process default is off, while a process where nobody opts in never
// pays for discovery.
//
// Returns the shared gopls Manager and the process-level bridge default.
func buildGoplsManager() (*gopls_bridge.Manager, bool) {
	goplsBridgeEnabled := *flagGoplsBridge || envTruthy("PIKO_LSP_GOPLS_BRIDGE")
	goplsPath := cmp.Or(*flagGoplsPath, os.Getenv("PIKO_GOPLS_PATH"))

	allow := !*flagGoplsDisable && !envTruthy("PIKO_LSP_GOPLS_DISABLE")
	manager := gopls_bridge.NewManager(gopls_bridge.ManagerConfig{
		GoplsPath:           goplsPath,
		Allow:               allow,
		MaxChildren:         cmp.Or(*flagGoplsMaxChildren, envInt("PIKO_LSP_GOPLS_MAX_CHILDREN")),
		MaxOverlaysPerChild: cmp.Or(*flagGoplsMaxOverlays, envInt("PIKO_LSP_GOPLS_MAX_OVERLAYS")),
	})
	return manager, goplsBridgeEnabled
}

// envInt reads a non-negative integer from an environment variable, returning 0 when it
// is unset, blank, malformed, or negative so the caller falls back to the configured
// default.
//
// Takes key (string) which names the environment variable to read.
//
// Returns the parsed non-negative integer, or 0 when the value is absent or invalid.
func envInt(key string) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0
	}
	parsed, parseErr := strconv.Atoi(value)
	if parseErr != nil || parsed < 0 {
		return 0
	}
	return parsed
}

// newDeadlineWriteCloser wraps the stdio output file with a per-write deadline.
//
// Takes file (*os.File) which is the stdio output file to wrap.
// Takes timeout (time.Duration) which is the per-write deadline.
//
// Returns an io.WriteCloser that applies the deadline before each write.
func newDeadlineWriteCloser(file *os.File, timeout time.Duration) io.WriteCloser {
	return &deadlineWriteCloser{file: file, timeout: timeout}
}

// Write sets the write deadline before delegating to the underlying file.
//
// Takes payload ([]byte) which holds the bytes to write.
//
// Returns the number of bytes written and any write error.
func (d *deadlineWriteCloser) Write(payload []byte) (int, error) {
	if d.timeout > 0 {
		_ = d.file.SetWriteDeadline(time.Now().Add(d.timeout))
	}
	return d.file.Write(payload)
}

// Close closes the underlying file.
//
// Returns any error from closing the file.
func (d *deadlineWriteCloser) Close() error {
	return d.file.Close()
}

// runServer starts the LSP server and handles shutdown.
//
// Takes driverMode (string) which specifies the transport mode (e.g. "stdio").
// Takes driver (lsp_domain.LSPServerPort) which provides the LSP server implementation.
func runServer(ctx context.Context, driverMode string, driver lsp_domain.LSPServerPort) {
	getLog().Info("Starting Piko LSP server...")

	var stream io.ReadWriteCloser
	if driverMode == "stdio" {
		stream = &stdrwc{Reader: os.Stdin, WriteCloser: newDeadlineWriteCloser(os.Stdout, stdioWriteTimeout)}
	}

	if err := driver.Run(ctx, stream); err != nil {
		fatalf("LSP server error: %v", err)
	}

	getLog().Info("Piko LSP server stopped gracefully.")
}

// cleanStaleLSPLogs removes pikopls log files from /tmp that are older than 24 hours as
// best-effort cleanup, silently ignoring errors.
func cleanStaleLSPLogs() {
	const maxAge = 24 * time.Hour
	cutoff := time.Now().Add(-maxAge)

	matches, err := filepath.Glob("/tmp/pikopls-*.log*")
	if err != nil {
		return
	}

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
	}
}

// validatePortFlags rejects out-of-range --port and --pprof-port values up front so a
// misconfiguration fails with a clear message rather than an opaque late net.Listen
// error.
func validatePortFlags() {
	const minPort, maxPort = 1, 65535
	if *flagPort < minPort || *flagPort > maxPort {
		fatalf("invalid --port %d: must be between %d and %d", *flagPort, minPort, maxPort)
	}
	if *flagPprofPort < minPort || *flagPprofPort > maxPort {
		fatalf("invalid --pprof-port %d: must be between %d and %d", *flagPprofPort, minPort, maxPort)
	}
}

// fatalf logs an error message and stops the program with exit code 1.
//
// Takes format (string) which specifies the format string for the message.
// Takes arguments (...any) which provides the values to format.
func fatalf(format string, arguments ...any) {
	message := fmt.Sprintf(format, arguments...)
	getLog().Error(message)
	_, _ = os.Stderr.WriteString(message + "\n")
	os.Exit(1)
}
