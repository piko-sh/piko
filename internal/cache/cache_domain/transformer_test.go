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

package cache_domain

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

type mockTransformer struct {
	name     string
	typ      cache_dto.TransformerType
	failMode string
	priority int
}

func newMockTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		typ:      cache_dto.TransformerCompression,
		priority: priority,
	}
}

func (m *mockTransformer) Name() string                    { return m.name }
func (m *mockTransformer) Type() cache_dto.TransformerType { return m.typ }
func (m *mockTransformer) Priority() int                   { return m.priority }

func (m *mockTransformer) Transform(_ context.Context, input []byte, _ any) ([]byte, error) {
	if m.failMode == "transform" {
		return nil, errors.New("mock transform error")
	}
	return append([]byte(m.name+":"), input...), nil
}

func (m *mockTransformer) Reverse(_ context.Context, input []byte, _ any) ([]byte, error) {
	if m.failMode == "reverse" {
		return nil, errors.New("mock reverse error")
	}
	prefix := []byte(m.name + ":")
	if len(input) < len(prefix) {
		return nil, errors.New("invalid transformed data")
	}
	return input[len(prefix):], nil
}

func TestTransformerRegistry_Register(t *testing.T) {
	registry := NewTransformerRegistry()
	transformer := newMockTransformer("test-compressor", 100)

	err := registry.Register(context.Background(), transformer)
	if err != nil {
		t.Fatalf("unexpected error registering transformer: %v", err)
	}

	if !registry.Has("test-compressor") {
		t.Error("expected transformer to be registered")
	}
}

func TestTransformerRegistry_RegisterNil(t *testing.T) {
	registry := NewTransformerRegistry()

	err := registry.Register(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when registering nil transformer, got nil")
	}

	expectedMessage := "cannot be nil"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestTransformerRegistry_RegisterEmptyName(t *testing.T) {
	registry := NewTransformerRegistry()
	transformer := newMockTransformer("", 100)

	err := registry.Register(context.Background(), transformer)
	if err == nil {
		t.Fatal("expected error when registering transformer with empty name, got nil")
	}

	expectedMessage := "cannot be empty"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestTransformerRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewTransformerRegistry()

	transformer1 := newMockTransformer("compressor", 100)
	err := registry.Register(context.Background(), transformer1)
	if err != nil {
		t.Fatalf("failed to register first transformer: %v", err)
	}

	transformer2 := newMockTransformer("compressor", 200)
	err = registry.Register(context.Background(), transformer2)
	if err == nil {
		t.Fatal("expected error when registering duplicate transformer name, got nil")
	}

	expectedMessage := "already registered"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestTransformerRegistry_Get(t *testing.T) {
	registry := NewTransformerRegistry()
	transformer := newMockTransformer("test-compressor", 100)

	err := registry.Register(context.Background(), transformer)
	if err != nil {
		t.Fatalf("failed to register transformer: %v", err)
	}

	retrieved, err := registry.Get("test-compressor")
	if err != nil {
		t.Fatalf("unexpected error retrieving transformer: %v", err)
	}
	if retrieved.Name() != "test-compressor" {
		t.Errorf("retrieved transformer name: got %q, want %q", retrieved.Name(), "test-compressor")
	}
}

func TestTransformerRegistry_GetNotFound(t *testing.T) {
	registry := NewTransformerRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent transformer, got nil")
	}

	expectedMessage := "not found"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestTransformerRegistry_Has(t *testing.T) {
	registry := NewTransformerRegistry()
	transformer := newMockTransformer("test-compressor", 100)

	if registry.Has("test-compressor") {
		t.Error("expected Has to return false for unregistered transformer")
	}

	err := registry.Register(context.Background(), transformer)
	if err != nil {
		t.Fatalf("failed to register transformer: %v", err)
	}

	if !registry.Has("test-compressor") {
		t.Error("expected Has to return true for registered transformer")
	}
}

func TestTransformerRegistry_GetNames(t *testing.T) {
	registry := NewTransformerRegistry()

	transformers := []string{"zstd", "aes", "lz4", "brotli"}
	for i, name := range transformers {
		err := registry.Register(context.Background(), newMockTransformer(name, i*100))
		if err != nil {
			t.Fatalf("failed to register transformer %q: %v", name, err)
		}
	}

	names := registry.GetNames()
	expected := []string{"aes", "brotli", "lz4", "zstd"}

	if len(names) != len(expected) {
		t.Fatalf("name count: got %d, want %d", len(names), len(expected))
	}

	for i, name := range expected {
		if names[i] != name {
			t.Errorf("name[%d]: got %q, want %q", i, names[i], name)
		}
	}
}

func TestNewTransformerChain_NilRegistry(t *testing.T) {
	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"test"},
	}

	_, err := NewTransformerChain(nil, config)
	if err == nil {
		t.Fatal("expected error for nil registry, got nil")
	}

	expectedMessage := "registry cannot be nil"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestNewTransformerChain_NilConfig(t *testing.T) {
	registry := NewTransformerRegistry()

	chain, err := NewTransformerChain(registry, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil config: %v", err)
	}

	if !chain.IsEmpty() {
		t.Error("expected empty chain for nil config")
	}
}

func TestNewTransformerChain_PrioritySorting(t *testing.T) {
	registry := NewTransformerRegistry()

	err := registry.Register(context.Background(), newMockTransformer("encrypt", 200))
	if err != nil {
		t.Fatalf("failed to register encrypt transformer: %v", err)
	}
	err = registry.Register(context.Background(), newMockTransformer("compress", 100))
	if err != nil {
		t.Fatalf("failed to register compress transformer: %v", err)
	}

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"encrypt", "compress"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := NewTransformerChain(registry, config)
	if err != nil {
		t.Fatalf("unexpected error creating chain: %v", err)
	}

	names := chain.GetTransformerNames()
	expected := []string{"compress", "encrypt"}

	if len(names) != 2 {
		t.Fatalf("transformer count: got %d, want 2", len(names))
	}

	for i, name := range expected {
		if names[i] != name {
			t.Errorf("transformer[%d]: got %q, want %q", i, names[i], name)
		}
	}
}

func TestNewTransformerChain_UnknownTransformer(t *testing.T) {
	registry := NewTransformerRegistry()

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"nonexistent"},
	}

	_, err := NewTransformerChain(registry, config)
	if err == nil {
		t.Fatal("expected error for unknown transformer, got nil")
	}

	expectedMessage := "failed to resolve transformer"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestTransformerChain_Transform(t *testing.T) {
	registry := NewTransformerRegistry()
	_ = registry.Register(context.Background(), newMockTransformer("compress", 100))
	_ = registry.Register(context.Background(), newMockTransformer("encrypt", 200))

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"compress", "encrypt"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := NewTransformerChain(registry, config)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	ctx := context.Background()
	input := []byte("test data")

	output, err := chain.Transform(ctx, input)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	expected := "encrypt:compress:test data"
	if string(output) != expected {
		t.Errorf("transform output: got %q, want %q", string(output), expected)
	}
}

func TestTransformerChain_Reverse(t *testing.T) {
	registry := NewTransformerRegistry()
	_ = registry.Register(context.Background(), newMockTransformer("compress", 100))
	_ = registry.Register(context.Background(), newMockTransformer("encrypt", 200))

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"compress", "encrypt"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := NewTransformerChain(registry, config)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	ctx := context.Background()

	transformed := []byte("encrypt:compress:test data")

	output, err := chain.Reverse(ctx, transformed)
	if err != nil {
		t.Fatalf("reverse failed: %v", err)
	}

	expected := "test data"
	if string(output) != expected {
		t.Errorf("reverse output: got %q, want %q", string(output), expected)
	}
}

func TestTransformerChain_RoundTrip(t *testing.T) {
	registry := NewTransformerRegistry()
	_ = registry.Register(context.Background(), newMockTransformer("compress", 100))
	_ = registry.Register(context.Background(), newMockTransformer("encrypt", 200))

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"compress", "encrypt"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := NewTransformerChain(registry, config)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	ctx := context.Background()
	original := []byte("test data")

	transformed, err := chain.Transform(ctx, original)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	reversed, err := chain.Reverse(ctx, transformed)
	if err != nil {
		t.Fatalf("reverse failed: %v", err)
	}

	if string(reversed) != string(original) {
		t.Errorf("round-trip failed: got %q, want %q", string(reversed), string(original))
	}
}

func TestTransformerChain_EmptyChain(t *testing.T) {
	registry := NewTransformerRegistry()

	chain, err := NewTransformerChain(registry, nil)
	if err != nil {
		t.Fatalf("failed to create empty chain: %v", err)
	}

	ctx := context.Background()
	input := []byte("test data")

	output, err := chain.Transform(ctx, input)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("empty chain should pass data through: got %q, want %q", string(output), string(input))
	}

	output, err = chain.Reverse(ctx, input)
	if err != nil {
		t.Fatalf("reverse failed: %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("empty chain should pass data through: got %q, want %q", string(output), string(input))
	}
}

func TestTransformerChain_TransformError(t *testing.T) {
	registry := NewTransformerRegistry()

	failing := newMockTransformer("failing", 100)
	failing.failMode = "transform"
	_ = registry.Register(context.Background(), failing)

	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"failing"},
		TransformerOptions:  make(map[string]any),
	}

	chain, err := NewTransformerChain(registry, config)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	ctx := context.Background()
	input := []byte("test data")

	_, err = chain.Transform(ctx, input)
	if err == nil {
		t.Fatal("expected error from failing transformer, got nil")
	}

	if !contains(err.Error(), "failed") {
		t.Errorf("error should indicate transformer failure: %v", err)
	}
}

func TestTransformedValue_MarshalUnmarshal(t *testing.T) {
	original := NewTransformedValue([]byte("test data"), []string{"compress", "encrypt"})

	marshalled, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	unmarshalled, err := UnmarshalTransformedValue(marshalled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if string(unmarshalled.Data) != string(original.Data) {
		t.Errorf("data mismatch: got %q, want %q", string(unmarshalled.Data), string(original.Data))
	}

	if len(unmarshalled.Transformers) != len(original.Transformers) {
		t.Fatalf("transformer count: got %d, want %d", len(unmarshalled.Transformers), len(original.Transformers))
	}

	for i, name := range original.Transformers {
		if unmarshalled.Transformers[i] != name {
			t.Errorf("transformer[%d]: got %q, want %q", i, unmarshalled.Transformers[i], name)
		}
	}

	if unmarshalled.Version != original.Version {
		t.Errorf("version: got %d, want %d", unmarshalled.Version, original.Version)
	}
}

func TestIsTransformedValue(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid transformed value",
			data: []byte(`{"data":"dGVzdA==","transformers":["compress"],"version":1}`),
			want: true,
		},
		{
			name: "invalid JSON",
			data: []byte("not json"),
			want: false,
		},
		{
			name: "valid JSON but no version field",
			data: []byte(`{"data":"test"}`),
			want: false,
		},
		{
			name: "zero version",
			data: []byte(`{"data":"test","version":0}`),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTransformedValue(tc.data)
			if got != tc.want {
				t.Errorf("IsTransformedValue: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalTransformedValue_InvalidJSON(t *testing.T) {
	_, err := UnmarshalTransformedValue([]byte("invalid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	if !errors.Is(err, err) {
		t.Errorf("expected unmarshal error")
	}
}
