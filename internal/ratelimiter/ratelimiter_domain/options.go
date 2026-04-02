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

package ratelimiter_domain

import (
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

// Option configures a Limiter.
type Option func(*Limiter)

// WithClock sets the clock used for time operations, defaulting to
// clock.RealClock() if not set. This is primarily useful for testing.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns Option which configures the limiter to use the given clock.
func WithClock(c clock.Clock) Option {
	return func(l *Limiter) {
		l.clock = c
	}
}

// WithFailPolicy sets the behaviour when the backing store is unavailable.
// The default is FailOpen, which allows requests when the store is unreachable.
//
// Takes policy (ratelimiter_dto.FailPolicy) which specifies the failure
// behaviour.
//
// Returns Option to apply to the limiter.
func WithFailPolicy(policy ratelimiter_dto.FailPolicy) Option {
	return func(l *Limiter) {
		l.failPolicy = policy
	}
}

// WithKeyPrefix sets a prefix prepended to all rate limit keys. This prevents
// key collisions when multiple limiters share the same backing store.
//
// Takes prefix (string) which is prepended to all keys with a colon separator.
//
// Returns Option to apply to the limiter.
func WithKeyPrefix(prefix string) Option {
	return func(l *Limiter) {
		l.keyPrefix = prefix
	}
}

// WithTokenStoreName sets the human-readable name of the token bucket store
// for monitoring inspection. This name appears in CLI output and gRPC
// responses.
//
// Takes name (string) such as "cache", "inmemory", or "noop".
//
// Returns Option to apply to the limiter.
func WithTokenStoreName(name string) Option {
	return func(l *Limiter) {
		l.tokenStoreName = name
	}
}

// WithCounterStoreName sets the human-readable name of the counter store
// for monitoring inspection. This name appears in CLI output and gRPC
// responses.
//
// Takes name (string) such as "cache" or "noop".
//
// Returns Option to apply to the limiter.
func WithCounterStoreName(name string) Option {
	return func(l *Limiter) {
		l.counterStoreName = name
	}
}
