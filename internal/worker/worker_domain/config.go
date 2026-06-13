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
	"time"
)

const (
	// DefaultPromoteInterval is how often the promote loop scans for due scheduled or
	// retryable rows to promote to pending.
	DefaultPromoteInterval   = 500 * time.Millisecond
	DefaultHeartbeatInterval = 5 * time.Minute
)

// WorkersConfig is the node-local timing model the pool and recovery loops read. Every
// field has a default; a partial config is backfilled by WithDefaults, so a caller may
// set only the knobs it cares about and leave the rest at their defaults.
type WorkersConfig struct {
	// PollInterval is the poll floor beneath the notify driver: the longest a claim loop
	// waits between polls when no wake arrives.
	PollInterval time.Duration

	// DefaultJobTimeout is the per-attempt budget applied when a job sets no timeout via its
	// own WithTimeout.
	DefaultJobTimeout time.Duration

	// VisibilityTimeout is the claimed_at age past which the recovery sweep reclaims a
	// running row as orphaned.
	VisibilityTimeout time.Duration

	// RecoveryInterval is how often the periodic recovery sweep scans for stale claims.
	RecoveryInterval time.Duration

	// PromoteInterval is how often the promotion scheduler runs to promote viable scheduled
	// or retryable jobs to pending.
	PromoteInterval time.Duration

	HeartbeatInterval time.Duration
}

// DefaultWorkersConfig returns the node-local timing defaults.
//
// Returns WorkersConfig which is the fully-populated default configuration.
func DefaultWorkersConfig() WorkersConfig {
	return WorkersConfig{
		PollInterval:      defaultPollFloor,
		DefaultJobTimeout: defaultJobTimeout,
		VisibilityTimeout: DefaultVisibilityTimeout,
		RecoveryInterval:  DefaultReclaimInterval,
		PromoteInterval:   DefaultPromoteInterval,
		HeartbeatInterval: DefaultHeartbeatInterval,
	}
}

// WithDefaults backfills any unset (non-positive) field from DefaultWorkersConfig so a
// partial config is always safe to use.
//
// Returns WorkersConfig which is the receiver with every zero field filled in.
func (config WorkersConfig) WithDefaults() WorkersConfig {
	defaults := DefaultWorkersConfig()
	if config.PollInterval <= 0 {
		config.PollInterval = defaults.PollInterval
	}
	if config.DefaultJobTimeout <= 0 {
		config.DefaultJobTimeout = defaults.DefaultJobTimeout
	}
	if config.VisibilityTimeout <= 0 {
		config.VisibilityTimeout = defaults.VisibilityTimeout
	}
	if config.RecoveryInterval <= 0 {
		config.RecoveryInterval = defaults.RecoveryInterval
	}
	if config.PromoteInterval <= 0 {
		config.PromoteInterval = defaults.PromoteInterval
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = defaults.HeartbeatInterval
	}
	return config
}
