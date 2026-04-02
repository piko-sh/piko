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

package notification_domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrProviderNotFound is returned when a requested provider is not registered.
	ErrProviderNotFound = errors.New("notification provider not found")

	// ErrNoProviders is returned when no providers are registered and no default
	// exists.
	ErrNoProviders = errors.New("no notification providers registered")

	// ErrNoDefaultProvider is returned when no default provider has been set.
	ErrNoDefaultProvider = errors.New("no default notification provider set")

	// ErrProviderAlreadyExists is returned when attempting to register a provider
	// with a name that already exists.
	ErrProviderAlreadyExists = errors.New("notification provider already exists")

	// ErrInvalidConfig is returned when the notification settings are not valid.
	ErrInvalidConfig = errors.New("invalid notification configuration")

	// ErrNoDispatcher is returned when attempting dispatcher operations without a
	// registered dispatcher.
	ErrNoDispatcher = errors.New("no notification dispatcher registered")

	// ErrDispatcherAlreadyRunning is returned when attempting to start an
	// already-running dispatcher.
	ErrDispatcherAlreadyRunning = errors.New("notification dispatcher is already running")

	// ErrDispatcherNotRunning is returned when attempting to stop a dispatcher
	// that is not running.
	ErrDispatcherNotRunning = errors.New("notification dispatcher is not running")

	// ErrCircuitBreakerOpen is returned when the circuit breaker is open and
	// preventing sends.
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// ErrEmptyMessage is returned when attempting to send a notification with no
	// message.
	ErrEmptyMessage = errors.New("notification message cannot be empty")

	// ErrEmptyTitle is returned when attempting to send a notification with no
	// title.
	ErrEmptyTitle = errors.New("notification title cannot be empty")

	// ErrUnsupportedContentType is returned when a provider does not support the
	// requested content type.
	ErrUnsupportedContentType = errors.New("provider does not support this content type")

	// ErrMessageTooLong is returned when a message exceeds the provider's maximum
	// length.
	ErrMessageTooLong = errors.New("notification message exceeds provider maximum length")

	// ErrNotificationEmpty is returned when a notification has neither a title
	// nor a message.
	ErrNotificationEmpty = errors.New("notification must have either a title or message")

	errDispatcherNil = errors.New("dispatcher cannot be nil")
)

// MultiError represents multiple errors that occurred during a multi-cast send.
type MultiError struct {
	// Errors holds the individual errors that occurred.
	Errors []error
}

// Error returns all collected error messages as a single string.
//
// Returns string which contains "no errors" when empty, the single error
// message when only one error exists, or all messages joined by semicolons
// with a count prefix when multiple errors exist.
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%d errors occurred: ", len(e.Errors))
	for i, err := range e.Errors {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap returns the underlying errors for error inspection.
//
// Returns []error which contains the collected errors.
func (e *MultiError) Unwrap() []error {
	return e.Errors
}

// HasErrors reports whether the error collection contains any errors.
//
// Returns bool which is true if there is at least one error stored.
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ProviderError wraps an error with the provider name for clearer error
// messages. It implements the error interface.
type ProviderError struct {
	// Err is the underlying error wrapped by this provider error.
	Err error

	// Provider is the name of the provider that caused the error.
	Provider string
}

// Error returns a string representation of the provider error.
//
// Returns string which contains the provider name and underlying error.
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %q: %v", e.Provider, e.Err)
}

// Unwrap returns the underlying error.
//
// Returns error which is the wrapped error, or nil if none exists.
func (e *ProviderError) Unwrap() error {
	return e.Err
}
