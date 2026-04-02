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

package logger_integration_sentry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sentryslog "github.com/getsentry/sentry-go/slog"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/wdk/logger/logger_state"
)

var (
	_ logger_domain.Integration = (*sentryIntegration)(nil)

	_ logger_domain.OtelIntegration = (*sentryIntegration)(nil)

	sentryInitOnce sync.Once
)

// sentryIntegration implements logger_domain.Integration and
// logger_domain.OtelIntegration.
type sentryIntegration struct{}

// Type returns the integration type name as used in config files.
//
// Returns string which is the identifier "sentry".
func (*sentryIntegration) Type() string {
	return "sentry"
}

// sentryConfig holds the parsed settings for Sentry setup.
type sentryConfig struct {
	// dsn is the Sentry Data Source Name used for error reporting.
	dsn string

	// environment specifies the deployment environment name for Sentry reporting.
	environment string

	// release is the app version sent to Sentry for tracking releases.
	release string

	// ignoreErrors lists error patterns to exclude from Sentry reports.
	ignoreErrors []string

	// eventLevels lists the log levels that trigger Sentry events.
	eventLevels []slog.Level

	// breadcrumbLevels specifies which log levels to record as Sentry breadcrumbs.
	breadcrumbLevels []slog.Level

	// tracesSampleRate is the portion of transactions to send to Sentry.
	// Values range from 0.0 (none) to 1.0 (all).
	tracesSampleRate float64

	// sampleRate is the fraction of events sent to Sentry; 1.0 sends all events.
	sampleRate float64

	// debug enables detailed logging for Sentry operations.
	debug bool

	// enableTracing enables Sentry performance tracing when set to true.
	enableTracing bool

	// sendDefaultPII enables sending personal data to Sentry.
	sendDefaultPII bool

	// addSource includes the source file location in log entries.
	addSource bool
}

// CreateHandler creates a new slog.Handler that sends log events to Sentry.
//
// Takes config (any) which should be *logger_dto.SentryConfig or Config.
//
// Returns slog.Handler which sends log events to Sentry.
// Returns error when the configuration is invalid or Sentry initialisation
// fails.
func (*sentryIntegration) CreateHandler(config any) (slog.Handler, error) {
	parsedConfig, err := parseSentryConfig(config)
	if err != nil {
		return nil, err
	}

	if err := initialiseSentryClient(parsedConfig); err != nil {
		return nil, err
	}

	registerSentryShutdownHook()

	return createSentryHandler(parsedConfig), nil
}

// OtelComponents returns the Sentry OpenTelemetry span processor and
// propagator.
//
// Returns logger_domain.SpanProcessor which processes spans for Sentry tracing.
// Returns propagation.TextMapPropagator which propagates trace context.
func (*sentryIntegration) OtelComponents() (logger_domain.SpanProcessor, propagation.TextMapPropagator) {
	spanProcessor := sentryotel.NewSentrySpanProcessor()
	propagator := sentryotel.NewSentryPropagator()
	return spanProcessor, propagator
}

// Config holds the settings for Sentry integration.
// It provides a stable public API for setting up Sentry in code.
type Config struct {
	// DSN is the data source name for connecting to the database.
	DSN string

	// Environment specifies the deployment environment name.
	Environment string

	// Release specifies the version or release identifier.
	Release string

	// TracesSampleRate sets the fraction of traces to collect; 0.0 means none,
	// 1.0 means all.
	TracesSampleRate float64

	// SampleRate specifies the sampling rate as a fraction; 1.0 samples
	// everything, 0.0 samples nothing.
	SampleRate float64

	// Debug enables detailed logging for troubleshooting; default is false.
	Debug bool
}

// Enable sets up the Sentry SDK and adds Sentry as a log handler. This
// function uses sync.Once to ensure Sentry is only set up once, even if
// called multiple times.
//
// Use this for programmatic Sentry setup. For config-based setup via
// piko.yaml, simply import this package and configure Sentry in your config
// file.
//
// Takes config (Config) which specifies the Sentry configuration settings.
func Enable(config Config) {
	sentryInitOnce.Do(func() {
		log.Info("Sentry integration enabled. Initialising Sentry SDK...")

		integration := logger_domain.GetIntegration("sentry")
		if integration == nil {
			log.Error("Sentry integration not registered - this should not happen")
			return
		}

		handler, err := integration.CreateHandler(config)
		if err != nil {
			log.Error("Failed to initialise Sentry handler", logger_domain.Error(err))
			return
		}

		if handler != nil {
			logger_state.AddHandler(handler, nil)
			log.Debug("Sentry handler added to logger")
		}
	})
}

// parseSentryConfig parses settings from different config types.
//
// Takes config (any) which is the config to parse. It should be
// *logger_dto.SentryConfig or Config.
//
// Returns sentryConfig which contains the parsed Sentry settings.
// Returns error when config is nil or has an unsupported type.
func parseSentryConfig(config any) (sentryConfig, error) {
	switch c := config.(type) {
	case *logger_dto.SentryConfig:
		if c == nil {
			return sentryConfig{}, errors.New("sentry config is nil")
		}
		return sentryConfig{
			dsn:              c.DSN,
			environment:      c.Environment,
			release:          c.Release,
			ignoreErrors:     c.IgnoreErrors,
			eventLevels:      parseSlogLevels(c.EventLevel),
			breadcrumbLevels: parseSlogLevels(c.BreadcrumbLevel),
			tracesSampleRate: c.TracesSampleRate,
			sampleRate:       c.SampleRate,
			debug:            c.Debug,
			enableTracing:    c.EnableTracing,
			sendDefaultPII:   c.SendDefaultPII,
			addSource:        c.AddSource,
		}, nil
	case Config:
		return sentryConfig{
			dsn:              c.DSN,
			environment:      c.Environment,
			release:          c.Release,
			tracesSampleRate: c.TracesSampleRate,
			sampleRate:       c.SampleRate,
			debug:            c.Debug,
			enableTracing:    true,
			sendDefaultPII:   false,
			addSource:        true,
		}, nil
	default:
		return sentryConfig{}, fmt.Errorf("unsupported config type: %T (expected *logger_dto.SentryConfig or Config)", config)
	}
}

// initialiseSentryClient sets up the Sentry SDK with the given settings.
//
// Takes config (sentryConfig) which specifies the Sentry client settings.
//
// Returns error when the Sentry SDK fails to start.
func initialiseSentryClient(config sentryConfig) error {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.dsn,
		Debug:            config.debug,
		EnableTracing:    config.enableTracing,
		SampleRate:       config.sampleRate,
		TracesSampleRate: config.tracesSampleRate,
		Environment:      config.environment,
		Release:          config.release,
		SendDefaultPII:   config.sendDefaultPII,
		IgnoreErrors:     config.ignoreErrors,
		EnableLogs:       true,
	})
	if err != nil {
		return fmt.Errorf("sentry SDK initialisation failed: %w", err)
	}
	return nil
}

// registerSentryShutdownHook registers a shutdown hook to flush Sentry events
// before the program exits.
func registerSentryShutdownHook() {
	logger_domain.RegisterShutdownHook(func() {
		slog.Debug("Flushing Sentry events for shutdown...")
		if sentry.Flush(2 * time.Second) {
			slog.Debug("Sentry flush completed successfully.")
		} else {
			slog.Warn("Sentry flush timed out. Some events may have been lost.")
		}
	})
}

// createSentryHandler creates a Sentry handler with the given settings.
//
// Takes config (sentryConfig) which sets the Sentry logging options.
//
// Returns slog.Handler which is ready to send logs to Sentry.
func createSentryHandler(config sentryConfig) slog.Handler {
	handlerOptions := sentryslog.Option{
		EventLevel: config.eventLevels,
		LogLevel:   config.breadcrumbLevels,
		AddSource:  config.addSource,
	}
	return handlerOptions.NewSentryHandler(context.Background())
}

// parseSlogLevels parses a comma-separated string of log level names into
// slog.Level values.
//
// Takes levels (string) which contains comma-separated level names such as
// "trace", "internal", "debug", "info", "notice", "warn", or "error".
//
// Returns []slog.Level which contains the parsed levels, or nil if the input
// is empty.
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

func init() {
	logger_domain.RegisterIntegration(&sentryIntegration{})
}
