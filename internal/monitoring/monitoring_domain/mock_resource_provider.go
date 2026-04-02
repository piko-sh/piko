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

// MockResourceProvider is a test double for ResourceProvider where nil
// function fields return zero values and call counts are tracked atomically.
type MockResourceProvider struct {
	// GetResourcesFunc is the function called by
	// GetResources.
	GetResourcesFunc func() ResourceData

	// GetResourcesCallCount tracks how many times
	// GetResources was called.
	GetResourcesCallCount int64
}

var _ ResourceProvider = (*MockResourceProvider)(nil)

// GetResources delegates to GetResourcesFunc if set.
//
// Returns ResourceData{} if GetResourcesFunc is nil.
func (m *MockResourceProvider) GetResources() ResourceData {
	atomic.AddInt64(&m.GetResourcesCallCount, 1)
	if m.GetResourcesFunc != nil {
		return m.GetResourcesFunc()
	}
	return ResourceData{}
}
