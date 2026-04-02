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

package logger_domain

import (
	"context"
	"log/slog"
	"time"
)

// GroupedError represents a batched error entry with occurrence tracking.
// Errors are grouped by a hash of their message and source location, allowing
// the notification system to deduplicate identical errors and show counts.
type GroupedError struct {
	// FirstSeen is when this error was first seen.
	FirstSeen time.Time

	// LastSeen is when this error was most recently recorded.
	LastSeen time.Time

	// SourceFile is the path to the file where the error happened.
	SourceFile string

	// LogRecord is the original log record with the level, message, and attributes.
	LogRecord slog.Record

	// SourceLine is the line number in SourceFile where the error happened.
	SourceLine int

	// Count is how many times this error has happened.
	Count int
}

// NotificationPort defines the logger's driven port for sending notifications.
// The notification service implements this interface through an adapter.
type NotificationPort interface {
	// SendGroupedErrors sends a batch of grouped errors as notifications.
	//
	// Takes batch (map[string]*GroupedError) which contains grouped errors to notify.
	//
	// Returns error when the notification operation fails.
	SendGroupedErrors(ctx context.Context, batch map[string]*GroupedError) error
}
