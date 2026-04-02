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

package email_domain

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/retry"
)

// maxJitterMilliseconds is the upper limit for random jitter when retrying emails.
const maxJitterMilliseconds = 1000

var (
	// ErrRecipientRequired is returned when an email has no recipients.
	ErrRecipientRequired = errors.New("at least one recipient is required")

	// ErrBodyRequired is returned when an email has neither HTML nor plain text body.
	ErrBodyRequired = errors.New("either BodyHTML or BodyPlain must be provided")

	errDispatcherNil = errors.New("dispatcher cannot be nil")

	errNoDispatcher = errors.New("no dispatcher registered")

	errDispatcherRunning = errors.New("dispatcher already running")

	errNoDLQ = errors.New("no dead letter queue configured")

	errTemplaterNotConfigured = errors.New("the templating service has not been configured for the email service")
)

// EmailError represents a single email send failure with retry information.
// It implements fmt.Stringer and tracks attempt history for retry scheduling.
type EmailError struct {
	// FirstAttempt is when the first delivery attempt was made.
	FirstAttempt time.Time

	// LastAttempt is when the last send attempt was made.
	LastAttempt time.Time

	// NextRetry is when this email should be retried; zero means ready now.
	NextRetry time.Time

	// Error is the error that caused the email to fail.
	Error error

	// Email holds the email message that failed to send.
	Email email_dto.SendParams

	// Attempt is the retry attempt number when this error occurred.
	Attempt int
}

// String returns a formatted string representation of the email error.
//
// Returns string which contains the recipient, attempt number, and error
// message.
func (e *EmailError) String() string {
	return fmt.Sprintf("Email to %v (attempt %d): %s", e.Email.To, e.Attempt, e.Error.Error())
}

// MultiError holds a list of email send failures and implements the error
// interface.
type MultiError struct {
	// Errors holds the list of email validation errors.
	Errors []EmailError
}

// Error implements the error interface for MultiError.
//
// Returns string which describes all errors. When one error exists, it returns
// a single message. When there are many errors, it returns a numbered list
// joined by semicolons.
func (me *MultiError) Error() string {
	if len(me.Errors) == 0 {
		return "no errors"
	}

	if len(me.Errors) == 1 {
		return me.Errors[0].String()
	}

	errorStrings := make([]string, 0, len(me.Errors))
	for i := range me.Errors {
		errorStrings = append(errorStrings, (&me.Errors[i]).String())
	}

	return fmt.Sprintf("multiple email send errors (%d): %s",
		len(me.Errors), strings.Join(errorStrings, "; "))
}

// Add appends an EmailError to the MultiError collection.
//
// Takes emailError (*EmailError) which is the error to add.
func (me *MultiError) Add(emailError *EmailError) {
	me.Errors = append(me.Errors, *emailError)
}

// HasErrors reports whether the error collection contains any errors.
//
// Returns bool which is true when at least one error is present.
func (me *MultiError) HasErrors() bool {
	return len(me.Errors) > 0
}

// Count returns the number of errors held in this MultiError.
//
// Returns int which is the count of collected errors.
func (me *MultiError) Count() int {
	return len(me.Errors)
}

// GetEmails returns all failed emails from the collected errors.
//
// Returns []email_dto.SendParams which contains the email parameters for each
// failed send operation.
func (me *MultiError) GetEmails() []email_dto.SendParams {
	emails := make([]email_dto.SendParams, len(me.Errors))
	for i := range me.Errors {
		emails[i] = me.Errors[i].Email
	}
	return emails
}

// GetReadyForRetry returns email errors that are ready for retry.
//
// Takes now (time.Time) which specifies the current time for comparison.
//
// Returns []EmailError which contains errors with NextRetry at or before now.
func (me *MultiError) GetReadyForRetry(now time.Time) []EmailError {
	ready := make([]EmailError, 0, len(me.Errors))
	for i := range me.Errors {
		err := &me.Errors[i]
		if err.NextRetry.IsZero() || err.NextRetry.Before(now) || err.NextRetry.Equal(now) {
			ready = append(ready, *err)
		}
	}
	return ready
}

// Split separates email errors into those ready for retry and those still
// waiting based on their next retry time.
//
// Takes now (time.Time) which specifies the current time for comparison.
//
// Returns readyForRetry ([]EmailError) which contains errors whose retry time
// has passed or is not set.
// Returns stillWaiting ([]EmailError) which contains errors whose retry time
// is in the future.
func (me *MultiError) Split(now time.Time) (readyForRetry, stillWaiting []EmailError) {
	readyForRetry = make([]EmailError, 0, len(me.Errors))
	stillWaiting = make([]EmailError, 0, len(me.Errors))
	for i := range me.Errors {
		err := &me.Errors[i]
		if err.NextRetry.IsZero() || err.NextRetry.Before(now) || err.NextRetry.Equal(now) {
			readyForRetry = append(readyForRetry, *err)
		} else {
			stillWaiting = append(stillWaiting, *err)
		}
	}
	return readyForRetry, stillWaiting
}

// RetryConfig holds settings for the email retry mechanism. It embeds the
// shared retry configuration which provides CalculateNextRetry and ShouldRetry.
type RetryConfig struct {
	retry.Config

	// DeadLetterQueue enables storing emails that fail after all retries.
	DeadLetterQueue bool
}

// flatJitter returns a random duration between 0 and 999 milliseconds,
// ignoring the delay parameter. This provides a flat jitter distribution
// suitable for email retry timing.
//
// Takes _ (time.Duration) which is the calculated delay (unused).
//
// Returns time.Duration which is a random value between 0 and 999 milliseconds.
func flatJitter(_ time.Duration) time.Duration {
	return time.Duration(rand.IntN(maxJitterMilliseconds)) * time.Millisecond //nolint:gosec // jitter, not security
}

// newMultiError creates a new MultiError from a slice of EmailError values.
//
// Takes errs ([]EmailError) which contains the validation errors to wrap.
//
// Returns *MultiError which wraps the errors, or nil if the slice is empty.
func newMultiError(errs []EmailError) *MultiError {
	if len(errs) == 0 {
		return nil
	}
	return &MultiError{Errors: errs}
}

// defaultRetryConfig returns a RetryConfig with sensible default values.
//
// Returns RetryConfig which contains standard retry settings with exponential
// backoff and a dead letter queue enabled.
func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		Config: retry.Config{
			JitterFunc:    flatJitter,
			MaxRetries:    defaultMaxRetries,
			InitialDelay:  defaultInitialDelay,
			MaxDelay:      defaultMaxDelay,
			BackoffFactor: defaultBackoffFactor,
		},
		DeadLetterQueue: true,
	}
}
