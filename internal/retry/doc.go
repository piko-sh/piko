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

// Package retry provides shared retry configuration and error classification
// for operations that may fail with transient errors.
//
// It supplies [Config] for exponential backoff with jitter, and
// [ErrorClassifier] for determining whether an error is worth retrying.
//
// # Design rationale
//
// Exponential backoff avoids thundering herd problems when many clients
// encounter the same transient failure simultaneously. Jitter (a
// randomised fraction of the calculated delay) breaks synchronisation
// between retrying clients so they do not all hit the recovering
// service at the same instant. The error classifier separates transient
// failures from permanent ones to avoid wasting retries on errors that
// will never succeed, such as permission denied or not found.
//
// # Config
//
// Config holds retry policy settings - max retries, initial delay, max delay,
// backoff factor, and an optional jitter function. The default jitter adds
// a random duration up to 10% of the calculated delay to prevent thundering
// herd effects.
//
//	config := retry.Config{
//	    MaxRetries:    3,
//	    InitialDelay:  1 * time.Second,
//	    MaxDelay:      30 * time.Second,
//	    BackoffFactor: 2.0,
//	}
//	nextRetry := config.CalculateNextRetry(attempt, time.Now())
//
// # ErrorClassifier
//
// ErrorClassifier determines if an error is retryable by checking network
// timeouts, syscall errors, and error message patterns. Domains can add
// their own permanent errors and retryable patterns:
//
//	classifier := retry.NewErrorClassifier(
//	    retry.WithPermanentErrors(os.ErrNotExist, os.ErrPermission),
//	    retry.WithRetryablePatterns("slack api", "webhook error"),
//	)
//	if classifier.IsRetryable(err) {
//	    // schedule retry
//	}
//
// # Thread safety
//
// Config methods and ErrorClassifier.IsRetryable are safe for concurrent use.
// The classifier's internal slices are set at construction and never modified.
package retry
