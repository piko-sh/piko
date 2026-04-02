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

package orchestrator_domain

import (
	"testing"
	"time"
)

func TestDefaultDispatcherConfig(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()

	if config.DefaultTimeout != 5*time.Minute {
		t.Errorf("DefaultTimeout: expected 5m, got %v", config.DefaultTimeout)
	}
	if config.DefaultMaxRetries != 3 {
		t.Errorf("DefaultMaxRetries: expected 3, got %d", config.DefaultMaxRetries)
	}
	if config.RecoveryInterval != 30*time.Second {
		t.Errorf("RecoveryInterval: expected 30s, got %v", config.RecoveryInterval)
	}
	if config.StaleTaskThreshold != 10*time.Minute {
		t.Errorf("StaleTaskThreshold: expected 10m, got %v", config.StaleTaskThreshold)
	}
	if config.SyncPersistence {
		t.Error("SyncPersistence: expected false, got true")
	}
	if config.WatermillHighHandlers != 10 {
		t.Errorf("WatermillHighHandlers: expected 10, got %d", config.WatermillHighHandlers)
	}
	if config.WatermillNormalHandlers != 5 {
		t.Errorf("WatermillNormalHandlers: expected 5, got %d", config.WatermillNormalHandlers)
	}
	if config.WatermillLowHandlers != 2 {
		t.Errorf("WatermillLowHandlers: expected 2, got %d", config.WatermillLowHandlers)
	}
}

func TestDispatcherStats_ZeroValues(t *testing.T) {
	t.Parallel()

	stats := DispatcherStats{}

	if stats.HighQueueLen != 0 {
		t.Errorf("HighQueueLen: expected 0, got %d", stats.HighQueueLen)
	}
	if stats.NormalQueueLen != 0 {
		t.Errorf("NormalQueueLen: expected 0, got %d", stats.NormalQueueLen)
	}
	if stats.LowQueueLen != 0 {
		t.Errorf("LowQueueLen: expected 0, got %d", stats.LowQueueLen)
	}
	if stats.ActiveWorkers != 0 {
		t.Errorf("ActiveWorkers: expected 0, got %d", stats.ActiveWorkers)
	}
	if stats.TotalWorkers != 0 {
		t.Errorf("TotalWorkers: expected 0, got %d", stats.TotalWorkers)
	}
	if stats.TasksDispatched != 0 {
		t.Errorf("TasksDispatched: expected 0, got %d", stats.TasksDispatched)
	}
	if stats.TasksCompleted != 0 {
		t.Errorf("TasksCompleted: expected 0, got %d", stats.TasksCompleted)
	}
	if stats.TasksFailed != 0 {
		t.Errorf("TasksFailed: expected 0, got %d", stats.TasksFailed)
	}
	if stats.TasksRetried != 0 {
		t.Errorf("TasksRetried: expected 0, got %d", stats.TasksRetried)
	}
}
