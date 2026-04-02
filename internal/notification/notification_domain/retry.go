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
	"piko.sh/piko/internal/retry"
)

// errorClassifier classifies errors for retry decisions, treating
// notification-specific validation errors as permanent in addition to the
// shared defaults.
var errorClassifier = retry.NewErrorClassifier(
	retry.WithPermanentErrors(
		ErrProviderNotFound,
		ErrNoProviders,
		ErrNoDefaultProvider,
		ErrEmptyMessage,
		ErrEmptyTitle,
		ErrUnsupportedContentType,
		ErrMessageTooLong,
	),
	retry.WithRetryablePatterns(
		"slack api",
		"discord webhook",
		"webhook error",
		"circuit breaker open",
	),
)

// IsRetryableError reports whether an error is temporary and worth retrying.
// It checks for network errors, system call errors, and known retryable
// messages, whilst filtering out permanent errors including validation
// failures and missing provider errors.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error can be retried, false otherwise.
func IsRetryableError(err error) bool {
	return errorClassifier.IsRetryable(err)
}
