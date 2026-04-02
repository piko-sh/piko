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
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	sonicjson "piko.sh/piko/wdk/json/json_provider_sonic"

	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_adapters"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/wdk/logger"
)

// stdrwc combines stdin and stdout into a single read-write-closer for
// JSON-RPC streams. It implements io.ReadWriteCloser.
type stdrwc struct {
	io.Reader

	io.WriteCloser
}

const (
	// defaultDriverMode is the default driver mode used when the PIKO_LSP_DRIVER
	// environment variable is not set.
	defaultDriverMode = "stdio"

	// defaultTCPHost is the default host address for TCP connections.
	defaultTCPHost = "127.0.0.1"

	// defaultTCPPort is the default port number for TCP connections.
	defaultTCPPort = 4389

	// defaultPprofPort is the default port for the pprof HTTP server.
	defaultPprofPort = 6060
)

var (
	flagTCP = flag.Bool("tcp", false, "Use TCP mode instead of stdio")

	flagPort = flag.Int("port", defaultTCPPort, "TCP port to listen on (used with --tcp)")

	flagHost = flag.String("host", defaultTCPHost, "TCP host to bind to (used with --tcp)")

	flagPprof = flag.Bool("pprof", false, "Enable pprof profiling server")

	flagPprofPort = flag.Int("pprof-port", defaultPprofPort, "Port for the pprof HTTP server (used with --pprof)")

	flagFormatting = flag.Bool("formatting", false, "Enable document formatting capabilities")

	flagFileLogging = flag.Bool("file-logging", false, "Enable file logging to /tmp/piko-lsp-<pid>.log")
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

	driverMode, tcpAddr := getDriverConfig()

	ctx := context.Background()

	if *flagFileLogging {
		cleanStaleLSPLogs()
		logFile := fmt.Sprintf("/tmp/piko-lsp-%d.log", os.Getpid())
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

// startPprofServer starts the pprof HTTP server on the given port.
// The server provides profiling endpoints at /_piko/debug/pprof/*.
//
// Takes port (int) which specifies the port number to listen on.
//
// Runs the server in a separate goroutine. The goroutine runs until the
// server stops or fails.
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

// getDriverConfig reads driver configuration from command-line flags first,
// then falls back to environment variables.
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

// initialiseLSP bootstraps the DI container and LSP-specific components.
//
// Returns lspServices which contains the initialised container, config
// provider, document cache, and LSP file reader.
func initialiseLSP() lspServices {
	configProvider := config.NewConfigProvider()
	deps := &bootstrap.Dependencies{
		ConfigProvider: configProvider,
		AppRouter:      chi.NewRouter(),
	}

	container, err := bootstrap.ConfigAndContainer(context.Background(), deps)
	if err != nil {
		fatalf("Failed to bootstrap container: %v", err)
	}

	container.SetCompilerDebugLogsEnabled(false)

	docCache := lsp_domain.NewDocumentCache()
	osReader := lsp_adapters.NewOsFSReader()
	lspReader := lsp_adapters.NewLspFSReader(docCache, osReader)

	container.SetCoordinatorDiagnosticOutputOverride(coordinator_adapters.NewSilentDiagnosticOutput())
	container.SetFSReaderOverride(lspReader)
	container.SetRenderRegistryOverride(lsp_adapters.NewNoopRenderRegistry())

	return lspServices{
		container:   container,
		pathsConfig: &configProvider.ServerConfig.Paths,
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

	switch driverMode {
	case "tcp":
		getLog().Info("Creating TCP driver adapter", logger.String("address", tcpAddr))
		return lsp_adapters.NewTCPAdapter(lsp_adapters.TCPAdapterDeps{
			Addr:                 tcpAddr,
			CoordinatorService:   coordinatorService,
			Resolver:             resolver,
			TypeInspectorManager: typeInspectorMgr,
			DocCache:             service.docCache,
			LSPReader:            service.lspReader,
			PathsConfig:          service.pathsConfig,
			FormattingEnabled:    *flagFormatting,
		})
	case "stdio":
		getLog().Info("Creating STDIO driver adapter")
		return lsp_adapters.NewStdioAdapter(coordinatorService, resolver, typeInspectorMgr, service.docCache, service.lspReader, service.pathsConfig, *flagFormatting)
	default:
		fatalf("Unknown driver mode: %s (must be 'stdio' or 'tcp')", driverMode)
		return nil
	}
}

// runServer starts the LSP server and handles shutdown.
//
// Takes driverMode (string) which specifies the transport mode (e.g. "stdio").
// Takes driver (lsp_domain.LSPServerPort) which provides the LSP server
// implementation.
func runServer(ctx context.Context, driverMode string, driver lsp_domain.LSPServerPort) {
	getLog().Info("Starting Piko LSP server...")

	var stream io.ReadWriteCloser
	if driverMode == "stdio" {
		stream = &stdrwc{Reader: os.Stdin, WriteCloser: os.Stdout}
	}

	if err := driver.Run(ctx, stream); err != nil {
		fatalf("LSP server error: %v", err)
	}

	getLog().Info("Piko LSP server stopped gracefully.")
}

// cleanStaleLSPLogs removes piko-lsp log files from /tmp that are
// older than 24 hours as best-effort cleanup, silently ignoring
// errors.
func cleanStaleLSPLogs() {
	const maxAge = 24 * time.Hour
	cutoff := time.Now().Add(-maxAge)

	matches, err := filepath.Glob("/tmp/piko-lsp-*.log*")
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
