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

package i18n_domain

import "sync/atomic"

// MockService is a test double for Service where nil function fields
// return zero values and call counts are tracked atomically.
type MockService struct {
	// GetStoreFunc is the function called by GetStore.
	GetStoreFunc func() *Store

	// GetStrBufPoolFunc is the function called by
	// GetStrBufPool.
	GetStrBufPoolFunc func() *StrBufPool

	// DefaultLocaleFunc is the function called by
	// DefaultLocale.
	DefaultLocaleFunc func() string

	// GetStoreCallCount tracks how many times GetStore
	// was called.
	GetStoreCallCount int64

	// GetStrBufPoolCallCount tracks how many times
	// GetStrBufPool was called.
	GetStrBufPoolCallCount int64

	// DefaultLocaleCallCount tracks how many times
	// DefaultLocale was called.
	DefaultLocaleCallCount int64
}

var _ Service = (*MockService)(nil)

// GetStore returns the translation Store for zero-allocation lookups.
//
// Returns *Store, or nil if GetStoreFunc is nil.
func (m *MockService) GetStore() *Store {
	atomic.AddInt64(&m.GetStoreCallCount, 1)
	if m.GetStoreFunc != nil {
		return m.GetStoreFunc()
	}
	return nil
}

// GetStrBufPool returns a shared buffer pool for string rendering.
//
// Returns *StrBufPool, or nil if GetStrBufPoolFunc is nil.
func (m *MockService) GetStrBufPool() *StrBufPool {
	atomic.AddInt64(&m.GetStrBufPoolCallCount, 1)
	if m.GetStrBufPoolFunc != nil {
		return m.GetStrBufPoolFunc()
	}
	return nil
}

// DefaultLocale returns the default locale for fallback resolution.
//
// Returns string, or "" if DefaultLocaleFunc is nil.
func (m *MockService) DefaultLocale() string {
	atomic.AddInt64(&m.DefaultLocaleCallCount, 1)
	if m.DefaultLocaleFunc != nil {
		return m.DefaultLocaleFunc()
	}
	return ""
}
