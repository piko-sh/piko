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

package logger_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/internal/logrotate"
)

var (
	globalState *loggerState

	globalStateMutex sync.RWMutex

	shutdownFunc func(context.Context) error

	shutdownFuncMutex sync.RWMutex
)

// loggerState holds the logger instance and its associated handlers.
type loggerState struct {
	// logger is the configured structured logger instance.
	logger *slog.Logger

	// handlers stores the registered slog handlers for the logger.
	handlers []slog.Handler
}

// Initialise creates and configures a logger based on the provided configuration.
// This is a test helper function.
func Initialise(ctx context.Context, loggerConfig logger_dto.Config, serverConfig *config.ServerConfig) (*slog.Logger, func(context.Context) error, error) {
	logger_domain.ClearLifecycle()

	handlerList := make([]slog.Handler, 0, len(loggerConfig.Outputs))

	for _, output := range loggerConfig.Outputs {
		handler, err := createOutputHandler(ctx, output, loggerConfig.AddSource)
		if err != nil {
			return nil, nil, fmt.Errorf("creating output handler %q: %w", output.Name, err)
		}
		handlerList = append(handlerList, handler)
	}

	for _, integration := range loggerConfig.Integrations {
		if !integration.Enabled {
			continue
		}
		handler, err := createIntegrationHandler(integration)
		if err != nil {
			return nil, nil, fmt.Errorf("creating integration handler %q: %w", integration.Name, err)
		}
		if handler != nil {
			handlerList = append(handlerList, handler)
		}
	}

	var finalHandler slog.Handler
	if len(handlerList) == 0 {
		levelVar := new(slog.LevelVar)
		levelVar.Set(slog.LevelInfo)
		finalHandler = driver_handlers.NewPrettyHandler(os.Stderr, &driver_handlers.Options{
			Level:     levelVar,
			AddSource: loggerConfig.AddSource,
		})
	} else if len(handlerList) == 1 {
		finalHandler = handlerList[0]
	} else {
		finalHandler = slog.NewMultiHandler(handlerList...)
	}

	otelHandler := logger_domain.NewOTelSlogHandler(finalHandler)
	logger := slog.New(otelHandler)

	slog.SetDefault(logger)

	shutdown := func(ctx context.Context) error {
		return logger_domain.Shutdown(ctx)
	}

	setShutdownFunc(shutdown)

	return logger, shutdown, nil
}

// InitDefaultFactory initialises the default logger factory with the provided
// logger.
//
// Takes logger (*slog.Logger) which is the logger instance to use as default.
func InitDefaultFactory(logger *slog.Logger) {
	logger_domain.InitDefaultFactory(logger)
}

// AddPrettyOutput adds a pretty-formatted console output to the global logger.
//
// Takes level (slog.Level) which sets the minimum log level for this output.
func AddPrettyOutput(level slog.Level) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)
	handler := driver_handlers.NewPrettyHandler(os.Stdout, &driver_handlers.Options{
		Level:     levelVar,
		AddSource: false,
		NoColour:  false,
	})
	addHandler(handler)
}

// AddFileOutput adds a file output to the global logger with the specified
// configuration.
//
// Takes ctx (context.Context) which controls the lifetime of the background
// rotation goroutine.
// Takes name (string) which identifies this output for later reference.
// Takes path (string) which specifies the file path to write logs to.
// Takes level (slog.Level) which sets the minimum log level for this output.
// Takes useJSON (bool) which selects JSON format when true, text format
// when false.
func AddFileOutput(ctx context.Context, name, path string, level slog.Level, useJSON bool) {
	fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
		Directory:  filepath.Dir(path),
		Filename:   filepath.Base(path),
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	})
	if fileError != nil {
		return
	}
	logger_domain.RegisterClosable(fileLogger)

	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   false,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		})
	} else {
		handler = slog.NewTextHandler(fileLogger, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   false,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		})
	}

	addHandler(handler)
}

// ResetAndApplyConfig resets the global logger state and applies a new
// configuration.
//
// Takes loggerConfig (logger_dto.Config) which specifies the new logger settings.
//
// Safe for concurrent use. The function holds the global state mutex while
// clearing and reinitialising the logger.
func ResetAndApplyConfig(loggerConfig logger_dto.Config) {
	globalStateMutex.Lock()
	defer globalStateMutex.Unlock()

	globalState = &loggerState{
		handlers: []slog.Handler{},
	}

	logger_domain.InitDefaultFactory(slog.Default())
}

// GetShutdownFunc returns the current shutdown function, if available.
//
// Returns func(context.Context) error which is the shutdown function, or nil
// if none has been set.
//
// Safe for concurrent use by multiple goroutines.
func GetShutdownFunc() func(context.Context) error {
	shutdownFuncMutex.RLock()
	defer shutdownFuncMutex.RUnlock()
	return shutdownFunc
}

// setShutdownFunc sets the shutdown function to be called during cleanup.
//
// Takes shutdownFunction (func(context.Context) error) which is the function to run on
// shutdown.
//
// Safe for concurrent use; uses a mutex to protect the shutdown function.
func setShutdownFunc(shutdownFunction func(context.Context) error) {
	shutdownFuncMutex.Lock()
	defer shutdownFuncMutex.Unlock()
	shutdownFunc = shutdownFunction
}

// addHandler appends a handler to the global logger state and rebuilds the
// logger.
//
// Takes handler (slog.Handler) which provides the logging output destination.
//
// Concurrency: Safe for concurrent use; protected by globalStateMutex.
func addHandler(handler slog.Handler) {
	globalStateMutex.Lock()
	defer globalStateMutex.Unlock()

	if globalState == nil {
		globalState = &loggerState{}
	}

	globalState.handlers = append(globalState.handlers, handler)

	var finalHandler slog.Handler
	if len(globalState.handlers) == 1 {
		finalHandler = globalState.handlers[0]
	} else {
		finalHandler = slog.NewMultiHandler(globalState.handlers...)
	}

	otelHandler := logger_domain.NewOTelSlogHandler(finalHandler)
	globalState.logger = slog.New(otelHandler)

	slog.SetDefault(globalState.logger)
	logger_domain.InitDefaultFactory(globalState.logger)
}

// createOutputHandler creates a slog.Handler based on the output configuration.
//
// Takes ctx (context.Context) which controls the lifetime of the background
// rotation goroutine for file outputs.
// Takes output (logger_dto.OutputConfig) which specifies the output type,
// format, and destination settings.
// Takes globalAddSource (bool) which sets the default for including source
// location in log entries.
//
// Returns slog.Handler which is configured for the specified output format.
// Returns error when the output type is unknown, the format is unknown, or
// file output is specified without file configuration.
func createOutputHandler(ctx context.Context, output logger_dto.OutputConfig, globalAddSource bool) (slog.Handler, error) {
	level := logger_dto.ParseLogLevel(output.Level, slog.LevelInfo)
	addSource := globalAddSource
	if output.AddSource != nil {
		addSource = *output.AddSource
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	var writer io.Writer
	switch output.Type {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		if output.File == nil {
			return nil, errors.New("file output requires file configuration")
		}
		fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
			Directory:  filepath.Dir(output.File.Path),
			Filename:   filepath.Base(output.File.Path),
			MaxSize:    output.File.MaxSize,
			MaxBackups: output.File.MaxBackups,
			MaxAge:     output.File.MaxAge,
			Compress:   output.File.Compress,
			LocalTime:  output.File.LocalTime,
		})
		if fileError != nil {
			return nil, fmt.Errorf("creating file output: %w", fileError)
		}
		logger_domain.RegisterClosable(fileLogger)
		writer = fileLogger
	default:
		return nil, fmt.Errorf("unknown output type: %s", output.Type)
	}

	switch output.Format {
	case "pretty":
		return driver_handlers.NewPrettyHandler(writer, &driver_handlers.Options{
			Level:     levelVar,
			AddSource: addSource,
			NoColour:  output.NoColour,
		}), nil
	case "json":
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   addSource,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		}), nil
	case "text":
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   addSource,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		}), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", output.Format)
	}
}

// createIntegrationHandler creates a logging handler for the specified
// integration type.
//
// Takes integrationConfig (logger_dto.IntegrationConfig) which specifies the
// integration type and its settings.
//
// Returns slog.Handler which is the configured handler, or nil if the
// integration adapter is not imported.
// Returns error when the configuration is invalid or handler creation fails.
func createIntegrationHandler(integrationConfig logger_dto.IntegrationConfig) (slog.Handler, error) {
	integration := logger_domain.GetIntegration(integrationConfig.Type)
	if integration == nil {
		slog.Info("Integration configured but adapter package not imported",
			slog.String("type", integrationConfig.Type))
		return nil, nil
	}

	var parsedConfig any
	switch integrationConfig.Type {
	case "sentry":
		if integrationConfig.Sentry == nil {
			return nil, errors.New("sentry integration requires sentry configuration")
		}
		parsedConfig = integrationConfig.Sentry
	default:
		return nil, fmt.Errorf("unknown integration type: %s", integrationConfig.Type)
	}

	handler, err := integration.CreateHandler(parsedConfig)
	if err != nil {
		return nil, fmt.Errorf("creating %s handler: %w", integrationConfig.Type, err)
	}
	return handler, nil
}
