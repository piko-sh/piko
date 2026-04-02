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

package watermill

import (
	"github.com/ThreeDotsLabs/watermill"
	"piko.sh/piko/internal/logger/logger_domain"
)

// Adapter connects Piko's logger to Watermill's LoggerAdapter interface.
// It allows Watermill components to use Piko's logging system.
type Adapter struct {
	// logger provides structured logging for the Watermill adapter.
	logger logger_domain.Logger
}

// Error logs an error-level message with fields.
//
// Takes message (string) which is the message to log.
// Takes err (error) which is the error to include in the log entry.
// Takes fields (watermill.LogFields) which provides additional context.
func (a *Adapter) Error(message string, err error, fields watermill.LogFields) {
	a.logger.Error(message, a.convertFields(fields, logger_domain.Error(err))...)
}

// Info logs an info-level message with fields.
// Watermill's internal logs are framework internals, so we map them to
// Internal.
//
// Takes message (string) which is the message to log.
// Takes fields (watermill.LogFields) which provides structured context.
func (a *Adapter) Info(message string, fields watermill.LogFields) {
	a.logger.Internal(message, a.convertFields(fields)...)
}

// Debug logs a debug-level message with fields.
// Watermill's debug logs are framework internals, so we map them to Internal.
//
// Takes message (string) which is the message to log.
// Takes fields (watermill.LogFields) which contains structured data for the log
// entry.
func (a *Adapter) Debug(message string, fields watermill.LogFields) {
	a.logger.Internal(message, a.convertFields(fields)...)
}

// Trace logs a trace-level message with fields.
// Watermill's trace logs are deep framework internals, so we map them to Trace.
//
// Takes message (string) which is the message to log.
// Takes fields (watermill.LogFields) which contains structured data for the log
// entry.
func (a *Adapter) Trace(message string, fields watermill.LogFields) {
	a.logger.Trace(message, a.convertFields(fields)...)
}

// With returns a new logger with the given fields added to all subsequent
// log calls.
//
// Takes fields (watermill.LogFields) which specifies the key-value pairs to
// add to each log entry.
//
// Returns watermill.LoggerAdapter which is a new adapter that includes the
// additional fields.
func (a *Adapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &Adapter{
		logger: a.logger.With(a.convertFields(fields)...),
	}
}

// convertFields converts Watermill's LogFields to Piko's logger fields.
//
// Takes fields (watermill.LogFields) which contains the Watermill log fields
// to convert.
// Takes extraFields (...logger_domain.Attr) which provides additional fields
// to append to the result.
//
// Returns []logger_domain.Attr which contains the converted and merged fields.
func (*Adapter) convertFields(fields watermill.LogFields, extraFields ...logger_domain.Attr) []logger_domain.Attr {
	pikoFields := make([]logger_domain.Attr, 0, len(fields)+len(extraFields))

	for key, value := range fields {
		switch v := value.(type) {
		case string:
			pikoFields = append(pikoFields, logger_domain.String(key, v))
		case int:
			pikoFields = append(pikoFields, logger_domain.Int(key, v))
		case int64:
			pikoFields = append(pikoFields, logger_domain.Int64(key, v))
		case bool:
			pikoFields = append(pikoFields, logger_domain.Bool(key, v))
		case error:
			pikoFields = append(pikoFields, logger_domain.Error(v))
		default:
			pikoFields = append(pikoFields, logger_domain.Field(key, v))
		}
	}

	pikoFields = append(pikoFields, extraFields...)
	return pikoFields
}

// NewAdapter creates a new Watermill logger adapter that wraps a Piko logger.
// Watermill's LoggerAdapter interface has no context.Context, so the adapter
// must hold a concrete logger instance.
//
// Takes logger (logger_domain.Logger) which is the Piko logger to adapt.
//
// Returns watermill.LoggerAdapter which wraps the Piko logger for use with
// Watermill event handling.
func NewAdapter(logger logger_domain.Logger) watermill.LoggerAdapter {
	return &Adapter{logger: logger}
}
