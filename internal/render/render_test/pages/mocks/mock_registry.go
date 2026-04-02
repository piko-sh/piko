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

package mocks

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"

	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

// TestingTB defines the testing interface used by mock types.
type TestingTB interface {
	// Helper marks the calling function as a test helper function.
	Helper()

	// Errorf logs an error message with the given format and arguments.
	//
	// Takes format (string) which specifies the message format string.
	// Takes arguments (...any) which provides the values for format placeholders.
	Errorf(format string, arguments ...any)

	// Fatalf logs a formatted message and marks the test as failed, then stops
	// execution.
	//
	// Takes format (string) which is the format string for the message.
	// Takes arguments (...any) which are the values to format into the message.
	Fatalf(format string, arguments ...any)
}

// MockRegistry is a convenience wrapper around render_domain.MockRegistryPort
// for integration tests. It provides OnGetComponent, OnGetSVG, and
// AssertComponentCalled helpers that configure the underlying function-pointer
// mock. Thread-safe via mutex for map access.
type MockRegistry struct {
	t TestingTB

	componentResults map[string]*render_dto.ComponentMetadata

	svgResults map[string]*render_domain.ParsedSvgData

	componentReqs map[string]int64

	svgReqs map[string]int64

	render_domain.MockRegistryPort

	mu sync.Mutex
}

// NewMockRegistry creates a new mock registry for testing.
//
// Takes t (TestingTB) which provides testing context for failure reporting.
//
// Returns *MockRegistry which is ready for use with pre-configured result maps.
func NewMockRegistry(t TestingTB) *MockRegistry {
	m := &MockRegistry{
		t:                t,
		componentResults: make(map[string]*render_dto.ComponentMetadata),
		svgResults:       make(map[string]*render_domain.ParsedSvgData),
		componentReqs:    make(map[string]int64),
		svgReqs:          make(map[string]int64),
	}

	m.GetComponentMetadataFunc = func(_ context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.componentReqs[componentType]++
		if metadata, ok := m.componentResults[componentType]; ok {
			return metadata, nil
		}
		return nil, fmt.Errorf("mock error: component '%s' not found", componentType)
	}

	m.GetAssetRawSVGFunc = func(_ context.Context, assetID string) (*render_domain.ParsedSvgData, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.svgReqs[assetID]++
		if svgData, ok := m.svgResults[assetID]; ok {
			return svgData, nil
		}
		return nil, fmt.Errorf("mock error: SVG asset '%s' not found", assetID)
	}

	m.BulkGetAssetRawSVGFunc = func(_ context.Context, assetIDs []string) (map[string]*render_domain.ParsedSvgData, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		results := make(map[string]*render_domain.ParsedSvgData, len(assetIDs))
		for _, id := range assetIDs {
			m.svgReqs[id]++
			if svgData, ok := m.svgResults[id]; ok {
				results[id] = svgData
			}
		}
		return results, nil
	}

	m.BulkGetComponentMetadataFunc = func(_ context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		results := make(map[string]*render_dto.ComponentMetadata, len(componentTypes))
		for _, ct := range componentTypes {
			m.componentReqs[ct]++
			if metadata, ok := m.componentResults[ct]; ok {
				results[ct] = metadata
			}
		}
		return results, nil
	}

	return m
}

// AssertComponentCalled verifies a component type was requested a specific
// number of times.
//
// Takes componentType (string) which identifies the component to check.
// Takes times (int) which specifies the expected request count.
//
// Safe for concurrent use.
func (m *MockRegistry) AssertComponentCalled(componentType string, times int) {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	actual := m.componentReqs[componentType]
	if actual != int64(times) {
		m.t.Errorf("expected component '%s' to be requested %d time(s), but was %d",
			componentType, times, actual)
	}
}

// OnGetComponent registers a mock result to return for the given component
// type.
//
// Takes componentType (string) which identifies the component to mock.
// Takes result (*render_dto.ComponentMetadata) which is the value to return.
//
// Safe for concurrent use.
func (m *MockRegistry) OnGetComponent(componentType string, result *render_dto.ComponentMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.componentResults[componentType] = result
}

// OnGetSVG registers a mock result for GetAssetRawSVG calls with the given
// asset ID.
//
// Takes assetID (string) which identifies the SVG asset.
// Takes result (*render_domain.ParsedSvgData) which is the mock data to return.
//
// Safe for concurrent use. Pre-computes CachedSymbol to match production
// behaviour.
func (m *MockRegistry) OnGetSVG(assetID string, result *render_domain.ParsedSvgData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if result != nil && result.CachedSymbol == "" {
		result.CachedSymbol = render_domain.ComputeSymbolString(assetID, result)
	}
	m.svgResults[assetID] = result
}

// NewMockCSRF creates a pre-configured security_domain.MockCSRFTokenService
// that returns fixed mock token values matching the legacy MockCSRFService
// behaviour.
//
// Returns *security_domain.MockCSRFTokenService with GenerateCSRFPairFunc
// returning fixed ephemeral and action tokens.
func NewMockCSRF() *security_domain.MockCSRFTokenService {
	return &security_domain.MockCSRFTokenService{
		GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
			buffer.Reset()
			buffer.WriteString("mock-action-token-payload^mock-signature")
			return security_dto.CSRFPair{
				RawEphemeralToken: "mock-ephemeral-token",
				ActionToken:       buffer.Bytes(),
			}, nil
		},
		ValidateCSRFPairFunc: func(_ *http.Request, _ string, _ []byte) (bool, error) {
			return true, nil
		},
		NameFunc: func() string {
			return "mock-csrf"
		},
	}
}
