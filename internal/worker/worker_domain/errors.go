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

package worker_domain

import (
	"errors"
	"time"
)

var (
	// ErrFatal marks a non-retryable failure; the pool sends the row straight to failed.
	ErrFatal = errors.New("worker: fatal error")

	// ErrServiceClosed is returned by operations attempted after the service has stopped.
	ErrServiceClosed = errors.New("worker: service is closed")

	// ErrServiceClosing is the cause used to cancel the run context while the service
	// drains.
	ErrServiceClosing = errors.New("worker: service is shutting down")

	// ErrWorkerNotRegistered is returned when a claimed row's kind has no registered worker.
	ErrWorkerNotRegistered = errors.New("worker: no worker registered for kind")

	// ErrJobNotFound is returned when a job references a row that does not exist.
	ErrJobNotFound = errors.New("worker: job not found")

	// ErrBatchTooLarge guards EnqueueMany against an unbounded caller slice.
	ErrBatchTooLarge = errors.New("worker: batch exceeds the maximum size")

	// ErrNotifierClosed is returned by Subscribe once the notifier has been closed.
	ErrNotifierClosed = errors.New("worker: notifier closed")

	// ErrSnooze marks a deferral, not a failure.
	//
	// A worker returns Snooze(duration), which the dispatcher uses to reschedule the
	// job. A snooze never hits the DLQ or advances the attempt counter.
	ErrSnooze = errors.New("worker: snooze")

	// ErrNotImplemented is returned by an operation the backing store does not support.
	ErrNotImplemented = errors.New("worker: not yet implemented")
)

// Fatal marks an error as non-retryable so the pool sends the row straight to failed.
//
// Takes err (error) which is the underlying failure, or nil for a bare fatal marker.
//
// Returns error which wraps err with ErrFatal and stays non-nil even when err is nil.
func Fatal(err error) error {
	if err == nil {
		return ErrFatal
	}
	return errors.Join(err, ErrFatal)
}

// IsFatal reports whether err, or anything in its chain, is fatal.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when err wraps ErrFatal.
func IsFatal(err error) bool {
	return errors.Is(err, ErrFatal)
}

// snoozeError is the error type carrying a snooze delay.
type snoozeError struct {
	// delay is the requested reschedule duration.
	delay time.Duration
}
