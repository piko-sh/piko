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

package driver_mock_cms

import (
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/collection/collection_dto"
)

// MockCMSRuntimeProvider is a runtime provider that simulates fetching from a CMS.
//
// This provider implements the RuntimeProvider interface and is designed to
// work with the MockCMSProvider build-time provider.
//
// Design Philosophy:
//   - In-memory mock: Stores data in a map for testing
//   - Thread-safe: Uses mutex for concurrent access
//   - JSON-based: Simulates API responses with JSON marshalling
//
// Usage:
//  1. Create provider: runtime := NewMockCMSRuntimeProvider("mock-cms")
//  2. Set mock data: runtime.SetMockData("blog", jsonBytes)
//  3. Register: pikoruntime.RegisterRuntimeProvider(runtime)
//  4. Generated code will call it via pikoruntime.FetchCollection()
type MockCMSRuntimeProvider struct {
	// data maps collection names to their JSON content.
	data map[string][]byte

	// name is the provider identifier returned by the Name method.
	name string

	// mu guards concurrent access to the data map.
	mu sync.RWMutex
}

// NewMockCMSRuntimeProvider creates a new mock CMS runtime provider.
//
// Takes name (string) which specifies the provider name (must match the
// build-time provider name).
//
// Returns *MockCMSRuntimeProvider which is a fully initialised mock provider
// ready for use.
func NewMockCMSRuntimeProvider(name string) *MockCMSRuntimeProvider {
	return &MockCMSRuntimeProvider{
		name: name,
		data: make(map[string][]byte),
	}
}

// Name returns the unique identifier for this provider.
//
// This must match the build-time provider's name so the runtime
// can correctly route fetch requests.
//
// Returns string which is the provider's unique name.
func (p *MockCMSRuntimeProvider) Name() string {
	return p.name
}

// Fetch retrieves collection data at runtime.
//
// Called by pikoruntime.FetchCollection() when generated code needs to fetch
// dynamic data.
//
// Takes collectionName (string) which identifies the collection to fetch.
// Takes target (any) which is a pointer to a slice to populate with the
// fetched data (e.g. *[]Post).
//
// Returns error when JSON unmarshalling of the mock data fails.
//
// Safe for concurrent use. Access is serialised by an internal
// read lock on the data map.
func (p *MockCMSRuntimeProvider) Fetch(
	_ context.Context,
	collectionName string,
	_ *collection_dto.FetchOptions,
	target any,
) error {
	p.mu.RLock()
	jsonData, ok := p.data[collectionName]
	p.mu.RUnlock()

	if !ok {
		jsonData = []byte("[]")
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("unmarshalling mock CMS data for collection '%s': %w",
			collectionName, err)
	}

	return nil
}

// SetMockData sets mock data for a collection.
//
// This is used in tests to simulate CMS responses.
//
// Takes collectionName (string) which is the name of the collection.
// Takes jsonData ([]byte) which is the JSON-encoded slice of items.
//
// Safe for concurrent use; protected by an internal mutex.
func (p *MockCMSRuntimeProvider) SetMockData(collectionName string, jsonData []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[collectionName] = jsonData
}

// ClearMockData removes all mock data.
//
// Use it to reset state between tests.
//
// Thread-safety: Safe to call concurrently.
func (p *MockCMSRuntimeProvider) ClearMockData() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data = make(map[string][]byte)
}
