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

package provider_multilevel

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// circuitBreakerBucketPeriod is the duration of each measurement bucket
// for tracking failure counts.
const circuitBreakerBucketPeriod = 10 * time.Second

// newCircuitBreaker initialises a new gobreaker instance with logging for state
// changes.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes name (string) which identifies the circuit breaker in log
// messages.
// Takes maxFailures (int) which sets the consecutive failure
// threshold before opening.
// Takes timeout (time.Duration) which specifies how long the
// circuit stays open before attempting recovery.
//
// Returns *gobreaker.CircuitBreaker[any] which is the configured
// circuit breaker.
func newCircuitBreaker(ctx context.Context, name string, maxFailures int, timeout time.Duration) *gobreaker.CircuitBreaker[any] {
	var threshold uint32
	if maxFailures > 0 {
		threshold = safeconv.IntToUint32(maxFailures)
	}

	settings := gobreaker.Settings{
		Name:         "cache-l2-" + name,
		MaxRequests:  1,
		Interval:     0,
		Timeout:      timeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= threshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("L2 cache circuit breaker state changed",
				logger_domain.String("cache_name", name),
				logger_domain.String("from_state", from.String()),
				logger_domain.String("to_state", to.String()),
			)
		},
		IsSuccessful: nil,
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[any](settings)
}
