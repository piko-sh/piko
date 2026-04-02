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
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockEncoder[V any] struct {
	marshalErr   error
	unmarshalErr error
	targetType   reflect.Type
}

func newMockEncoder[V any]() *mockEncoder[V] {
	var zero V
	return &mockEncoder[V]{
		targetType: reflect.TypeOf(zero),
	}
}

func newFailingMockEncoder[V any](marshalErr, unmarshalErr error) *mockEncoder[V] {
	var zero V
	return &mockEncoder[V]{
		targetType:   reflect.TypeOf(zero),
		marshalErr:   marshalErr,
		unmarshalErr: unmarshalErr,
	}
}

func (m *mockEncoder[V]) Marshal(value V) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	return fmt.Appendf(nil, "%v", value), nil
}

func (m *mockEncoder[V]) Unmarshal(data []byte, target *V) error {
	if m.unmarshalErr != nil {
		return m.unmarshalErr
	}
	return nil
}

func (m *mockEncoder[V]) MarshalAny(value any) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	v, ok := value.(V)
	if !ok {
		return nil, fmt.Errorf("type mismatch: expected %v, got %T", m.targetType, value)
	}
	return m.Marshal(v)
}

func (m *mockEncoder[V]) UnmarshalAny(data []byte) (any, error) {
	if m.unmarshalErr != nil {
		return nil, m.unmarshalErr
	}
	var result V
	if err := m.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *mockEncoder[V]) HandlesType() reflect.Type {
	return m.targetType
}

type TestUser struct {
	ID   string
	Name string
}

type TestProduct struct {
	ID    int
	Price float64
}

func TestNewEncodingRegistry(t *testing.T) {
	tests := []struct {
		defaultEncoder   AnyEncoder
		name             string
		expectHasDefault bool
	}{
		{
			name:             "with default encoder",
			defaultEncoder:   newMockEncoder[any](),
			expectHasDefault: true,
		},
		{
			name:             "without default encoder",
			defaultEncoder:   nil,
			expectHasDefault: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewEncodingRegistry(tc.defaultEncoder)

			if registry == nil {
				t.Fatal("expected non-nil registry")
			}

			if registry.HasDefault() != tc.expectHasDefault {
				t.Errorf("HasDefault: got %v, want %v", registry.HasDefault(), tc.expectHasDefault)
			}

			if registry.Count() != 0 {
				t.Errorf("expected empty registry, got %d encoders", registry.Count())
			}
		})
	}
}

func TestEncodingRegistry_Register(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	encoder := newMockEncoder[TestUser]()
	err := registry.Register(context.Background(), encoder)
	if err != nil {
		t.Fatalf("unexpected error registering encoder: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("count: got %d, want 1", registry.Count())
	}

	expectedType := reflect.TypeFor[TestUser]()
	if !registry.Has(expectedType) {
		t.Error("expected type to be registered")
	}
}

func TestEncodingRegistry_RegisterNil(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	err := registry.Register(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when registering nil encoder, got nil")
	}

	expectedMessage := "cannot be nil"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestEncodingRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	encoder1 := newMockEncoder[TestUser]()
	err := registry.Register(context.Background(), encoder1)
	if err != nil {
		t.Fatalf("failed to register first encoder: %v", err)
	}

	encoder2 := newMockEncoder[TestUser]()
	err = registry.Register(context.Background(), encoder2)
	if err == nil {
		t.Fatal("expected error when registering duplicate type, got nil")
	}

	expectedMessage := "already registered"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestEncodingRegistry_Get(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	encoder := newMockEncoder[TestUser]()
	err := registry.Register(context.Background(), encoder)
	if err != nil {
		t.Fatalf("failed to register encoder: %v", err)
	}

	user := TestUser{ID: "123", Name: "Alice"}
	retrieved, err := registry.Get(user)
	if err != nil {
		t.Fatalf("unexpected error retrieving encoder: %v", err)
	}

	expectedType := reflect.TypeFor[TestUser]()
	if retrieved.HandlesType() != expectedType {
		t.Errorf("retrieved encoder type: got %v, want %v", retrieved.HandlesType(), expectedType)
	}
}

func TestEncodingRegistry_GetWithDefault(t *testing.T) {
	defaultEncoder := newMockEncoder[any]()
	registry := NewEncodingRegistry(defaultEncoder)

	user := TestUser{ID: "123", Name: "Alice"}

	retrieved, err := registry.Get(user)
	if err != nil {
		t.Fatalf("unexpected error retrieving encoder: %v", err)
	}

	if retrieved != defaultEncoder {
		t.Error("expected default encoder to be returned")
	}
}

func TestEncodingRegistry_GetNoMatch(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	user := TestUser{ID: "123", Name: "Alice"}

	_, err := registry.Get(user)
	if err == nil {
		t.Fatal("expected error when no encoder found, got nil")
	}

	expectedMessage := "no encoder found"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestEncodingRegistry_GetNilValue(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	_, err := registry.Get(nil)
	if err == nil {
		t.Fatal("expected error for nil value, got nil")
	}

	expectedMessage := "cannot encode nil value"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestEncodingRegistry_GetByType(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	encoder := newMockEncoder[TestUser]()
	err := registry.Register(context.Background(), encoder)
	if err != nil {
		t.Fatalf("failed to register encoder: %v", err)
	}

	userType := reflect.TypeFor[TestUser]()
	retrieved, err := registry.GetByType(userType)
	if err != nil {
		t.Fatalf("unexpected error retrieving encoder: %v", err)
	}

	if retrieved.HandlesType() != userType {
		t.Errorf("retrieved encoder type: got %v, want %v", retrieved.HandlesType(), userType)
	}
}

func TestEncodingRegistry_GetByTypeWithDefault(t *testing.T) {
	defaultEncoder := newMockEncoder[any]()
	registry := NewEncodingRegistry(defaultEncoder)

	userType := reflect.TypeFor[TestUser]()

	retrieved, err := registry.GetByType(userType)
	if err != nil {
		t.Fatalf("unexpected error retrieving encoder: %v", err)
	}

	if retrieved != defaultEncoder {
		t.Error("expected default encoder to be returned")
	}
}

func TestEncodingRegistry_GetByTypeNil(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	_, err := registry.GetByType(nil)
	if err == nil {
		t.Fatal("expected error for nil type, got nil")
	}

	expectedMessage := "type cannot be nil"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain %q, got: %v", expectedMessage, err)
	}
}

func TestEncodingRegistry_Has(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	userType := reflect.TypeFor[TestUser]()

	if registry.Has(userType) {
		t.Error("expected Has to return false for unregistered type")
	}

	encoder := newMockEncoder[TestUser]()
	err := registry.Register(context.Background(), encoder)
	if err != nil {
		t.Fatalf("failed to register encoder: %v", err)
	}

	if !registry.Has(userType) {
		t.Error("expected Has to return true for registered type")
	}

	productType := reflect.TypeFor[TestProduct]()
	if registry.Has(productType) {
		t.Error("expected Has to return false for type only covered by default")
	}
}

func TestEncodingRegistry_RegisteredTypes(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	_ = registry.Register(context.Background(), newMockEncoder[TestUser]())
	_ = registry.Register(context.Background(), newMockEncoder[TestProduct]())
	_ = registry.Register(context.Background(), newMockEncoder[string]())

	types := registry.RegisteredTypes()

	if len(types) != 3 {
		t.Fatalf("type count: got %d, want 3", len(types))
	}

	expectedTypes := []string{
		"cache_domain.TestProduct",
		"cache_domain.TestUser",
		"string",
	}

	for i, expectedString := range expectedTypes {
		if !contains(types[i].String(), expectedString) {
			t.Errorf("type[%d]: got %v, want to contain %q", i, types[i], expectedString)
		}
	}
}

func TestEncodingRegistry_Count(t *testing.T) {
	registry := NewEncodingRegistry(newMockEncoder[any]())

	if registry.Count() != 0 {
		t.Errorf("initial count: got %d, want 0", registry.Count())
	}

	_ = registry.Register(context.Background(), newMockEncoder[TestUser]())
	if registry.Count() != 1 {
		t.Errorf("count after one registration: got %d, want 1", registry.Count())
	}

	_ = registry.Register(context.Background(), newMockEncoder[TestProduct]())
	if registry.Count() != 2 {
		t.Errorf("count after two registrations: got %d, want 2", registry.Count())
	}
}

func TestEncodingRegistry_HasDefault(t *testing.T) {
	registryWithDefault := NewEncodingRegistry(newMockEncoder[any]())
	if !registryWithDefault.HasDefault() {
		t.Error("expected HasDefault to return true when default is set")
	}

	registryWithoutDefault := NewEncodingRegistry(nil)
	if registryWithoutDefault.HasDefault() {
		t.Error("expected HasDefault to return false when default is nil")
	}
}

func TestEncodingRegistry_MultipleTypes(t *testing.T) {
	registry := NewEncodingRegistry(nil)

	_ = registry.Register(context.Background(), newMockEncoder[TestUser]())
	_ = registry.Register(context.Background(), newMockEncoder[TestProduct]())
	_ = registry.Register(context.Background(), newMockEncoder[string]())

	tests := []struct {
		value any
		want  reflect.Type
		name  string
	}{
		{name: "user", value: TestUser{ID: "1"}, want: reflect.TypeFor[TestUser]()},
		{name: "product", value: TestProduct{ID: 1}, want: reflect.TypeFor[TestProduct]()},
		{name: "string", value: "test", want: reflect.TypeFor[string]()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoder, err := registry.Get(tc.value)
			if err != nil {
				t.Fatalf("failed to get encoder: %v", err)
			}

			if encoder.HandlesType() != tc.want {
				t.Errorf("type: got %v, want %v", encoder.HandlesType(), tc.want)
			}
		})
	}
}

func TestEncodingRegistry_Concurrent(t *testing.T) {
	registry := NewEncodingRegistry(newMockEncoder[any]())

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for i := range numGoroutines {
		go func(index int) {
			defer wg.Done()

			if index%2 == 0 {
				_ = registry.Register(context.Background(), newMockEncoder[TestUser]())
			} else {
				_ = registry.Register(context.Background(), newMockEncoder[TestProduct]())
			}
		}(i)
	}

	for range numGoroutines {
		go func() {
			defer wg.Done()

			user := TestUser{ID: "123"}
			_, _ = registry.Get(user)

			types := registry.RegisteredTypes()
			_ = types

			_ = registry.Count()
			_ = registry.HasDefault()
		}()
	}

	wg.Wait()

	if registry.Count() < 0 {
		t.Error("count should not be negative after concurrent operations")
	}
}

func TestMockEncoder_MarshalUnmarshal(t *testing.T) {
	encoder := newMockEncoder[TestUser]()

	user := TestUser{ID: "123", Name: "Alice"}

	data, err := encoder.Marshal(user)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty marshalled data")
	}

	var result TestUser
	err = encoder.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestMockEncoder_MarshalAny(t *testing.T) {
	encoder := newMockEncoder[TestUser]()

	tests := []struct {
		value     any
		name      string
		wantError bool
	}{
		{name: "correct type", value: TestUser{ID: "123"}, wantError: false},
		{name: "wrong type", value: TestProduct{ID: 1}, wantError: true},
		{name: "string", value: "test", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encoder.MarshalAny(tc.value)

			if tc.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMockEncoder_HandlesType(t *testing.T) {
	encoder := newMockEncoder[TestUser]()

	expectedType := reflect.TypeFor[TestUser]()
	if encoder.HandlesType() != expectedType {
		t.Errorf("HandlesType: got %v, want %v", encoder.HandlesType(), expectedType)
	}
}

func TestMockEncoder_ErrorPropagation(t *testing.T) {
	marshalErr := errors.New("marshal error")
	unmarshalErr := errors.New("unmarshal error")

	encoder := newFailingMockEncoder[TestUser](marshalErr, unmarshalErr)

	user := TestUser{ID: "123"}

	_, err := encoder.Marshal(user)
	if !errors.Is(err, marshalErr) {
		t.Errorf("marshal error: got %v, want %v", err, marshalErr)
	}

	_, err = encoder.MarshalAny(user)
	if !errors.Is(err, marshalErr) {
		t.Errorf("marshalAny error: got %v, want %v", err, marshalErr)
	}

	var result TestUser
	err = encoder.Unmarshal([]byte("data"), &result)
	if !errors.Is(err, unmarshalErr) {
		t.Errorf("unmarshal error: got %v, want %v", err, unmarshalErr)
	}

	_, err = encoder.UnmarshalAny([]byte("data"))
	if !errors.Is(err, unmarshalErr) {
		t.Errorf("unmarshalAny error: got %v, want %v", err, unmarshalErr)
	}
}

func TestNewEncoder_Creates(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return fmt.Appendf(nil, "%s:%s", v.ID, v.Name), nil },
		func(data []byte, target *TestUser) error { *target = TestUser{ID: "decoded"}; return nil },
	)
	if enc == nil {
		t.Fatal("expected non-nil encoder")
	}

	expectedType := reflect.TypeFor[TestUser]()
	anyEnc, ok := enc.(AnyEncoder)
	if !ok {
		t.Fatal("expected encoder to implement AnyEncoder")
	}
	if anyEnc.HandlesType() != expectedType {
		t.Errorf("HandlesType: got %v, want %v", anyEnc.HandlesType(), expectedType)
	}
}

func TestBaseEncoder_MarshalUnmarshal(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return fmt.Appendf(nil, "%s", v.Name), nil },
		func(data []byte, target *TestUser) error { *target = TestUser{Name: string(data)}; return nil },
	)

	user := TestUser{Name: "Alice"}
	data, err := enc.Marshal(user)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != "Alice" {
		t.Errorf("expected 'Alice', got %q", string(data))
	}

	var result TestUser
	err = enc.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if result.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", result.Name)
	}
}

func TestBaseEncoder_MarshalAny_Success(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return []byte("ok"), nil },
		func(data []byte, target *TestUser) error { return nil },
	)

	anyEnc, ok := enc.(AnyEncoder)
	require.True(t, ok, "expected enc to be AnyEncoder")
	data, err := anyEnc.MarshalAny(TestUser{ID: "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("expected 'ok', got %q", string(data))
	}
}

func TestBaseEncoder_MarshalAny_TypeMismatch(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return []byte("ok"), nil },
		func(data []byte, target *TestUser) error { return nil },
	)

	anyEnc, ok := enc.(AnyEncoder)
	require.True(t, ok, "expected enc to be AnyEncoder")
	_, err := anyEnc.MarshalAny("not a TestUser")
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !contains(err.Error(), "cannot handle value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBaseEncoder_UnmarshalAny_Success(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return nil, nil },
		func(data []byte, target *TestUser) error { *target = TestUser{ID: "decoded"}; return nil },
	)

	anyEnc, ok := enc.(AnyEncoder)
	require.True(t, ok, "expected enc to be AnyEncoder")
	result, err := anyEnc.UnmarshalAny([]byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user, ok := result.(TestUser)
	if !ok {
		t.Fatalf("expected TestUser, got %T", result)
	}
	if user.ID != "decoded" {
		t.Errorf("expected ID 'decoded', got %q", user.ID)
	}
}

func TestBaseEncoder_UnmarshalAny_Error(t *testing.T) {
	enc := NewEncoder(
		func(v TestUser) ([]byte, error) { return nil, nil },
		func(data []byte, target *TestUser) error { return errors.New("decode failed") },
	)

	anyEnc, ok := enc.(AnyEncoder)
	require.True(t, ok, "expected enc to be AnyEncoder")
	_, err := anyEnc.UnmarshalAny([]byte("bad"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "decode failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBaseEncoder_HandlesType(t *testing.T) {
	enc := NewEncoder(
		func(v TestProduct) ([]byte, error) { return nil, nil },
		func(data []byte, target *TestProduct) error { return nil },
	)

	anyEnc, ok := enc.(AnyEncoder)
	require.True(t, ok, "expected enc to be AnyEncoder")
	expected := reflect.TypeFor[TestProduct]()
	if anyEnc.HandlesType() != expected {
		t.Errorf("HandlesType: got %v, want %v", anyEnc.HandlesType(), expected)
	}
}
