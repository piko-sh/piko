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

package tui_dto

import (
	"time"

	"piko.sh/piko/wdk/clock"
)

const (
	// DefaultRefreshInterval is the default interval for refreshing data.
	DefaultRefreshInterval = 2 * time.Second

	// DefaultTheme is the default theme for the user interface.
	DefaultTheme = "default"

	// DefaultTitle is the default title shown in the terminal interface.
	DefaultTitle = "Piko TUI"

	// DefaultPikoEndpoint is the default Piko application URL.
	DefaultPikoEndpoint = "http://localhost:8080"

	// DefaultHealthEndpoint is the default Piko health server URL.
	// The health server runs on a separate port (9090) for Kubernetes probes.
	DefaultHealthEndpoint = "http://localhost:9090"

	// DefaultMonitoringEndpoint is the default Piko gRPC monitoring server address
	// on port 9091. Uses 127.0.0.1 explicitly to avoid IPv6/IPv4 mismatch issues
	// with "localhost".
	DefaultMonitoringEndpoint = "127.0.0.1:9091"
)

// Config holds the configuration for the TUI monitoring tool.
// This struct is populated by the public facade's With* options.
type Config struct {
	// Clock provides time operations for testing.
	// If nil, defaults to the real system clock.
	Clock clock.Clock

	// Title is the text shown in the TUI header.
	Title string

	// Theme selects the UI colour theme; valid values are "default" or "minimal".
	Theme string

	// PikoEndpoint is the base URL of the Piko application, for example
	// "http://localhost:8080". An empty value means no endpoint is configured.
	PikoEndpoint string

	// HealthEndpoint is the URL of the Piko health server, which runs on a
	// separate port (default 9090) and exposes liveness and readiness probe
	// endpoints (/live, /ready). Example: "http://localhost:9090".
	HealthEndpoint string

	// MonitoringEndpoint is the address of the Piko gRPC monitoring
	// server, enabling database-agnostic remote monitoring via gRPC
	// instead of direct database access.
	//
	// Example: "localhost:9091"
	MonitoringEndpoint string

	// PrometheusURL is the URL of an external Prometheus server.
	// Optional; only needed if not using PikoEndpoint for metrics.
	PrometheusURL string

	// JaegerURL is the URL of an external Jaeger server for trace collection.
	// This field is optional and only needed if not using PikoEndpoint for traces.
	JaegerURL string

	// RefreshInterval controls how often data is refreshed from providers.
	// Zero means no automatic refresh.
	RefreshInterval time.Duration
}

// GetClock returns the configured clock, defaulting to RealClock if nil.
//
// Returns clock.Clock which is the configured clock or a real clock if none
// was set.
func (c *Config) GetClock() clock.Clock {
	if c.Clock == nil {
		return clock.RealClock()
	}
	return c.Clock
}

// HasPikoEndpoint returns true if a Piko endpoint is configured.
//
// Returns bool which is true when PikoEndpoint is set to a non-empty value.
func (c *Config) HasPikoEndpoint() bool {
	return c.PikoEndpoint != ""
}

// HasHealthEndpoint returns true if a health endpoint is configured.
//
// Returns bool which is true when a health endpoint path has been set.
func (c *Config) HasHealthEndpoint() bool {
	return c.HealthEndpoint != ""
}

// HasMonitoringEndpoint reports whether a gRPC monitoring endpoint is configured.
//
// Returns bool which is true when the monitoring endpoint is set.
func (c *Config) HasMonitoringEndpoint() bool {
	return c.MonitoringEndpoint != ""
}

// HasPrometheus returns true if a Prometheus URL is configured.
//
// Returns bool which is true when PrometheusURL is set.
func (c *Config) HasPrometheus() bool {
	return c.PrometheusURL != ""
}

// HasJaeger returns true if a Jaeger URL is configured.
//
// Returns bool which indicates whether Jaeger tracing is enabled.
func (c *Config) HasJaeger() bool {
	return c.JaegerURL != ""
}

// HasAnyOTELSource returns true if any OTEL data source is configured.
//
// Returns bool which is true when Piko, Prometheus, or Jaeger is configured.
func (c *Config) HasAnyOTELSource() bool {
	return c.HasPikoEndpoint() || c.HasPrometheus() || c.HasJaeger()
}

// DefaultConfig returns a Config with sensible defaults.
//
// Returns *Config which contains the default settings ready for use.
func DefaultConfig() *Config {
	return &Config{
		Title:              DefaultTitle,
		Theme:              DefaultTheme,
		RefreshInterval:    DefaultRefreshInterval,
		PikoEndpoint:       DefaultPikoEndpoint,
		HealthEndpoint:     DefaultHealthEndpoint,
		MonitoringEndpoint: DefaultMonitoringEndpoint,
		Clock:              clock.RealClock(),
	}
}
