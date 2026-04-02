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

package collection_domain

import "sync/atomic"

// MockSearchIndexLoader is a test double for SearchIndexLoaderPort that
// returns zero values from nil function fields and tracks call counts
// atomically.
type MockSearchIndexLoader struct {
	// GetIndexFunc is the function called by GetIndex.
	GetIndexFunc func(collectionName, searchMode string) (any, error)

	// GetIndexCallCount tracks how many times GetIndex
	// was called.
	GetIndexCallCount int64
}

var _ SearchIndexLoaderPort = (*MockSearchIndexLoader)(nil)

// GetIndex delegates to GetIndexFunc if set.
//
// Takes collectionName (string) which identifies the collection by name.
// Takes searchMode (string) which specifies the search mode to use.
//
// Returns (nil, nil) if GetIndexFunc is nil.
func (m *MockSearchIndexLoader) GetIndex(collectionName, searchMode string) (any, error) {
	atomic.AddInt64(&m.GetIndexCallCount, 1)
	if m.GetIndexFunc != nil {
		return m.GetIndexFunc(collectionName, searchMode)
	}
	return nil, nil
}
