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

package bootstrap

import (
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_dto"
)

// ServerConfig is the bootstrap-owned aggregate of resolved framework
// values.
//
// The With* options each write to a single sub-field, then the
// config_domain loader applies defaults, resolves placeholders, and
// validates the result. Only the bootstrap pipeline constructs and
// consumes ServerConfig. Users configure piko exclusively through With*
// options.
type ServerConfig struct {
	// Security holds settings for security headers, rate limiting, and reporting.
	Security config.SecurityConfig

	// Paths specifies directory paths for pages, emails, partials, and other
	// source locations.
	Paths config.PathsConfig

	// HealthProbe configures the health check endpoint for liveness and readiness
	// probes.
	HealthProbe config.HealthProbeConfig

	// Network holds the network configuration for the server.
	Network config.NetworkConfig

	// Storage sets up the storage service, including presigned URL support.
	Storage config.StorageConfig

	// Database specifies the database connection settings.
	Database config.DatabaseConfig

	// Otlp holds the settings for the OpenTelemetry Protocol exporter used for
	// tracing and metrics.
	Otlp config.OtlpConfig

	// Build sets build options such as watch mode and asset pre-rendering.
	Build config.BuildModeConfig

	// I18nDefaultLocale specifies the default locale for internationalisation;
	// defaults to "en" if not set.
	I18nDefaultLocale *string `default:"en"`

	// CSRFSecret is the secret for CSRF token generation. If not set, a random
	// secret is generated and persisted to a temp file.
	CSRFSecret *string

	// Logger configures the application logging behaviour.
	Logger logger_dto.Config
}
