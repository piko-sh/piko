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

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger/logger_state"
)

var (
	// AddHandler is an alias for logger_state.AddHandler.
	AddHandler = logger_state.AddHandler

	// AddWrapper is an alias to the logger state AddWrapper function.
	AddWrapper = logger_state.AddWrapper

	// GetShutdownFunc is an alias for logger_state.GetShutdownFunc.
	GetShutdownFunc = logger_state.GetShutdownFunc

	// ResetLogger restores the logger to its default state.
	ResetLogger = logger_state.ResetState

	// ClearAllHandlers is a function that removes all registered log handlers.
	ClearAllHandlers = logger_state.ClearAllHandlers
)

// Apply configures the logger with server-specific settings, including
// OpenTelemetry integration. It sets up trace and metrics exporters based on
// the provided OtelSetupConfig.
//
// Takes otelConfig (driver_handlers.OtelSetupConfig) which specifies the OTLP
// exporter settings for OpenTelemetry integration.
//
// Returns error when the OpenTelemetry setup fails.
func Apply(otelConfig driver_handlers.OtelSetupConfig) error {
	otelShutdown, err := driver_handlers.SetupOtel(context.Background(), otelConfig, nil, nil)
	if err != nil {
		return err
	}
	if otelShutdown != nil {
		logger_state.AddShutdownHook(otelShutdown)
	}

	logger_domain.GetLogger("logger").Debug("Server configuration applied to logger")
	return nil
}
