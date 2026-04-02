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

// MockProviderRegistry is a test double for ProviderRegistryPort that returns
// zero values from nil function fields and tracks call counts atomically.
type MockProviderRegistry struct {
	// RegisterFunc is the function called by Register.
	RegisterFunc func(provider CollectionProvider) error

	// GetFunc is the function called by Get.
	GetFunc func(name string) (CollectionProvider, bool)

	// ListFunc is the function called by List.
	ListFunc func() []string

	// HasFunc is the function called by Has.
	HasFunc func(name string) bool

	// RegisterCallCount tracks how many times Register
	// was called.
	RegisterCallCount int64

	// GetCallCount tracks how many times Get was called.
	GetCallCount int64

	// ListCallCount tracks how many times List was
	// called.
	ListCallCount int64

	// HasCallCount tracks how many times Has was called.
	HasCallCount int64
}

var _ ProviderRegistryPort = (*MockProviderRegistry)(nil)

// Register delegates to RegisterFunc if set.
//
// Takes provider (CollectionProvider) which is the collection provider
// to register.
//
// Returns nil if RegisterFunc is nil.
func (m *MockProviderRegistry) Register(provider CollectionProvider) error {
	atomic.AddInt64(&m.RegisterCallCount, 1)
	if m.RegisterFunc != nil {
		return m.RegisterFunc(provider)
	}
	return nil
}

// Get delegates to GetFunc if set.
//
// Takes name (string) which identifies the provider by name.
//
// Returns (nil, false) if GetFunc is nil.
func (m *MockProviderRegistry) Get(name string) (CollectionProvider, bool) {
	atomic.AddInt64(&m.GetCallCount, 1)
	if m.GetFunc != nil {
		return m.GetFunc(name)
	}
	return nil, false
}

// List delegates to ListFunc if set.
//
// Returns nil if ListFunc is nil.
func (m *MockProviderRegistry) List() []string {
	atomic.AddInt64(&m.ListCallCount, 1)
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil
}

// Has delegates to HasFunc if set.
//
// Takes name (string) which identifies the provider by name.
//
// Returns false if HasFunc is nil.
func (m *MockProviderRegistry) Has(name string) bool {
	atomic.AddInt64(&m.HasCallCount, 1)
	if m.HasFunc != nil {
		return m.HasFunc(name)
	}
	return false
}
