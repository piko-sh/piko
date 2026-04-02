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

package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logrotate"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/wdk/logger/logger_state"
)

// logKeyType is the log attribute key for integration type.
const logKeyType = "type"

// initialisationState holds the state that builds up during logger setup.
type initialisationState struct {
	// shutdownTasks holds cleanup functions called in reverse order during
	// shutdown.
	shutdownTasks []func(context.Context) error

	// handlers holds the collected slog handlers for building the core handler.
	handlers []slog.Handler

	// closers holds writers that must be closed during shutdown.
	closers []io.Closer
}

// earlyInitialisationResult is returned when initialisation
// completes early, such as when explicit handlers exist.
type earlyInitialisationResult struct {
	// logger is the configured logger for early initialisation output.
	logger *slog.Logger

	// shutdown releases resources when the early init path is taken.
	shutdown func(context.Context) error
}

// Initialise creates and configures the logging system based on the provided
// configuration. It sets up OpenTelemetry integration, configures output
// handlers, integrations, and notifications.
//
// Takes ctx (context.Context) which controls cancellation during setup.
// Takes logConfig (logger_dto.Config) which specifies the logging configuration.
// Takes otelConfig (driver_handlers.OtelSetupConfig) which provides OTLP
// exporter settings.
// Takes otelOpts (*driver_handlers.OtelSetupOptions) which controls
// OpenTelemetry setup behaviour.
//
// Returns *slog.Logger which is the configured logger.
// Returns func(context.Context) error which is the shutdown function for
// graceful cleanup.
// Returns error when setup fails.
func Initialise(
	ctx context.Context, logConfig logger_dto.Config,
	otelConfig driver_handlers.OtelSetupConfig,
	otelOpts *driver_handlers.OtelSetupOptions,
) (*slog.Logger, func(context.Context) error, error) {
	state := &initialisationState{
		handlers: make([]slog.Handler, 0, len(logConfig.Outputs)),
		closers:  make([]io.Closer, 0, len(logConfig.Outputs)),
	}

	if err := setupOtelIntegration(ctx, logConfig, otelConfig, otelOpts, state); err != nil {
		return nil, nil, err
	}

	globalLevel := logger_dto.ParseLogLevel(logConfig.Level, slog.LevelInfo)

	processOutputHandlers(ctx, logConfig, globalLevel, state)
	processIntegrationHandlers(logConfig, state)

	if len(state.handlers) == 0 {
		if earlyResult := handleNoHandlersConfigured(logConfig, globalLevel, state); earlyResult != nil {
			return earlyResult.logger, earlyResult.shutdown, nil
		}
	}

	coreHandler := buildCoreHandler(state.handlers)
	var wrappedHandler slog.Handler = logger_domain.NewOTelSlogHandler(coreHandler)

	finalLogger := slog.New(wrappedHandler)
	if envAttrs := logger_domain.EnvironmentSlogAttrs(); len(envAttrs) > 0 {
		finalLogger = finalLogger.With(logger_domain.SlogAttrsToAny(envAttrs)...)
	}
	slog.SetDefault(finalLogger)
	logger_domain.InitDefaultFactory(finalLogger)

	return finalLogger, buildCompositeShutdown(state), nil
}

// setupOtelIntegration configures OpenTelemetry and appends its shutdown task.
//
// Takes logConfig (logger_dto.Config) which provides the logger configuration.
// Takes otelConfig (driver_handlers.OtelSetupConfig) which specifies OTLP exporter
// settings.
// Takes otelOpts (*driver_handlers.OtelSetupOptions) which controls
// OpenTelemetry
// setup behaviour.
// Takes state (*initialisationState) which holds initialisation
// state including shutdown tasks.
//
// Returns error when OpenTelemetry setup fails.
func setupOtelIntegration(ctx context.Context, logConfig logger_dto.Config, otelConfig driver_handlers.OtelSetupConfig, otelOpts *driver_handlers.OtelSetupOptions, state *initialisationState) error {
	enabledIntegrations := getEnabledIntegrationTypes(logConfig)
	otelShutdown, err := driver_handlers.SetupOtel(ctx, otelConfig, enabledIntegrations, otelOpts)
	if err != nil {
		return fmt.Errorf("failed to setup OpenTelemetry: %w", err)
	}
	if otelShutdown != nil {
		state.shutdownTasks = append(state.shutdownTasks, otelShutdown)
	}
	return nil
}

// processOutputHandlers creates handlers for each configured output.
//
// Takes ctx (context.Context) which controls cancellation during setup.
// Takes logConfig (logger_dto.Config) which specifies the logging configuration.
// Takes globalLevel (slog.Level) which sets the default log level for handlers.
// Takes state (*initialisationState) which collects the created
// handlers and closers.
func processOutputHandlers(ctx context.Context, logConfig logger_dto.Config, globalLevel slog.Level, state *initialisationState) {
	for _, outConfig := range logConfig.Outputs {
		shouldUseSource := resolveAddSource(outConfig.AddSource, logConfig.AddSource)
		handler, closer, err := createHandlerForOutput(ctx, outConfig, shouldUseSource, globalLevel)
		if err != nil {
			slog.Error("Failed to create log output handler", "name", outConfig.Name, "error", err)
			continue
		}
		if closer != nil {
			state.closers = append(state.closers, closer)
		}
		state.handlers = append(state.handlers, handler)
	}
}

// resolveAddSource determines whether to add source information to log entries.
//
// Takes outputSpecific (*bool) which overrides the default when set.
// Takes globalDefault (bool) which specifies the fallback value.
//
// Returns bool which is the resolved setting for adding source information.
func resolveAddSource(outputSpecific *bool, globalDefault bool) bool {
	if outputSpecific != nil {
		return *outputSpecific
	}
	return globalDefault
}

// processIntegrationHandlers creates handlers for each enabled integration.
//
// Takes logConfig (logger_dto.Config) which provides the integration settings.
// Takes state (*initialisationState) which collects the created
// handlers and shutdown tasks.
func processIntegrationHandlers(logConfig logger_dto.Config, state *initialisationState) {
	for _, integrationConfig := range logConfig.Integrations {
		if !integrationConfig.Enabled {
			continue
		}
		handler, integrationShutdown, err := createHandlerForIntegration(integrationConfig)
		if err != nil {
			slog.Error("Failed to create integration handler", logKeyType, integrationConfig.Type, "error", err)
			continue
		}
		if handler != nil {
			state.handlers = append(state.handlers, handler)
		}
		if integrationShutdown != nil {
			state.shutdownTasks = append(state.shutdownTasks, integrationShutdown)
		}
	}
}

// handleNoHandlersConfigured handles the case when no output handlers were
// configured.
//
// Takes logConfig (logger_dto.Config) which provides the logger configuration.
// Takes globalLevel (slog.Level) which specifies the minimum logging level.
// Takes state (*initialisationState) which holds the
// initialisation state to update.
//
// Returns *earlyInitialisationResult which contains an early
// result if explicit handlers exist, otherwise returns nil after
// appending a default handler to state.
func handleNoHandlersConfigured(logConfig logger_dto.Config, globalLevel slog.Level, state *initialisationState) *earlyInitialisationResult {
	if logger_state.HasExplicitHandlers() {
		return &earlyInitialisationResult{
			logger:   slog.Default(),
			shutdown: func(context.Context) error { return nil },
		}
	}

	defaultHandler := slog.NewTextHandler(logger_domain.StderrWriter(), &slog.HandlerOptions{
		Level:       globalLevel,
		AddSource:   logConfig.AddSource,
		ReplaceAttr: logger_domain.ReplaceLevelAttr,
	})
	state.handlers = append(state.handlers, defaultHandler)
	return nil
}

// buildCoreHandler creates either a single handler or a multi-handler from
// the slice.
//
// Takes handlers ([]slog.Handler) which contains the handlers to combine.
//
// Returns slog.Handler which is a single handler if only one is provided, or
// a multi-handler that combines all handlers otherwise.
func buildCoreHandler(handlers []slog.Handler) slog.Handler {
	if len(handlers) == 1 {
		return handlers[0]
	}
	return slog.NewMultiHandler(handlers...)
}

// buildCompositeShutdown creates a shutdown function that closes all writers
// and runs shutdown tasks.
//
// Takes state (*initialisationState) which holds the closers and
// shutdown tasks to run.
//
// Returns func(context.Context) error which closes all writers and executes
// shutdown tasks in reverse order, returning any combined errors.
func buildCompositeShutdown(state *initialisationState) func(context.Context) error {
	return func(shutdownCtx context.Context) error {
		var allErrors []error

		for _, c := range state.closers {
			if err := c.Close(); err != nil {
				allErrors = append(allErrors, fmt.Errorf("failed to close writer: %w", err))
			}
		}

		for i := len(state.shutdownTasks) - 1; i >= 0; i-- {
			if err := state.shutdownTasks[i](shutdownCtx); err != nil {
				allErrors = append(allErrors, err)
			}
		}

		return errors.Join(allErrors...)
	}
}

// getEnabledIntegrationTypes returns a list of integration type names that are
// both enabled in the config and have their adapter package imported.
//
// Takes config (logger_dto.Config) which specifies the logger configuration
// containing integration settings.
//
// Returns []string which contains the names of available enabled integrations.
func getEnabledIntegrationTypes(config logger_dto.Config) []string {
	var result []string
	for _, integration := range config.Integrations {
		if !integration.Enabled {
			continue
		}
		if logger_domain.IsIntegrationAvailable(integration.Type) {
			result = append(result, integration.Type)
		}
	}
	return result
}

// createHandlerForIntegration creates a log handler for the given integration
// configuration.
//
// Takes config (logger_dto.IntegrationConfig) which specifies the integration
// type and its settings.
//
// Returns slog.Handler which is the created handler, or nil if the
// integration is unavailable.
// Returns func(context.Context) error which is the shutdown function that
// releases integration resources, or nil if none is needed.
// Returns error when the integration handler cannot be created.
func createHandlerForIntegration(config logger_dto.IntegrationConfig) (slog.Handler, func(context.Context) error, error) {
	integration := logger_domain.GetIntegration(config.Type)
	if integration == nil {
		slog.Warn("Integration type not available - import the corresponding adapter package",
			slog.String(logKeyType, config.Type),
			slog.String("hint", fmt.Sprintf("import _ \"piko.sh/piko/wdk/logger/logger_integration_%s\"", config.Type)))
		return nil, nil, nil
	}

	var integrationConfig any
	switch strings.ToLower(config.Type) {
	case "sentry":
		if config.Sentry == nil {
			return nil, nil, errors.New("sentry config is nil but integration is enabled")
		}
		integrationConfig = config.Sentry
	default:
		slog.Warn("No config mapping for integration type", slog.String(logKeyType, config.Type))
		return nil, nil, nil
	}

	handler, err := integration.CreateHandler(integrationConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create %s handler: %w", config.Type, err)
	}

	return handler, nil, nil
}

// createHandlerForOutput creates a log handler for the specified output
// configuration.
//
// Takes ctx (context.Context) which controls the lifetime of background
// goroutines such as log rotation.
// Takes config (logger_dto.OutputConfig) which specifies the output type, format,
// and optional file settings.
// Takes addSource (bool) which controls whether source location is included in
// log entries.
// Takes globalLevel (slog.Level) which provides the default log level when not
// overridden by the output configuration.
//
// Returns slog.Handler which is the configured handler ready for use.
// Returns io.Closer which is non-nil for file outputs and must be closed when
// logging is complete.
// Returns error when the output type is unknown or file configuration is
// missing.
func createHandlerForOutput(ctx context.Context, config logger_dto.OutputConfig, addSource bool, globalLevel slog.Level) (slog.Handler, io.Closer, error) {
	level := globalLevel
	if config.Level != "" {
		level = logger_dto.ParseLogLevel(config.Level, globalLevel)
	}

	var writer io.Writer
	var closer io.Closer

	switch strings.ToLower(config.Type) {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = logger_domain.StderrWriter()
	case "file":
		if config.File == nil || config.File.Path == "" {
			return nil, nil, errors.New("output type 'file' requires a path")
		}
		fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
			Directory:  filepath.Dir(config.File.Path),
			Filename:   filepath.Base(config.File.Path),
			MaxSize:    config.File.MaxSize,
			MaxBackups: config.File.MaxBackups,
			MaxAge:     config.File.MaxAge,
			Compress:   config.File.Compress,
			LocalTime:  config.File.LocalTime,
		})
		if fileError != nil {
			return nil, nil, fmt.Errorf("creating file output %q: %w", config.File.Path, fileError)
		}
		writer = fileLogger
		closer = fileLogger
	case "discard":
		writer = io.Discard
	default:
		return nil, nil, fmt.Errorf("unknown output type: %s", config.Type)
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	var handler slog.Handler
	if strings.EqualFold(config.Format, "pretty") {
		isFileOutput := strings.EqualFold(config.Type, "file")
		noColour := config.NoColour || isFileOutput
		handler = driver_handlers.NewPrettyHandler(writer, &driver_handlers.Options{
			Level:     levelVar,
			AddSource: addSource,
			NoColour:  noColour,
		})
	} else {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   addSource,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		})
	}

	return handler, closer, nil
}
