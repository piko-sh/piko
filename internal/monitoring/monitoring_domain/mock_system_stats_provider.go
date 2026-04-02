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

package monitoring_domain

import "sync/atomic"

// MockSystemStatsProvider is a test double for SystemStatsProvider where
// nil function fields return zero values and call counts are tracked
// atomically.
type MockSystemStatsProvider struct {
	// GetStatsFunc is the function called by GetStats.
	GetStatsFunc func() SystemStats

	// GetStatsCallCount tracks how many times GetStats
	// was called.
	GetStatsCallCount int64
}

var _ SystemStatsProvider = (*MockSystemStatsProvider)(nil)

// GetStats delegates to GetStatsFunc if set.
//
// Returns SystemStats{} if GetStatsFunc is nil.
func (m *MockSystemStatsProvider) GetStats() SystemStats {
	atomic.AddInt64(&m.GetStatsCallCount, 1)
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return SystemStats{}
}
