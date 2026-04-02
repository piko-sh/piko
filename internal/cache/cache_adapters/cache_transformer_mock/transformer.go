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

package cache_transformer_mock

import (
	"bytes"
	"context"
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// MockCacheTransformer is a test double that implements CacheTransformerPort.
// It adds "MOCK:<name>:" at the start of values on Transform and removes it on
// Reverse.
type MockCacheTransformer struct {
	// name identifies this transformer for logging and prefix generation.
	name string

	// priority is the execution priority value returned by the Priority method.
	priority int
}

var _ cache_domain.CacheTransformerPort = (*MockCacheTransformer)(nil)

// NewMockCacheTransformer creates a new mock cache transformer.
//
// Takes name (string) which specifies the transformer's identifier.
// Takes priority (int) which sets the transformer's processing order.
//
// Returns *MockCacheTransformer which is the configured mock transformer.
func NewMockCacheTransformer(name string, priority int) *MockCacheTransformer {
	return &MockCacheTransformer{
		name:     name,
		priority: priority,
	}
}

// Name returns the transformer name.
//
// Returns string which is the name of this mock transformer.
func (m *MockCacheTransformer) Name() string {
	return m.name
}

// Type returns the transformer type.
//
// Returns cache_dto.TransformerType which identifies this as a custom
// transformer.
func (*MockCacheTransformer) Type() cache_dto.TransformerType {
	return cache_dto.TransformerCustom
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the execution order.
func (m *MockCacheTransformer) Priority() int {
	return m.priority
}

// Transform prepends "MOCK:<name>:" to the input bytes.
//
// Takes input ([]byte) which contains the data to transform.
//
// Returns []byte which contains the prefixed data.
// Returns error which is always nil for this mock
// implementation.
func (m *MockCacheTransformer) Transform(ctx context.Context, input []byte, _ any) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)
	prefix := fmt.Appendf(nil, "MOCK:%s:", m.name)
	result := make([]byte, 0, len(prefix)+len(input))
	result = append(result, prefix...)
	result = append(result, input...)

	l.Trace("Applied mock transformation",
		logger_domain.String("transformer", m.name),
		logger_domain.Int("input_size", len(input)),
		logger_domain.Int("output_size", len(result)))

	return result, nil
}

// Reverse strips the "MOCK:<name>:" prefix from the input bytes.
//
// Takes input ([]byte) which contains the prefixed data to
// reverse.
//
// Returns []byte which contains the data with the prefix removed.
// Returns error when the expected prefix is not found.
func (m *MockCacheTransformer) Reverse(ctx context.Context, input []byte, _ any) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)
	prefix := fmt.Appendf(nil, "MOCK:%s:", m.name)

	if !bytes.HasPrefix(input, prefix) {
		return nil, fmt.Errorf("mock transformer '%s': expected prefix %q not found", m.name, string(prefix))
	}

	result := input[len(prefix):]

	l.Trace("Reversed mock transformation",
		logger_domain.String("transformer", m.name),
		logger_domain.Int("input_size", len(input)),
		logger_domain.Int("output_size", len(result)))

	return result, nil
}
