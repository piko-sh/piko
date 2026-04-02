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
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
)

// Option configures the TUI by modifying its configuration and providers.
type Option func(*tui_dto.Config, *tui_domain.Providers)

// Config is the configuration for the TUI monitoring tool.
type Config = TUIConfig

// WithConfig applies settings from a Piko Config so that the TUI can be
// configured via piko.yaml or environment variables.
//
// Takes tuiConfig (Config) which provides the Piko configuration settings.
//
// Returns Option which applies the configuration to the TUI.
//
// The following fields are applied:
//   - Endpoint -> PikoEndpoint
//   - RefreshInterval -> RefreshInterval (parsed as duration)
//   - Theme -> Theme
//   - Title -> Title
//
// Example:
// // Load from Piko config
// tuiConfig := loadPikoConfig() // your config loading logic
// t, err := tui.New(
//
//	tui.WithConfig(tuiConfig.TUI),
//	tui.WithMonitoringEndpoint("localhost:9091"),
//
// )
func WithConfig(tuiConfig Config) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		if tuiConfig.Endpoint != "" {
			c.PikoEndpoint = tuiConfig.Endpoint
		}
		if tuiConfig.RefreshInterval != "" {
			if d, err := time.ParseDuration(tuiConfig.RefreshInterval); err == nil {
				c.RefreshInterval = d
			}
		}
		if tuiConfig.Theme != "" {
			c.Theme = tuiConfig.Theme
		}
		if tuiConfig.Title != "" {
			c.Title = tuiConfig.Title
		}
	}
}

// WithPikoEndpoint sets the base URL for the Piko server.
// This is the main address where the Piko application is running.
//
// Takes url (string) which is the base URL (e.g. "http://localhost:8080").
//
// Returns Option which sets the Piko endpoint in the configuration.
func WithPikoEndpoint(url string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.PikoEndpoint = url
	}
}

// WithMonitoringEndpoint configures the gRPC monitoring server endpoint.
//
// When set, the TUI uses gRPC to fetch all monitoring data instead of
// direct database access or HTTP endpoints. This enables remote monitoring
// via kubectl port-forward and makes the TUI database-agnostic.
//
// The gRPC monitoring server provides:
//   - Resource data (orchestrator tasks, registry artefacts)
//   - Metrics, traces, and system stats
//   - Health status
//   - File descriptor information
//
// Takes address (string) which is the gRPC server address,
// such as "localhost:9091".
//
// Returns Option which configures the TUI to use gRPC monitoring.
func WithMonitoringEndpoint(address string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.MonitoringEndpoint = address
	}
}

// WithPrometheus configures an external Prometheus server for metrics,
// queried alongside PikoEndpoint when both are set.
//
// Takes url (string) which is the Prometheus URL (e.g., "http://localhost:9090").
//
// Returns Option which configures the TUI to use Prometheus.
func WithPrometheus(url string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.PrometheusURL = url
	}
}

// WithJaeger configures an external Jaeger server for traces.
//
// When both PikoEndpoint and Jaeger are configured, both will be queried.
//
// Takes url (string) which is the Jaeger URL (e.g., "http://localhost:16686").
//
// Returns Option which configures the TUI to use Jaeger.
func WithJaeger(url string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.JaegerURL = url
	}
}

// WithRefreshInterval sets the interval between data refreshes.
// Lower values give more frequent updates but increase system load.
//
// Takes d (time.Duration) which is the refresh interval (default: 2s).
//
// Returns Option which configures the refresh interval.
func WithRefreshInterval(d time.Duration) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.RefreshInterval = d
	}
}

// WithTheme sets the UI theme for the terminal interface.
// Available themes: "default" (256-colour), "minimal" (16-colour).
//
// Takes theme (string) which specifies the theme name to use.
//
// Returns Option which applies the theme setting to the configuration.
func WithTheme(theme string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.Theme = theme
	}
}

// WithTitle sets the title bar text shown in the TUI header.
//
// Takes title (string) which is the text to display.
//
// Returns Option which configures the title setting.
func WithTitle(title string) Option {
	return func(c *tui_dto.Config, _ *tui_domain.Providers) {
		c.Title = title
	}
}

// WithPanel adds a custom panel to the TUI.
// Custom panels appear after the built-in panels.
//
// Takes p (Panel) which is the panel to add.
//
// Returns Option which configures the TUI to include the panel.
func WithPanel(p Panel) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.Panels = append(providers.Panels, p)
	}
}

// WithMetricsProvider returns an option that adds a metrics provider.
//
// Takes p (MetricsProvider) which is the provider to add.
//
// Returns Option which registers the provider when applied.
func WithMetricsProvider(p MetricsProvider) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.Metrics = append(providers.Metrics, p)
	}
}

// WithTracesProvider returns an option that adds a traces provider.
//
// Takes p (TracesProvider) which is the provider to add.
//
// Returns Option which configures the providers to include the given traces
// provider.
func WithTracesProvider(p TracesProvider) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.Traces = append(providers.Traces, p)
	}
}

// WithResourceProvider adds a resource provider to the provider list.
//
// Takes p (ResourceProvider) which is the provider to add.
//
// Returns Option which appends the provider to the resources list.
func WithResourceProvider(p ResourceProvider) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.Resources = append(providers.Resources, p)
	}
}

// WithHealthProvider adds a health provider to the list of providers.
//
// Takes p (HealthProvider) which is the health provider to add.
//
// Returns Option which configures the providers to include p.
func WithHealthProvider(p HealthProvider) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.Health = append(providers.Health, p)
	}
}

// WithFDsProvider returns an option that adds a file descriptor provider.
//
// Takes p (FDsProvider) which is the provider to add.
//
// Returns Option which adds the provider to the list of file descriptor
// providers.
func WithFDsProvider(p FDsProvider) Option {
	return func(_ *tui_dto.Config, providers *tui_domain.Providers) {
		providers.FDs = append(providers.FDs, p)
	}
}
