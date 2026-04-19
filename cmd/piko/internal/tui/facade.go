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
	"fmt"
	"os"
	"path/filepath"

	"piko.sh/piko/cmd/piko/internal/tui/tui_adapters/provider_grpc"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/wdk/logger"
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

// log is the package-level logger for the tui package.
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

// LoadConfig loads TUI configuration using the user-facing config_domain
// loader, with PIKO_TUI_* environment variables and tui.yaml file support.
//
// When configPath is not empty, that file is loaded directly. Otherwise the
// loader searches ./tui.yaml and then $HOME/.config/piko/tui.yaml.
//
// Precedence (lowest to highest): struct-tag defaults, file values,
// PIKO_TUI_* environment variables.
//
// Takes configPath (string) which is an optional path to the config file.
// Takes opts (...LoadConfigOption) which provides optional configuration
// such as WithHomeDir for testing.
//
// Returns Config which contains the loaded configuration with defaults applied.
// Returns error when the config file cannot be read or parsed.
func LoadConfig(configPath string, opts ...LoadConfigOption) (Config, error) {
	o := &loadConfigOptions{}
	for _, opt := range opts {
		opt(o)
	}

	filePaths := discoverConfigFiles(configPath, o.homeDir)

	loaderOpts := config_domain.LoaderOptions{
		FilePaths:          filePaths,
		StrictFile:         false,
		UseGlobalResolvers: false,
		PassOrder: []config_domain.Pass{
			config_domain.PassDefaults,
			config_domain.PassFiles,
			config_domain.PassEnv,
		},
	}

	var tuiConfig Config
	if _, err := config_domain.Load(context.Background(), &tuiConfig, loaderOpts); err != nil {
		return Config{}, fmt.Errorf("loading TUI config: %w", err)
	}
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
	providers.Watchdog = append(providers.Watchdog, grpcProviders.Watchdog...)
	providers.ProvidersInfo = append(providers.ProvidersInfo, grpcProviders.ProvidersInfo...)
	providers.DLQ = append(providers.DLQ, grpcProviders.DLQ...)
	providers.RateLimiter = append(providers.RateLimiter, grpcProviders.RateLimiter...)
	providers.Profiling = append(providers.Profiling, grpcProviders.Profiling...)
	l.Debug("Using gRPC monitoring providers", logger.String("endpoint", tuiConfig.MonitoringEndpoint))
}

// discoverConfigFiles returns the existing tui.yaml paths to feed to the
// config_domain loader.
//
// When configPath is non-empty it is the only file loaded. Otherwise the
// loader looks in the user's home config directory first (lowest
// precedence) and then the working directory (highest precedence) so a
// project-local tui.yaml wins over the global one.
//
// Takes configPath (string) which is an optional explicit config file path.
// Takes homeDir (string) which overrides the user home directory when non-empty.
//
// Returns []string which lists existing config files in load order
// (lowest-to-highest precedence; later files override earlier).
func discoverConfigFiles(configPath, homeDir string) []string {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return []string{configPath}
		}
		return nil
	}

	var candidates []string
	if homeDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			homeDir = home
		}
	}
	if homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".config", "piko", "tui.yaml"))
	}
	candidates = append(candidates, "tui.yaml")

	var existing []string
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	return existing
}
