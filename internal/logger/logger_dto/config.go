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

package logger_dto

import (
	"log/slog"
	"strings"
)

// SentryConfig holds settings for connecting to the Sentry error tracking
// service.
type SentryConfig struct {
	// DSN is the Sentry Data Source Name used to connect to Sentry.
	DSN string `env:"DSN" validate:"required" yaml:"dsn" json:"dsn"`

	// Environment identifies the deployment environment, such as "production"
	// or "staging".
	Environment string `env:"ENVIRONMENT" yaml:"environment,omitempty" json:"environment,omitempty"`

	// Release is the application version or commit hash sent to Sentry.
	Release string `env:"RELEASE" yaml:"release,omitempty" json:"release,omitempty"`

	// EventLevel sets the lowest log level that sends events to Sentry.
	EventLevel string `env:"EVENT_LEVEL" default:"error" yaml:"eventLevel,omitempty" json:"eventLevel,omitempty"`

	// BreadcrumbLevel sets the minimum log level for Sentry breadcrumbs.
	BreadcrumbLevel string `env:"BREADCRUMB_LEVEL" default:"info" yaml:"breadcrumbLevel,omitempty" json:"breadcrumbLevel,omitempty"`

	// IgnoreErrors is a list of error message patterns to exclude from Sentry.
	IgnoreErrors []string `env:"IGNORE_ERRORS" yaml:"ignoreErrors,omitempty" json:"ignoreErrors,omitempty"`

	// TracesSampleRate is the sampling fraction for
	// tracing transactions (0.0-1.0).
	TracesSampleRate float64 `env:"TRACES_SAMPLE_RATE" default:"0.2" yaml:"tracesSampleRate,omitempty" json:"tracesSampleRate,omitempty"`

	// SampleRate is the portion of error events to send to Sentry (0.0 to 1.0).
	SampleRate float64 `env:"SAMPLE_RATE" default:"1.0" yaml:"sampleRate,omitempty" json:"sampleRate,omitempty"`

	// Debug enables detailed SDK logging when set to true.
	Debug bool `env:"DEBUG" default:"false" yaml:"debug,omitempty" json:"debug,omitempty"`

	// EnableTracing enables Sentry performance tracing when set to true.
	EnableTracing bool `env:"ENABLE_TRACING" default:"true" yaml:"enableTracing,omitempty" json:"enableTracing,omitempty"`

	// SendDefaultPII enables sending personal data to Sentry; default is false.
	SendDefaultPII bool `env:"SEND_DEFAULT_PII" default:"false" yaml:"sendDefaultPII,omitempty" json:"sendDefaultPII,omitempty"`

	// AddSource indicates whether to include the source location in log entries.
	AddSource bool `env:"ADD_SOURCE" default:"true" yaml:"addSource,omitempty" json:"addSource,omitempty"`
}

// FileOutputConfig holds settings for writing logs to a file.
type FileOutputConfig struct {
	// Path is the file path where logs are written; must not be empty.
	Path string `env:"PATH" validate:"required" yaml:"path" json:"path"`

	// MaxSize is the maximum log file size in megabytes before rotation occurs.
	MaxSize int `env:"MAX_SIZE_MB" default:"100" yaml:"maxSize,omitempty" json:"maxSize,omitempty"`

	// MaxBackups is the number of old log files to keep; default is 5.
	MaxBackups int `env:"MAX_BACKUPS" default:"5" yaml:"maxBackups,omitempty" json:"maxBackups,omitempty"`

	// MaxAge is the maximum number of days to keep old log files.
	MaxAge int `env:"MAX_AGE_DAYS" default:"30" yaml:"maxAge,omitempty" json:"maxAge,omitempty"`

	// Compress enables gzip compression for rotated log files.
	Compress bool `env:"COMPRESS" default:"true" yaml:"compress,omitempty" json:"compress,omitempty"`

	// LocalTime uses local time for backup file timestamps instead of UTC.
	LocalTime bool `env:"LOCAL_TIME" default:"false" yaml:"localTime,omitempty" json:"localTime,omitempty"`
}

// OutputConfig holds settings for a single log output destination.
type OutputConfig struct {
	// AddSource overrides the global AddSource setting
	// for this output; nil uses global.
	AddSource *bool `yaml:"addSource,omitempty" json:"addSource,omitempty"`

	// File holds log file settings when output Type is "file"; nil when unused.
	File *FileOutputConfig `envPrefix:"FILE_" yaml:"file,omitempty" json:"file,omitempty"`

	// Name identifies this output for logging and error messages.
	Name string `env:"NAME" yaml:"name,omitempty" json:"name,omitempty"`

	// Type specifies the output destination ("stdout", "stderr", or "file").
	Type string `env:"TYPE" validate:"required" yaml:"type" json:"type"`

	// Level sets the log level for this output; empty uses the global level.
	Level string `env:"LEVEL" default:"info" yaml:"level,omitempty" json:"level,omitempty"`

	// Format specifies the log output format; valid values are "pretty" or "json".
	Format string `env:"FORMAT" default:"pretty" yaml:"format,omitempty" json:"format,omitempty"`

	// NoColour disables coloured output; defaults to true for file outputs.
	NoColour bool `env:"NO_COLOR" default:"false" yaml:"noColor,omitempty" json:"noColor,omitempty"`
}

// IntegrationConfig holds configuration for third-party logging integrations.
type IntegrationConfig struct {
	// Sentry holds the Sentry settings; nil when Type is not "sentry".
	Sentry *SentryConfig `envPrefix:"SENTRY_" yaml:"sentry,omitempty" json:"sentry,omitempty"`

	// Name identifies this logging integration for error messages.
	Name string `env:"NAME" yaml:"name,omitempty" json:"name,omitempty"`

	// Type specifies which integration handler to use (e.g. "sentry").
	Type string `env:"TYPE" validate:"required" yaml:"type" json:"type"`

	// Enabled indicates whether this integration is active.
	Enabled bool `env:"ENABLED" default:"false" yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// Config holds the settings for the logger.
type Config struct {
	// Level specifies how much detail to show in logs; defaults to "info".
	Level string `env:"LOG_LEVEL" default:"info" yaml:"level,omitempty" json:"level,omitempty"`

	// Outputs lists the log output destinations; if
	// empty, standard output is used.
	Outputs []OutputConfig `envPrefix:"OUTPUT_" yaml:"outputs,omitempty" json:"outputs,omitempty"`

	// Integrations holds the settings for each logging integration to enable.
	Integrations []IntegrationConfig `envPrefix:"INTEGRATION_" yaml:"integrations,omitempty" json:"integrations,omitempty"`

	// AddSource enables the logging of source file location in log entries.
	AddSource bool `env:"LOG_ADD_SOURCE" default:"true" yaml:"addSource" json:"addSource"`
}

// ParseLogLevel converts a string log level to its slog.Level equivalent.
//
// Takes level (string) which is the log level name to parse (case-insensitive).
// Takes defaultLevel (slog.Level) which is returned when level is unrecognised.
//
// Returns slog.Level which is the parsed level or defaultLevel if not matched.
func ParseLogLevel(level string, defaultLevel slog.Level) slog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return slog.Level(-8)
	case "internal":
		return slog.Level(-6)
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "notice":
		return slog.Level(2)
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return defaultLevel
	}
}

// parseSlogLevels parses a comma-separated string of log level names into
// slog.Level values.
//
// Takes levels (string) which contains comma-separated level names such as
// "debug,info,warn". Valid names are: trace, internal, debug, info, notice,
// warn, error. Names are case-insensitive and unknown names are ignored.
//
// Returns []slog.Level which contains the parsed levels, or nil if levels is
// empty.
func parseSlogLevels(levels string) []slog.Level {
	var result []slog.Level
	if levels == "" {
		return nil
	}
	for p := range strings.SplitSeq(strings.ToLower(levels), ",") {
		switch strings.TrimSpace(p) {
		case "trace":
			result = append(result, slog.Level(-8))
		case "internal":
			result = append(result, slog.Level(-6))
		case "debug":
			result = append(result, slog.LevelDebug)
		case "info":
			result = append(result, slog.LevelInfo)
		case "notice":
			result = append(result, slog.Level(2))
		case "warn":
			result = append(result, slog.LevelWarn)
		case "error":
			result = append(result, slog.LevelError)
		}
	}
	return result
}
