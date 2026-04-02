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

package tui

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"piko.sh/piko/cmd/piko/internal/tui/tui_adapters/provider_grpc"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safedisk"
)

// Re-export domain types for use in custom panels and providers.
type (
	// Panel is a discrete UI section that can be focused and rendered.
	Panel = tui_domain.Panel

	// KeyBinding describes a key and its action for help display.
	KeyBinding = tui_domain.KeyBinding

	// Provider is the base interface for all data providers.
	Provider = tui_domain.Provider

	// RefreshableProvider supports periodic data refresh.
	RefreshableProvider = tui_domain.RefreshableProvider

	// MetricsProvider fetches metrics data.
	MetricsProvider = tui_domain.MetricsProvider

	// TracesProvider fetches trace data.
	TracesProvider = tui_domain.TracesProvider

	// ResourceProvider fetches application resource data.
	ResourceProvider = tui_domain.ResourceProvider

	// HealthProvider fetches health check data.
	HealthProvider = tui_domain.HealthProvider

	// SystemProvider fetches system stats data.
	SystemProvider = tui_domain.SystemProvider

	// FDsProvider fetches file descriptor information.
	FDsProvider = tui_domain.FDsProvider

	// SystemStats holds runtime system statistics.
	SystemStats = tui_domain.SystemStats

	// MetricValue represents a single metric data point.
	MetricValue = tui_domain.MetricValue

	// MetricSeries represents a time series of metric values.
	MetricSeries = tui_domain.MetricSeries

	// Span represents a single trace span.
	Span = tui_domain.Span

	// SpanStatus represents the status of a trace span.
	SpanStatus = tui_domain.SpanStatus

	// Resource represents a generic application resource.
	Resource = tui_domain.Resource

	// ResourceStatus represents the health status of a resource.
	ResourceStatus = tui_domain.ResourceStatus
)

const (
	// SpanStatusUnset is the default span status indicating no status has been set.
	SpanStatusUnset = tui_domain.SpanStatusUnset

	// SpanStatusOK is the status code for a successful span operation.
	SpanStatusOK = tui_domain.SpanStatusOK

	// SpanStatusError indicates that the operation ended with an error.
	SpanStatusError = tui_domain.SpanStatusError

	// ResourceStatusUnknown represents an unknown or uninitialized resource status.
	ResourceStatusUnknown = tui_domain.ResourceStatusUnknown

	// ResourceStatusHealthy represents a resource that is operating normally.
	ResourceStatusHealthy = tui_domain.ResourceStatusHealthy

	// ResourceStatusDegraded indicates a resource is operating with reduced
	// capability or performance.
	ResourceStatusDegraded = tui_domain.ResourceStatusDegraded

	// ResourceStatusUnhealthy indicates the resource is not functioning correctly.
	ResourceStatusUnhealthy = tui_domain.ResourceStatusUnhealthy

	// ResourceStatusPending is the status for resources awaiting processing.
	ResourceStatusPending = tui_domain.ResourceStatusPending
)

// TUI provides the main entry point to the terminal monitoring interface.
// It implements io.Closer and MCPServerPort.
type TUI struct {
	// config holds the TUI configuration settings.
	config *tui_dto.Config

	// providers holds the dependency injection container for TUI services.
	providers *tui_domain.Providers

	// service delegates TUI operations to the domain service.
	service *tui_domain.Service
}

var log = logger.GetLogger("tui")

// Run starts the TUI and blocks until the user exits or an error occurs.
// The context can be used for cancellation.
//
// Returns error when the TUI encounters a fatal error.
func (t *TUI) Run(ctx context.Context) error {
	return t.service.Run(ctx)
}

// Close releases resources held by the TUI.
//
// Returns error when cleanup fails.
func (t *TUI) Close() error {
	return t.service.Close()
}

// loadConfigOptions holds optional settings for LoadConfig.
type loadConfigOptions struct {
	// sandbox provides isolated disk access for loading configuration files.
	sandbox safedisk.Sandbox

	// homeDir is the path to the user's home directory.
	homeDir string
}

// LoadConfigOption configures LoadConfig behaviour.
type LoadConfigOption func(*loadConfigOptions)

// New creates a new TUI instance with the given options.
//
// Takes opts (...Option) which configure the TUI.
//
// Returns *TUI which is the configured TUI instance.
// Returns error when configuration is invalid or providers fail to initialise.
func New(opts ...Option) (*TUI, error) {
	tuiConfig := tui_dto.DefaultConfig()
	providers := &tui_domain.Providers{}

	for _, opt := range opts {
		opt(tuiConfig, providers)
	}

	initialiseProviders(tuiConfig, providers)

	service, err := tui_domain.NewService(tuiConfig, providers)
	if err != nil {
		return nil, err
	}

	return &TUI{
		config:    tuiConfig,
		providers: providers,
		service:   service,
	}, nil
}

// WithConfigSandbox injects a sandbox for reading config files.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file system access.
//
// Returns LoadConfigOption which applies the sandbox to config loading.
func WithConfigSandbox(sandbox safedisk.Sandbox) LoadConfigOption {
	return func(o *loadConfigOptions) {
		o.sandbox = sandbox
	}
}

// WithHomeDir overrides the user home directory used for config search.
//
// Takes directory (string) which specifies the directory path to use as the home
// directory.
//
// Returns LoadConfigOption which configures the home directory for config
// search.
func WithHomeDir(directory string) LoadConfigOption {
	return func(o *loadConfigOptions) {
		o.homeDir = directory
	}
}

// LoadConfig loads TUI configuration from a piko.yaml file, searching standard
// locations and applying environment variable overrides.
//
// When configPath is not empty, it is checked first. Otherwise, the function
// searches ./piko.yaml and then $HOME/.config/piko/piko.yaml in order.
//
// Environment variables (PIKO_TUI_*) override file values.
//
// Takes configPath (string) which is an optional path to the config file.
// Takes opts (...LoadConfigOption) which provides optional configuration
// such as WithConfigSandbox for testing.
//
// Returns Config which contains the loaded configuration with defaults applied.
// Returns error when the config file cannot be read or parsed.
func LoadConfig(configPath string, opts ...LoadConfigOption) (Config, error) {
	o := &loadConfigOptions{}
	for _, opt := range opts {
		opt(o)
	}

	paths := buildConfigSearchPaths(configPath, o.homeDir)
	tuiConfig := searchAndLoadConfig(paths, o)
	applyConfigEnvOverrides(&tuiConfig)
	return tuiConfig, nil
}

// initialiseProviders creates providers based on endpoint configuration.
//
// Takes tuiConfig (*tui_dto.Config) which specifies the TUI configuration
// settings.
// Takes providers (*tui_domain.Providers) which receives the initialised data
// providers.
func initialiseProviders(tuiConfig *tui_dto.Config, providers *tui_domain.Providers) {
	_, l := logger.From(context.Background(), log)

	if tuiConfig.MonitoringEndpoint == "" {
		l.Warn("No monitoring endpoint configured, TUI will have no data providers")
		return
	}

	grpcProviders, err := provider_grpc.NewProviders(tuiConfig.MonitoringEndpoint,
		provider_grpc.WithRefreshInterval(tuiConfig.RefreshInterval))
	if err != nil {
		l.Warn("Failed to connect to gRPC monitoring endpoint",
			logger.String("endpoint", tuiConfig.MonitoringEndpoint),
			logger.Error(err))
		return
	}

	providers.Resources = append(providers.Resources, grpcProviders.Resources...)
	providers.Metrics = append(providers.Metrics, grpcProviders.Metrics...)
	providers.Traces = append(providers.Traces, grpcProviders.Traces...)
	providers.Health = append(providers.Health, grpcProviders.Health...)
	providers.System = append(providers.System, grpcProviders.System...)
	providers.FDs = append(providers.FDs, grpcProviders.FDs...)
	l.Debug("Using gRPC monitoring providers", logger.String("endpoint", tuiConfig.MonitoringEndpoint))
}

// buildConfigSearchPaths assembles the ordered list of config file paths to
// try, from the explicit path, then the working directory, then the user
// home config directory.
//
// Takes configPath (string) which is an optional explicit config file path.
// Takes homeDir (string) which overrides the user home directory when non-empty.
//
// Returns []string which lists the paths to search in priority order.
func buildConfigSearchPaths(configPath, homeDir string) []string {
	var paths []string
	if configPath != "" {
		paths = append(paths, configPath)
	}
	paths = append(paths, "piko.yaml")

	if homeDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			homeDir = home
		}
	}
	if homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".config", "piko", "piko.yaml"))
	}
	return paths
}

// searchAndLoadConfig tries each path in order, returning the TUI config from
// the first successfully parsed file, or a default config if none is found.
//
// Takes paths ([]string) which lists the config file paths to try.
// Takes o (*loadConfigOptions) which provides optional sandbox access.
//
// Returns TUIConfig which is the loaded or default configuration.
func searchAndLoadConfig(paths []string, o *loadConfigOptions) TUIConfig {
	readFile := configFileReader(o)

	for _, path := range paths {
		data, err := readFile(path)
		if err != nil {
			continue
		}
		var pikoConfig pikoConfig
		if err := yaml.Unmarshal(data, &pikoConfig); err != nil {
			continue
		}
		return pikoConfig.TUI
	}

	return TUIConfig{
		Endpoint:        "http://localhost:8080",
		RefreshInterval: "2s",
		Theme:           "default",
	}
}

// configFileReader returns a function that reads a file using the sandbox when
// available, or os.ReadFile otherwise.
//
// Takes o (*loadConfigOptions) which provides optional sandbox access.
//
// Returns func(string) ([]byte, error) which reads file contents from the
// appropriate source.
func configFileReader(o *loadConfigOptions) func(string) ([]byte, error) {
	if o.sandbox != nil {
		return func(path string) ([]byte, error) {
			return o.sandbox.ReadFile(o.sandbox.RelPath(path))
		}
	}
	return func(path string) ([]byte, error) {
		return os.ReadFile(path) //nolint:gosec // trusted config path
	}
}

// applyConfigEnvOverrides applies PIKO_TUI_* environment variable overrides
// to the given TUI configuration.
//
// Takes tuiConfig (*TUIConfig) which is the configuration to modify in place.
func applyConfigEnvOverrides(tuiConfig *TUIConfig) {
	if v := os.Getenv("PIKO_TUI_ENDPOINT"); v != "" {
		tuiConfig.Endpoint = v
	}
	if v := os.Getenv("PIKO_TUI_REFRESH_INTERVAL"); v != "" {
		tuiConfig.RefreshInterval = v
	}
	if v := os.Getenv("PIKO_TUI_THEME"); v != "" {
		tuiConfig.Theme = v
	}
	if v := os.Getenv("PIKO_TUI_TITLE"); v != "" {
		tuiConfig.Title = v
	}
}
