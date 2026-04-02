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

package logger_output_file

import (
	"context"
	"log/slog"
	"path/filepath"

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logrotate"
	"piko.sh/piko/wdk/logger/logger_state"
)

const (
	// defaultMaxSizeMB is the maximum log file size in megabytes before rotation.
	defaultMaxSizeMB = 10

	// defaultMaxBackups is the number of old log files to keep.
	defaultMaxBackups = 3

	// defaultMaxAgeDays is the default maximum age in days for log file retention.
	defaultMaxAgeDays = 7
)

// Config holds the settings for file output.
type Config struct {
	// Path is the file path for log output.
	Path string

	// Level specifies the minimum log level; defaults to slog.LevelInfo.
	Level slog.Level

	// AsJSON enables JSON format output instead of text.
	AsJSON bool
}

// Enable adds a file output handler with automatic rotation.
//
// The file will be rotated when it reaches 10MB, with up to 3 backups kept
// for 7 days. Rotated files are compressed.
//
// Takes ctx (context.Context) which controls the lifetime of the background
// rotation goroutine.
// Takes name (string) which is a descriptive name for this output handler.
// Takes config (Config) which specifies the file output configuration.
func Enable(ctx context.Context, name string, config Config) {
	fileLogger, fileError := logrotate.New(ctx, logrotate.Config{
		Directory:  filepath.Dir(config.Path),
		Filename:   filepath.Base(config.Path),
		MaxSize:    defaultMaxSizeMB,
		MaxBackups: defaultMaxBackups,
		MaxAge:     defaultMaxAgeDays,
		Compress:   true,
	})
	if fileError != nil {
		log.Warn("Failed to create file output", logger_domain.Error(fileError))
		return
	}

	var handler slog.Handler
	if config.AsJSON {
		handler = slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{
			Level:       config.Level,
			AddSource:   true,
			ReplaceAttr: logger_domain.ReplaceLevelAttr,
		})
	} else {
		levelVar := new(slog.LevelVar)
		levelVar.Set(config.Level)
		handler = driver_handlers.NewPrettyHandler(fileLogger, &driver_handlers.Options{
			Level:     levelVar,
			AddSource: true,
			NoColour:  true,
		})
	}

	logger_state.AddHandler(handler, fileLogger)
	log.Info("Added file output.",
		slog.String("name", name),
		slog.String("path", config.Path),
		slog.String("level", config.Level.String()),
	)
}
