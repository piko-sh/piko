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

package email_dto

import (
	"testing"
	"time"
)

func TestDefaultDispatcherConfig(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()

	if config.BatchSize != 10 {
		t.Errorf("BatchSize = %d, want 10", config.BatchSize)
	}
	if config.FlushInterval != 30*time.Second {
		t.Errorf("FlushInterval = %v, want 30s", config.FlushInterval)
	}
	if config.QueueSize != 1000 {
		t.Errorf("QueueSize = %d, want 1000", config.QueueSize)
	}
	if config.RetryQueueSize != 500 {
		t.Errorf("RetryQueueSize = %d, want 500", config.RetryQueueSize)
	}
	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.InitialDelay != 5*time.Second {
		t.Errorf("InitialDelay = %v, want 5s", config.InitialDelay)
	}
	if config.MaxDelay != 5*time.Minute {
		t.Errorf("MaxDelay = %v, want 5m", config.MaxDelay)
	}
	if config.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", config.BackoffFactor)
	}
	if !config.DeadLetterQueue {
		t.Error("DeadLetterQueue should be true")
	}
	if config.MaxRetryHeapSize != 50000 {
		t.Errorf("MaxRetryHeapSize = %d, want 50000", config.MaxRetryHeapSize)
	}
	if config.MaxConsecutiveFailures != 5 {
		t.Errorf("MaxConsecutiveFailures = %d, want 5", config.MaxConsecutiveFailures)
	}
	if config.CircuitBreakerTimeout != 60*time.Second {
		t.Errorf("CircuitBreakerTimeout = %v, want 60s", config.CircuitBreakerTimeout)
	}
	if config.CircuitBreakerInterval != 10*time.Second {
		t.Errorf("CircuitBreakerInterval = %v, want 10s", config.CircuitBreakerInterval)
	}
}
