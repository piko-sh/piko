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
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

func TestNewCacheBuilder(t *testing.T) {
	service := NewService("")

	builder := NewCacheBuilder[string, string](service)

	if builder == nil {
		t.Fatal("expected non-nil builder")
	}

	if builder.service != service {
		t.Error("builder should reference the provided service")
	}
}

func TestCacheBuilder_Provider(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.Provider("redis")

	if result != builder {
		t.Error("WithProvider should return the same builder instance")
	}
}

func TestCacheBuilder_MethodChaining(t *testing.T) {
	service := NewService("")

	builder := NewCacheBuilder[string, string](service).
		Provider("otter").
		MaximumSize(1000).
		InitialCapacity(100)

	if builder == nil {
		t.Fatal("method chaining should return non-nil builder")
	}
}

func TestCacheBuilder_MaximumSize(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.MaximumSize(1000)

	if result != builder {
		t.Error("WithMaximumSize should return the same builder instance")
	}
}

func TestCacheBuilder_MaximumWeight(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.MaximumWeight(5000)

	if result != builder {
		t.Error("WithMaximumWeight should return the same builder instance")
	}
}

func TestCacheBuilder_Weigher(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	weigher := func(k string, v string) uint32 {
		return uint32(len(k) + len(v))
	}

	result := builder.Weigher(weigher)

	if result != builder {
		t.Error("WithWeigher should return the same builder instance")
	}
}

func TestCacheBuilder_InitialCapacity(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.InitialCapacity(500)

	if result != builder {
		t.Error("WithInitialCapacity should return the same builder instance")
	}
}

func TestCacheBuilder_WithOptions(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	options := map[string]any{
		"address": "localhost:6379",
		"timeout": "5s",
	}

	result := builder.Options(options)

	if result != builder {
		t.Error("WithOptions should return the same builder instance")
	}
}

func TestCacheBuilder_Transformer(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.Transformer("zstd")

	if result != builder {
		t.Error("WithTransformer should return the same builder instance")
	}

	config := map[string]any{"level": 3}
	result = builder.Transformer("aes", config)

	if result != builder {
		t.Error("WithTransformer with config should return the same builder instance")
	}
}

func TestCacheBuilder_WithCompression(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.Compression()

	if result != builder {
		t.Error("WithCompression should return the same builder instance")
	}
}

func TestCacheBuilder_WithEncryption(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.Encryption()

	if result != builder {
		t.Error("Encryption should return the same builder instance")
	}
}

func TestCacheBuilder_WithEncryptionWithService(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	result := builder.EncryptionWithService("mock-crypto-service")

	if result != builder {
		t.Error("EncryptionWithService should return the same builder instance")
	}
}

func TestCacheBuilder_Encoder(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	encoder := newMockEncoder[string]()

	result := builder.Encoder(encoder)

	if result != builder {
		t.Error("WithEncoder should return the same builder instance")
	}
}

func TestCacheBuilder_DefaultEncoder(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)

	encoder := newMockEncoder[any]()

	result := builder.DefaultEncoder(encoder)

	if result != builder {
		t.Error("WithDefaultEncoder should return the same builder instance")
	}
}

func TestCacheBuilder_MultipleTransformers(t *testing.T) {
	service := NewService("")

	builder := NewCacheBuilder[string, string](service).
		Transformer("compress").
		Transformer("encrypt")

	if builder == nil {
		t.Fatal("expected non-nil builder after adding multiple transformers")
	}
}

func TestCacheBuilder_MultipleEncoders(t *testing.T) {
	service := NewService("")

	userEncoder := newMockEncoder[TestUser]()
	productEncoder := newMockEncoder[TestProduct]()

	builder := NewCacheBuilder[string, string](service).
		Encoder(userEncoder).
		Encoder(productEncoder)

	if builder == nil {
		t.Fatal("expected non-nil builder after adding multiple encoders")
	}
}

func TestCacheBuilder_Clone(t *testing.T) {
	service := NewService("")

	original := NewCacheBuilder[string, string](service).
		Provider("redis").
		MaximumSize(1000).
		InitialCapacity(100)

	clone := original.Clone()

	if clone == nil {
		t.Fatal("expected non-nil cloned builder")
	}

	if clone == original {
		t.Error("cloned builder should be a different instance")
	}

	clone.MaximumSize(2000)
}

func TestCacheBuilder_BuildWithMockProvider(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		MaximumSize(100)

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	defer func() { _ = cache.Close(context.Background()) }()
}

func TestCacheBuilder_BuildWithoutProvider(t *testing.T) {
	service := NewService("")

	builder := NewCacheBuilder[string, string](service)

	cache, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected error when building without provider, got nil")
	}

	if cache != nil {
		t.Error("expected nil cache on error")
	}
}

func TestCacheBuilder_BuildWithInvalidConfig(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		MaximumSize(100).
		MaximumWeight(1000).
		Weigher(func(k, v string) uint32 { return 1 })

	cache, err := builder.Build(context.Background())
	if err == nil {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		t.Fatal("expected validation error for conflicting config, got nil")
	}

	expectedMessage := "MaximumSize"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should relate to configuration conflict, got: %v", err)
	}
}

func TestCacheBuilder_BuildWithMaximumWeightRequiresWeigher(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		MaximumWeight(1000)

	cache, err := builder.Build(context.Background())
	if err == nil {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		t.Fatal("expected error for MaximumWeight without Weigher, got nil")
	}

	expectedMessage := "Weigher"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should mention Weigher requirement, got: %v", err)
	}
}

func TestCacheBuilder_ComplexConfiguration(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		Provider("mock").
		MaximumSize(5000).
		InitialCapacity(1000)

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build failed with complex config: %v", err)
	}

	if cache == nil {
		t.Fatal("expected non-nil cache from complex config")
	}

	defer func() { _ = cache.Close(context.Background()) }()
}

func TestRegisterTransformerBlueprint(t *testing.T) {
	transformerBlueprintsMutex.Lock()
	originalBlueprints := transformerBlueprints
	transformerBlueprints = make(map[string]TransformerBlueprintFactory)
	transformerBlueprintsMutex.Unlock()
	defer func() {
		transformerBlueprintsMutex.Lock()
		transformerBlueprints = originalBlueprints
		transformerBlueprintsMutex.Unlock()
	}()

	blueprintName := "test-blueprint-unique-name"

	factory := func(config any) (CacheTransformerPort, error) {
		return newMockTransformer("test", 100), nil
	}

	RegisterTransformerBlueprint(blueprintName, factory)

	retrieved, exists := getTransformerBlueprint(blueprintName)
	if !exists {
		t.Error("expected blueprint to be registered")
	}
	if retrieved == nil {
		t.Error("expected non-nil factory")
	}
}

func Test_getTransformerBlueprint_NotFound(t *testing.T) {
	_, exists := getTransformerBlueprint("nonexistent-blueprint-name-12345")
	if exists {
		t.Error("expected false for nonexistent blueprint")
	}
}

func TestCacheBuilder_BuildWithTransformerError(t *testing.T) {
	service := NewService("mock")

	provider := newMockTestProvider("mock")
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		Transformer("nonexistent-transformer-xyz")

	cache, err := builder.Build(context.Background())
	if err == nil {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		t.Fatal("expected error for nonexistent transformer, got nil")
	}
}

func TestCacheBuilder_ExpiryCalculator(t *testing.T) {
	service := NewService("")

	expiryCalc := &mockExpiryCalculator[string, string]{}

	builder := NewCacheBuilder[string, string](service).
		ExpiryCalculator(expiryCalc)

	if builder == nil {
		t.Fatal("expected non-nil builder after WithExpiryCalculator")
	}
}

func TestCacheBuilder_RefreshCalculator(t *testing.T) {
	service := NewService("")

	refreshCalc := &mockRefreshCalculator[string, string]{}

	builder := NewCacheBuilder[string, string](service).
		RefreshCalculator(refreshCalc)

	if builder == nil {
		t.Fatal("expected non-nil builder after WithRefreshCalculator")
	}
}

type mockExpiryCalculator[K comparable, V any] struct{}

func (m *mockExpiryCalculator[K, V]) ExpireAfterCreate(entry cache_dto.Entry[K, V]) time.Duration {
	return time.Hour
}

func (m *mockExpiryCalculator[K, V]) ExpireAfterUpdate(entry cache_dto.Entry[K, V], oldValue V) time.Duration {
	return time.Hour
}

func (m *mockExpiryCalculator[K, V]) ExpireAfterRead(entry cache_dto.Entry[K, V]) time.Duration {
	return time.Hour
}

type mockRefreshCalculator[K comparable, V any] struct{}

func (m *mockRefreshCalculator[K, V]) RefreshAfterCreate(entry cache_dto.Entry[K, V]) time.Duration {
	return time.Minute
}

func (m *mockRefreshCalculator[K, V]) RefreshAfterUpdate(entry cache_dto.Entry[K, V], oldValue V) time.Duration {
	return time.Minute
}

func (m *mockRefreshCalculator[K, V]) RefreshAfterRead(entry cache_dto.Entry[K, V]) time.Duration {
	return time.Minute
}

func (m *mockRefreshCalculator[K, V]) RefreshAfterReload(entry cache_dto.Entry[K, V], oldValue V) time.Duration {
	return time.Minute
}

func (m *mockRefreshCalculator[K, V]) RefreshAfterReloadFailure(entry cache_dto.Entry[K, V], err error) time.Duration {
	return time.Minute
}

func TestCacheBuilder_OnDeletion(t *testing.T) {
	service := NewService("")

	onDeletion := func(e cache_dto.DeletionEvent[string, string]) {}

	builder := NewCacheBuilder[string, string](service).
		OnDeletion(onDeletion)

	if builder == nil {
		t.Fatal("expected non-nil builder after WithOnDeletion")
	}
}

func TestCacheBuilder_ProviderErrorPropagation(t *testing.T) {
	service := NewService("mock")

	providerErr := errors.New("provider initialisation failed")
	provider := newMockTestProviderWithError("mock", providerErr)
	err := service.RegisterProvider(context.Background(), "mock", provider)
	if err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	builder := NewCacheBuilder[string, string](service)

	cache, err := builder.Build(context.Background())
	if err == nil {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		t.Fatal("expected provider error to propagate, got nil")
	}

	expectedMessage := "provider initialisation failed"
	if !contains(err.Error(), expectedMessage) {
		t.Errorf("error should contain provider error message, got: %v", err)
	}
}

func TestCacheBuilder_Namespace(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.Namespace("users")
	if result != builder {
		t.Error("Namespace should return the same builder instance")
	}
	if builder.namespace != "users" {
		t.Errorf("expected namespace 'users', got %q", builder.namespace)
	}
}

func TestCacheBuilder_FactoryBlueprint_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.FactoryBlueprint("artefact-metadata")
	if result != builder {
		t.Error("FactoryBlueprint should return the same builder instance")
	}
	if builder.factoryBlueprint != "artefact-metadata" {
		t.Errorf("expected factoryBlueprint 'artefact-metadata', got %q", builder.factoryBlueprint)
	}
}

func TestCacheBuilder_MultiLevel_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.MultiLevel("otter", "redis")
	if result != builder {
		t.Error("MultiLevel should return the same builder instance")
	}
	if !builder.isMultiLevel {
		t.Error("expected isMultiLevel to be true")
	}
	if builder.l1ProviderName != "otter" {
		t.Errorf("expected l1ProviderName 'otter', got %q", builder.l1ProviderName)
	}
	if builder.l2ProviderName != "redis" {
		t.Errorf("expected l2ProviderName 'redis', got %q", builder.l2ProviderName)
	}
	if builder.l2MaxFailures != defaultL2MaxFailures {
		t.Errorf("expected default l2MaxFailures %d, got %d", defaultL2MaxFailures, builder.l2MaxFailures)
	}
	if builder.l2OpenTimeout != defaultL2OpenTimeoutSeconds*time.Second {
		t.Errorf("expected default l2OpenTimeout, got %v", builder.l2OpenTimeout)
	}
}

func TestCacheBuilder_L1Options_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	opts := map[string]any{"pool_size": 10}
	result := builder.L1Options(opts)
	if result != builder {
		t.Error("L1Options should return the same builder instance")
	}
}

func TestCacheBuilder_L2Options_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	opts := map[string]any{"address": "localhost:6379"}
	result := builder.L2Options(opts)
	if result != builder {
		t.Error("L2Options should return the same builder instance")
	}
}

func TestCacheBuilder_L2CircuitBreaker_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.L2CircuitBreaker(3, 60*time.Second)
	if result != builder {
		t.Error("L2CircuitBreaker should return the same builder instance")
	}
	if builder.l2MaxFailures != 3 {
		t.Errorf("expected l2MaxFailures 3, got %d", builder.l2MaxFailures)
	}
	if builder.l2OpenTimeout != 60*time.Second {
		t.Errorf("expected l2OpenTimeout 60s, got %v", builder.l2OpenTimeout)
	}
}

func TestCacheBuilder_OnAtomicDeletion_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	callback := func(e cache_dto.DeletionEvent[string, string]) {}
	result := builder.OnAtomicDeletion(callback)
	if result != builder {
		t.Error("OnAtomicDeletion should return the same builder instance")
	}
	if builder.onAtomicDeletion == nil {
		t.Error("expected onAtomicDeletion to be set")
	}
}

func TestCacheBuilder_StatsRecorder_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	recorder := &mockStatsRecorder{}
	result := builder.StatsRecorder(recorder)
	if result != builder {
		t.Error("StatsRecorder should return the same builder instance")
	}
}

func TestCacheBuilder_Executor_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	executor := func(callback func()) { callback() }
	result := builder.Executor(executor)
	if result != builder {
		t.Error("Executor should return the same builder instance")
	}
}

func TestCacheBuilder_Clock_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	clock := &mockClock{}
	result := builder.Clock(clock)
	if result != builder {
		t.Error("Clock should return the same builder instance")
	}
}

func TestCacheBuilder_Searchable_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	schema := &cache_dto.SearchSchema{}
	result := builder.Searchable(schema)
	if result != builder {
		t.Error("Searchable should return the same builder instance")
	}
}

func TestCacheBuilder_Logger_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.Logger(nil)
	if result != builder {
		t.Error("Logger should return the same builder instance")
	}
}

func TestCacheBuilder_Expiration_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.Expiration(10 * time.Minute)
	if result != builder {
		t.Error("Expiration should return the same builder instance")
	}
	if builder.expiryCalculator == nil {
		t.Error("expected expiryCalculator to be set")
	}
}

func TestCacheBuilder_WriteExpiration_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.WriteExpiration(5 * time.Minute)
	if result != builder {
		t.Error("WriteExpiration should return the same builder instance")
	}
	if _, ok := builder.expiryCalculator.(*writeExpiryCalculator[string, string]); !ok {
		t.Errorf("expected writeExpiryCalculator, got %T", builder.expiryCalculator)
	}
}

func TestCacheBuilder_AccessExpiration_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	result := builder.AccessExpiration(30 * time.Minute)
	if result != builder {
		t.Error("AccessExpiration should return the same builder instance")
	}
	if _, ok := builder.expiryCalculator.(*accessExpiryCalculator[string, string]); !ok {
		t.Errorf("expected accessExpiryCalculator, got %T", builder.expiryCalculator)
	}
}

func TestCacheBuilder_TypedEncoder_Setter(t *testing.T) {
	service := NewService("")
	builder := NewCacheBuilder[string, string](service)
	encoder := newMockEncoder[string]()
	result := builder.TypedEncoder(encoder)
	if result != builder {
		t.Error("TypedEncoder should return the same builder instance")
	}
	if len(builder.encoders) != 1 {
		t.Errorf("expected 1 encoder, got %d", len(builder.encoders))
	}
}

func TestWriteExpiryCalculator(t *testing.T) {
	calc := &writeExpiryCalculator[string, string]{duration: 10 * time.Minute}
	entry := cache_dto.Entry[string, string]{}

	if got := calc.ExpireAfterCreate(entry); got != 10*time.Minute {
		t.Errorf("ExpireAfterCreate: expected 10m, got %v", got)
	}
	if got := calc.ExpireAfterUpdate(entry, "old"); got != 10*time.Minute {
		t.Errorf("ExpireAfterUpdate: expected 10m, got %v", got)
	}
	if got := calc.ExpireAfterRead(entry); got != -1 {
		t.Errorf("ExpireAfterRead: expected -1, got %v", got)
	}
}

func TestAccessExpiryCalculator(t *testing.T) {
	calc := &accessExpiryCalculator[string, string]{duration: 30 * time.Minute}
	entry := cache_dto.Entry[string, string]{}

	if got := calc.ExpireAfterCreate(entry); got != 30*time.Minute {
		t.Errorf("ExpireAfterCreate: expected 30m, got %v", got)
	}
	if got := calc.ExpireAfterUpdate(entry, "old"); got != 30*time.Minute {
		t.Errorf("ExpireAfterUpdate: expected 30m, got %v", got)
	}
	if got := calc.ExpireAfterRead(entry); got != 30*time.Minute {
		t.Errorf("ExpireAfterRead: expected 30m, got %v", got)
	}
}

func TestAdaptWeigher(t *testing.T) {
	service := NewService("")
	t.Run("nil weigher returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		if builder.adaptWeigher() != nil {
			t.Error("expected nil")
		}
	})
	t.Run("non-nil weigher returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		builder.weigher = func(k, v string) uint32 { return 1 }
		if builder.adaptWeigher() != nil {
			t.Error("expected nil (documented limitation)")
		}
	})
}

func TestAdaptExpiryCalculator(t *testing.T) {
	service := NewService("")
	t.Run("nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		if builder.adaptExpiryCalculator() != nil {
			t.Error("expected nil")
		}
	})
	t.Run("non-nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		builder.expiryCalculator = &mockExpiryCalculator[string, string]{}
		if builder.adaptExpiryCalculator() != nil {
			t.Error("expected nil (documented limitation)")
		}
	})
}

func TestAdaptOnDeletion(t *testing.T) {
	service := NewService("")
	t.Run("nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		if builder.adaptOnDeletion() != nil {
			t.Error("expected nil")
		}
	})
	t.Run("non-nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		builder.onDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
		if builder.adaptOnDeletion() != nil {
			t.Error("expected nil (documented limitation)")
		}
	})
}

func TestAdaptOnAtomicDeletion(t *testing.T) {
	service := NewService("")
	t.Run("nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		if builder.adaptOnAtomicDeletion() != nil {
			t.Error("expected nil")
		}
	})
	t.Run("non-nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		builder.onAtomicDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
		if builder.adaptOnAtomicDeletion() != nil {
			t.Error("expected nil (documented limitation)")
		}
	})
}

func TestAdaptRefreshCalculator(t *testing.T) {
	service := NewService("")
	t.Run("nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		if builder.adaptRefreshCalculator() != nil {
			t.Error("expected nil")
		}
	})
	t.Run("non-nil returns nil", func(t *testing.T) {
		builder := NewCacheBuilder[string, string](service)
		builder.refreshCalculator = &mockRefreshCalculator[string, string]{}
		if builder.adaptRefreshCalculator() != nil {
			t.Error("expected nil (documented limitation)")
		}
	})
}

func TestWarnAboutTransformerLimitations(t *testing.T) {
	tests := []struct {
		setup func(b *CacheBuilder[string, string])
		name  string
	}{
		{
			name: "searchSchema set",
			setup: func(b *CacheBuilder[string, string]) {
				b.searchSchema = &cache_dto.SearchSchema{}
			},
		},
		{
			name: "weigher set",
			setup: func(b *CacheBuilder[string, string]) {
				b.weigher = func(k, v string) uint32 { return 1 }
			},
		},
		{
			name: "custom expiryCalculator",
			setup: func(b *CacheBuilder[string, string]) {
				b.expiryCalculator = &mockExpiryCalculator[string, string]{}
			},
		},
		{
			name: "built-in writeExpiryCalculator skips warning",
			setup: func(b *CacheBuilder[string, string]) {
				b.expiryCalculator = &writeExpiryCalculator[string, string]{duration: time.Minute}
			},
		},
		{
			name: "built-in accessExpiryCalculator skips warning",
			setup: func(b *CacheBuilder[string, string]) {
				b.expiryCalculator = &accessExpiryCalculator[string, string]{duration: time.Minute}
			},
		},
		{
			name: "onDeletion set",
			setup: func(b *CacheBuilder[string, string]) {
				b.onDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
			},
		},
		{
			name: "onAtomicDeletion set",
			setup: func(b *CacheBuilder[string, string]) {
				b.onAtomicDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
			},
		},
		{
			name: "refreshCalculator set",
			setup: func(b *CacheBuilder[string, string]) {
				b.refreshCalculator = &mockRefreshCalculator[string, string]{}
			},
		},
		{
			name: "maximumWeight without weigher",
			setup: func(b *CacheBuilder[string, string]) {
				b.maximumWeight = 5000
			},
		},
		{
			name: "all warnings combined",
			setup: func(b *CacheBuilder[string, string]) {
				b.searchSchema = &cache_dto.SearchSchema{}
				b.weigher = func(k, v string) uint32 { return 1 }
				b.expiryCalculator = &mockExpiryCalculator[string, string]{}
				b.onDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
				b.onAtomicDeletion = func(e cache_dto.DeletionEvent[string, string]) {}
				b.refreshCalculator = &mockRefreshCalculator[string, string]{}
				b.maximumWeight = 5000
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewCacheBuilder[string, string](NewService(""))
			tc.setup(builder)

			builder.warnAboutTransformerLimitations(context.Background())
		})
	}
}

func TestCacheBuilder_BuildFromBlueprint_NotFound(t *testing.T) {
	service := NewService("mock")
	builder := NewCacheBuilder[string, string](service).
		FactoryBlueprint("nonexistent-blueprint-xyz-" + t.Name())

	cache, err := builder.Build(context.Background())
	if err == nil {
		if cache != nil {
			_ = cache.Close(context.Background())
		}
		t.Fatal("expected error for missing blueprint")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheBuilder_BuildFromBlueprint_Success(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	blueprintName := "test-blueprint-build-" + t.Name()
	RegisterProviderFactory(blueprintName, func(service Service, namespace string, options any) (any, error) {
		return &mockCache[string, string]{data: make(map[string]string)}, nil
	})

	service := NewService("mock")
	builder := NewCacheBuilder[string, string](service).
		FactoryBlueprint(blueprintName).
		Namespace("test-ns")

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	defer func() { _ = cache.Close(context.Background()) }()
}

func TestCacheBuilder_BuildFromBlueprint_FactoryError(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	blueprintName := "test-blueprint-err-" + t.Name()
	RegisterProviderFactory(blueprintName, func(service Service, namespace string, options any) (any, error) {
		return nil, errors.New("factory boom")
	})

	service := NewService("mock")
	builder := NewCacheBuilder[string, string](service).
		FactoryBlueprint(blueprintName)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected error from factory")
	}
	if !contains(err.Error(), "factory boom") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheBuilder_BuildFromBlueprint_TypeMismatch(t *testing.T) {
	isolateProviderFactoryRegistry(t)

	blueprintName := "test-blueprint-type-" + t.Name()
	RegisterProviderFactory(blueprintName, func(service Service, namespace string, options any) (any, error) {
		return "not-a-cache", nil
	})

	service := NewService("mock")
	builder := NewCacheBuilder[string, string](service).
		FactoryBlueprint(blueprintName)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !contains(err.Error(), "incorrect type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheBuilder_BuildWrappedCache_WithEncoder(t *testing.T) {
	service := NewService("mock")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	encoder := newMockEncoder[string]()
	builder := NewCacheBuilder[string, string](service).
		Encoder(encoder)

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	defer func() { _ = cache.Close(context.Background()) }()
}

func TestCacheBuilder_BuildWrappedCache_WithDefaultEncoder(t *testing.T) {
	service := NewService("mock")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	defaultEnc := newMockEncoder[any]()
	builder := NewCacheBuilder[string, string](service).
		DefaultEncoder(defaultEnc)

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	defer func() { _ = cache.Close(context.Background()) }()
}

func TestCacheBuilder_BuildWrappedCache_WithTransformer(t *testing.T) {
	transformerBlueprintsMutex.Lock()
	originalBlueprints := transformerBlueprints
	transformerBlueprints = make(map[string]TransformerBlueprintFactory)
	transformerBlueprintsMutex.Unlock()
	defer func() {
		transformerBlueprintsMutex.Lock()
		transformerBlueprints = originalBlueprints
		transformerBlueprintsMutex.Unlock()
	}()

	transformerName := "test-wrap-transformer-" + t.Name()
	RegisterTransformerBlueprint(transformerName, func(config any) (CacheTransformerPort, error) {
		return newMockTransformer(transformerName, 100), nil
	})

	service := NewService("mock")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		Transformer(transformerName)

	cache, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	defer func() { _ = cache.Close(context.Background()) }()
}

func TestCacheBuilder_BuildMultiLevel_NoConstructor(t *testing.T) {

	multiLevelAdapterMutex.Lock()
	original := multiLevelAdapterConstructor
	multiLevelAdapterConstructor = nil
	multiLevelAdapterMutex.Unlock()
	defer func() {
		multiLevelAdapterMutex.Lock()
		multiLevelAdapterConstructor = original
		multiLevelAdapterMutex.Unlock()
	}()

	service := NewService("")
	provider := newMockTestProvider("mock")
	if err := service.RegisterProvider(context.Background(), "mock", provider); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	builder := NewCacheBuilder[string, string](service).
		MultiLevel("mock", "mock")

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected error when multi-level constructor is not registered")
	}
	if !contains(err.Error(), "not registered") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetMultiLevelAdapterConstructor_NotRegistered(t *testing.T) {
	multiLevelAdapterMutex.Lock()
	original := multiLevelAdapterConstructor
	multiLevelAdapterConstructor = nil
	multiLevelAdapterMutex.Unlock()
	defer func() {
		multiLevelAdapterMutex.Lock()
		multiLevelAdapterConstructor = original
		multiLevelAdapterMutex.Unlock()
	}()

	constructor, ok := getMultiLevelAdapterConstructor()
	if ok {
		t.Error("expected false when no constructor registered")
	}
	if constructor != nil {
		t.Error("expected nil constructor")
	}
}

func TestRegisterMultiLevelAdapterConstructor_Success(t *testing.T) {
	multiLevelAdapterMutex.Lock()
	original := multiLevelAdapterConstructor
	multiLevelAdapterConstructor = nil
	multiLevelAdapterMutex.Unlock()
	defer func() {
		multiLevelAdapterMutex.Lock()
		multiLevelAdapterConstructor = original
		multiLevelAdapterMutex.Unlock()
	}()

	RegisterMultiLevelAdapterConstructor(func(ctx context.Context, name string, l1, l2, config any) (any, error) {
		return "multilevel", nil
	})

	constructor, ok := getMultiLevelAdapterConstructor()
	if !ok {
		t.Fatal("expected constructor to be registered")
	}
	result, err := constructor(context.Background(), "test", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "multilevel" {
		t.Errorf("expected 'multilevel', got %v", result)
	}
}

func TestRegisterMultiLevelAdapterConstructor_DuplicatePanics(t *testing.T) {
	multiLevelAdapterMutex.Lock()
	original := multiLevelAdapterConstructor
	multiLevelAdapterConstructor = nil
	multiLevelAdapterMutex.Unlock()
	defer func() {
		multiLevelAdapterMutex.Lock()
		multiLevelAdapterConstructor = original
		multiLevelAdapterMutex.Unlock()
	}()

	RegisterMultiLevelAdapterConstructor(func(ctx context.Context, name string, l1, l2, config any) (any, error) {
		return nil, nil
	})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
		message := fmt.Sprintf("%v", r)
		if !contains(message, "already registered") {
			t.Errorf("unexpected panic message: %s", message)
		}
	}()

	RegisterMultiLevelAdapterConstructor(func(ctx context.Context, name string, l1, l2, config any) (any, error) {
		return nil, nil
	})
}

func TestSetupEncoderRegistry_Empty(t *testing.T) {
	builder := NewCacheBuilder[string, string](NewService(""))
	registry, err := builder.setupEncoderRegistry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry != nil {
		t.Error("expected nil registry when no encoders are configured")
	}
}

func TestSetupEncoderRegistry_WithEncoders(t *testing.T) {
	builder := NewCacheBuilder[string, string](NewService(""))
	builder.encoders = append(builder.encoders, newMockEncoder[string]())
	registry, err := builder.setupEncoderRegistry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 encoder, got %d", registry.Count())
	}
}

func TestSetupEncoderRegistry_WithDefaultOnly(t *testing.T) {
	builder := NewCacheBuilder[string, string](NewService(""))
	builder.defaultEncoder = newMockEncoder[any]()
	registry, err := builder.setupEncoderRegistry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if !registry.HasDefault() {
		t.Error("expected registry to have default encoder")
	}
}

type mockStatsRecorder struct{}

func (*mockStatsRecorder) RecordHits(uint64)               {}
func (*mockStatsRecorder) RecordMisses(uint64)             {}
func (*mockStatsRecorder) RecordLoadSuccess(time.Duration) {}
func (*mockStatsRecorder) RecordLoadFailure(time.Duration) {}
func (*mockStatsRecorder) RecordEviction()                 {}

type mockClock struct{}

func (*mockClock) Now() time.Time { return time.Now() }
