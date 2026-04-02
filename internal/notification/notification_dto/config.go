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

package notification_dto

import "time"

// DispatcherConfig holds settings for the notification dispatcher.
type DispatcherConfig struct {
	// DeadLetterPath is the file path where dead letter queue entries are stored.
	DeadLetterPath string `yaml:"dead_letter_path" json:"dead_letter_path"`

	// BatchSize specifies the number of notifications to process in a batch.
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// FlushInterval specifies how often to send queued notifications; zero or
	// negative values use the default interval.
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`

	// MaxRetries specifies the maximum number of retry attempts for
	// failed notifications.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// InitialDelay is the wait time before the first retry attempt.
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`

	// MaxDelay is the upper limit for delay between retry attempts.
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`

	// BackoffFactor is the multiplier applied to the delay after each retry;
	// defaults to a standard value when zero or negative.
	BackoffFactor float64 `yaml:"backoff_factor" json:"backoff_factor"`

	// CircuitBreakerThreshold is the number of consecutive failures
	// before opening the circuit breaker.
	CircuitBreakerThreshold int `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold"`

	// CircuitBreakerTimeout is how long the circuit breaker stays open before
	// attempting to close. A value of 0 or less uses the default timeout.
	CircuitBreakerTimeout time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout"`

	// CircuitBreakerInterval is the period for which circuit breaker
	// stats are tracked.
	CircuitBreakerInterval time.Duration `yaml:"circuit_breaker_interval" json:"circuit_breaker_interval"`

	// Enabled controls whether the dispatcher is active.
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// ServiceConfig holds settings for the notification service.
type ServiceConfig struct {
	// MaxNotificationsPerSecond limits how many notifications are sent per second.
	// 0 means unlimited.
	MaxNotificationsPerSecond int `yaml:"max_notifications_per_second" json:"max_notifications_per_second"`

	// DefaultTimeout is the time limit for sending a single notification.
	DefaultTimeout time.Duration `yaml:"default_timeout" json:"default_timeout"`
}
