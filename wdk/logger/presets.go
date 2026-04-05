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
	"log/slog"
	"os"
	"path/filepath"

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/internal/logrotate"
)

const (
	// defaultMaxSizeMB is the default maximum log file size in megabytes.
	defaultMaxSizeMB = 10

	// defaultMaxBackups is the default number of old log files to keep.
	defaultMaxBackups = 3

	// defaultMaxAgeDays is the number of days to retain old log files.
	defaultMaxAgeDays = 7
)

// OutputOption is a functional option for setting up log outputs.
type OutputOption func(*outputConfig)

// outputConfig holds the resolved settings for an output destination.
type outputConfig struct {
	// level overrides the log level; nil means use LOG_LEVEL env var or default.
	level *slog.Level

	// asJSON enables JSON format for log output; default is text format.
	asJSON bool

	// noColour disables colour output when true.
	noColour bool
}

// resolveLevel returns the effective log level, reading from LOG_LEVEL env
// var if not explicitly set.
//
// Returns slog.Level which is the configured level or the environment default.
func (config *outputConfig) resolveLevel() slog.Level {
	if config.level != nil {
		return *config.level
	}
	return logger_dto.ParseLogLevel(os.Getenv("LOG_LEVEL"), slog.LevelInfo)
}

// WithLevel explicitly sets the minimum log level for this output. If not
// specified, the level defaults to the LOG_LEVEL environment variable, or INFO
// if LOG_LEVEL is not set.
//
// Takes level (slog.Level) which specifies the minimum severity for log output.
//
// Returns OutputOption which configures the log level when applied.
func WithLevel(level slog.Level) OutputOption {
	return func(config *outputConfig) {
		config.level = &level
	}
}

// WithJSON configures the output to use JSON format instead of text/pretty.
//
// Returns OutputOption which applies JSON formatting to the output
// configuration.
func WithJSON() OutputOption {
	return func(config *outputConfig) {
		config.asJSON = true
	}
}

// WithNoColour disables colour output (useful for file outputs or CI
// environments).
//
// Returns OutputOption which configures the output to suppress colour codes.
func WithNoColour() OutputOption {
	return func(config *outputConfig) {
		config.noColour = true
	}
}

// ResetAndApplyConfig resets the logger to its default state and applies
// the provided configuration. It is a convenience function for
// reconfiguring the logger.
func ResetAndApplyConfig(_ logger_dto.Config) {
	ResetLogger()
}

// AddPrettyOutput adds a pretty-formatted console output handler to stdout.
//
// Takes opts (...OutputOption) which configures the output behaviour.
//
// By default, it uses the LOG_LEVEL environment variable (or INFO if not set).
// Use WithLevel() to explicitly override the level.
//
// Examples:
//
//	logger.AddPrettyOutput()
//	logger.AddPrettyOutput(logger.WithLevel(slog.LevelDebug))
func AddPrettyOutput(opts ...OutputOption) {
	config := &outputConfig{}
	for _, opt := range opts {
		opt(config)
	}
	level := config.resolveLevel()

	handler := driver_handlers.NewPrettyHandler(logger_domain.StdoutWriter(), &driver_handlers.Options{
		Level:     level,
		AddSource: true,
		NoColour:  config.noColour,
	})
	AddHandler(handler, nil)
	logger_domain.GetLogger("logger").Debug("Added pretty console output", slog.String("level", level.String()))
}

// AddJSONOutput adds a JSON-formatted console output to stdout.
//
// Takes opts (...OutputOption) which provides optional configuration for the
// output handler.
//
// By default, it uses the LOG_LEVEL environment variable (or INFO if not set).
// Use WithLevel() to explicitly override the level.
func AddJSONOutput(opts ...OutputOption) {
	config := &outputConfig{}
	for _, opt := range opts {
		opt(config)
	}
	level := config.resolveLevel()

	handler := slog.NewJSONHandler(logger_domain.StdoutWriter(), &slog.HandlerOptions{
		Level:       level,
		AddSource:   true,
		ReplaceAttr: logger_domain.ReplaceLevelAttr,
	})
	AddHandler(handler, nil)
	logger_domain.GetLogger("logger").Info("Added JSON console output.", slog.String("level", level.String()))
}

// AddFileOutput adds a file output with rotation support. By default, it uses
// the LOG_LEVEL environment variable, or INFO if not set.
//
// Takes ctx (context.Context) which controls the lifetime of the background
// rotation goroutine.
// Takes name (string) which identifies this output for logging purposes.
// Takes path (string) which specifies the file path for log output.
// Takes opts (...OutputOption) which provides optional settings such as
// WithLevel() to override the log level, or WithJSON() for JSON format.
//
// Examples:
//
//	logger.AddFileOutput(ctx, "app", "/var/log/app.log")
//	logger.AddFileOutput(ctx, "debug", path,
//		logger.WithLevel(slog.LevelDebug))
//	logger.AddFileOutput(ctx, "errors", path,
//		logger.WithLevel(slog.LevelError),
//		logger.WithJSON())
func AddFileOutput(ctx context.Context, name, path string, opts ...OutputOption) {
	config := &outputConfig{noColour: true}
	for _, opt := range opts {
		opt(config)
	}
	level := config.resolveLevel()

	fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
		Directory:  filepath.Dir(path),
		Filename:   filepath.Base(path),
		MaxSize:    defaultMaxSizeMB,
		MaxBackups: defaultMaxBackups,
		MaxAge:     defaultMaxAgeDays,
		Compress:   true,
	})
	if fileError != nil {
		logger_domain.GetLogger("logger").Error("Failed to create file output", logger_domain.Error(fileError))
		return
	}

	var handler slog.Handler
	if config.asJSON {
		handler = slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{Level: level, AddSource: true, ReplaceAttr: logger_domain.ReplaceLevelAttr})
	} else {
		handler = driver_handlers.NewPrettyHandler(fileLogger, &driver_handlers.Options{Level: level, AddSource: true, NoColour: true})
	}

	AddHandler(handler, fileLogger)
	logger_domain.GetLogger("logger").Info("Added file output.", slog.String("name", name), slog.String("path", path), slog.String("level", level.String()))
}

// AddFileOutputOnly clears all existing handlers and adds only a
// file output, bypassing normal handler management to ensure no
// console output for use cases like LSP where stdout/stderr must
// remain clean.
//
// Takes ctx (context.Context) which controls the background cleanup
// goroutine lifetime.
// Takes path (string) which specifies the file path for log output.
// Takes opts (...OutputOption) which provides optional settings such as
// WithLevel() to override the log level, or WithJSON() for JSON format.
func AddFileOutputOnly(ctx context.Context, _, path string, opts ...OutputOption) {
	config := &outputConfig{noColour: true}
	for _, opt := range opts {
		opt(config)
	}
	level := config.resolveLevel()

	fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
		Directory:  filepath.Dir(path),
		Filename:   filepath.Base(path),
		MaxSize:    defaultMaxSizeMB,
		MaxBackups: defaultMaxBackups,
		MaxAge:     defaultMaxAgeDays,
		Compress:   true,
	})
	if fileError != nil {
		logger_domain.GetLogger("logger").Error("Failed to create file output", logger_domain.Error(fileError))
		return
	}

	var handler slog.Handler
	if config.asJSON {
		handler = slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{Level: level, AddSource: true, ReplaceAttr: logger_domain.ReplaceLevelAttr})
	} else {
		handler = driver_handlers.NewPrettyHandler(fileLogger, &driver_handlers.Options{Level: level, AddSource: true, NoColour: true})
	}

	ClearAllHandlers()
	AddHandler(handler, fileLogger)
}
