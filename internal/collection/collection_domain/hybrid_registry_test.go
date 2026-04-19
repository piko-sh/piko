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

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/wdk/clock"
)

func mustCastToDefaultHybridRegistry(t *testing.T, reg HybridRegistryPort) *defaultHybridRegistry {
	t.Helper()
	hr, ok := reg.(*defaultHybridRegistry)
	if !ok {
		t.Fatal("expected *defaultHybridRegistry")
	}
	return hr
}

func TestRegisterHybridSnapshot(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("test blob data")
	etag := "md-abc123"

	RegisterHybridSnapshot(context.Background(), "markdown", "blog", blob, etag, config)

	if !HasHybridCollection("markdown", "blog") {
		t.Error("Expected hybrid collection to be registered")
	}

	if HasHybridCollection("markdown", "nonexistent") {
		t.Error("Expected nonexistent collection to not be registered")
	}
}

func TestGetHybridBlob(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("test blob data")
	etag := "md-abc123"

	RegisterHybridSnapshot(context.Background(), "markdown", "blog", blob, etag, config)

	retrievedBlob, needsRevalidation := GetHybridBlob(context.Background(), "markdown", "blog")

	if string(retrievedBlob) != string(blob) {
		t.Errorf("Expected blob %q, got %q", blob, retrievedBlob)
	}

	if needsRevalidation {
		t.Error("Expected needsRevalidation to be false immediately after registration")
	}

	nilBlob, needsRevalidation := GetHybridBlob(context.Background(), "markdown", "nonexistent")
	if nilBlob != nil {
		t.Error("Expected nil blob for nonexistent collection")
	}
	if needsRevalidation {
		t.Error("Expected needsRevalidation to be false for nonexistent collection")
	}
}

func TestGetHybridETag(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("test blob data")
	etag := "md-abc123"

	RegisterHybridSnapshot(context.Background(), "markdown", "blog", blob, etag, config)

	retrievedETag := GetHybridETag("markdown", "blog")
	if retrievedETag != etag {
		t.Errorf("Expected ETag %q, got %q", etag, retrievedETag)
	}

	emptyETag := GetHybridETag("markdown", "nonexistent")
	if emptyETag != "" {
		t.Errorf("Expected empty ETag for nonexistent collection, got %q", emptyETag)
	}
}

func TestListHybridCollections(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("test blob")

	RegisterHybridSnapshot(context.Background(), "markdown", "blog", blob, "etag1", config)
	RegisterHybridSnapshot(context.Background(), "markdown", "docs", blob, "etag2", config)
	RegisterHybridSnapshot(context.Background(), "cms", "products", blob, "etag3", config)

	keys := ListHybridCollections()

	if len(keys) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(keys))
	}

	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	expectedKeys := []string{"markdown:blog", "markdown:docs", "cms:products"}
	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("Expected key %q to be in list", expected)
		}
	}
}

func TestMakeHybridKey(t *testing.T) {
	testCases := []struct {
		name           string
		providerName   string
		collectionName string
		expectedKey    string
	}{
		{
			name:           "simple key",
			providerName:   "markdown",
			collectionName: "blog",
			expectedKey:    "markdown:blog",
		},
		{
			name:           "empty provider",
			providerName:   "",
			collectionName: "blog",
			expectedKey:    ":blog",
		},
		{
			name:           "empty collection",
			providerName:   "markdown",
			collectionName: "",
			expectedKey:    "markdown:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := makeHybridKey(tc.providerName, tc.collectionName)
			if key != tc.expectedKey {
				t.Errorf("Expected key %q, got %q", tc.expectedKey, key)
			}
		})
	}
}

func TestShouldRevalidate_TTLExpired(t *testing.T) {

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 100 * time.Millisecond,
		StaleIfError:    true,
	}

	entry := HybridCacheValue{
		LastRevalidated: baseTime.Add(-200 * time.Millisecond),
		Config:          config,
	}

	if !shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return true for expired TTL")
	}
}

func TestShouldRevalidate_TTLNotExpired(t *testing.T) {

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 10 * time.Second,
		StaleIfError:    true,
	}

	entry := HybridCacheValue{
		LastRevalidated: baseTime,
		Config:          config,
	}

	if shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return false for non-expired TTL")
	}
}

func TestShouldRevalidate_ZeroTTL(t *testing.T) {

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 0,
		StaleIfError:    true,
	}

	entry := HybridCacheValue{
		LastRevalidated: baseTime,
		Config:          config,
	}

	if !shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return true for zero TTL")
	}
}

func TestShouldRevalidate_ClockAdvance(t *testing.T) {

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 1 * time.Minute,
		StaleIfError:    true,
	}

	entry := HybridCacheValue{
		LastRevalidated: baseTime,
		Config:          config,
	}

	if shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return false initially")
	}

	mockClock.Advance(30 * time.Second)
	if shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return false at 30 seconds")
	}

	mockClock.Advance(31 * time.Second)
	if !shouldRevalidate(entry, mockClock) {
		t.Error("Expected shouldRevalidate to return true after TTL expires")
	}
}

func TestConcurrentRegistration(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			blob := []byte("blob")
			etag := "etag"
			RegisterHybridSnapshot(context.Background(), "provider", "collection", blob, etag, config)
		})
	}

	wg.Wait()

	if !HasHybridCollection("provider", "collection") {
		t.Error("Expected hybrid collection to be registered")
	}
}

func TestConcurrentGetHybridBlob(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("concurrent test blob")
	etag := "md-concurrent"

	RegisterHybridSnapshot(context.Background(), "markdown", "concurrent", blob, etag, config)

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			retrievedBlob, _ := GetHybridBlob(context.Background(), "markdown", "concurrent")
			if string(retrievedBlob) != string(blob) {
				t.Errorf("Concurrent access returned wrong blob")
			}
		})
	}

	wg.Wait()
}

func TestTriggerHybridRevalidation_NotRegistered(t *testing.T) {
	ResetHybridRegistry()

	ctx := context.Background()

	TriggerHybridRevalidation(ctx, "nonexistent", "collection")
}

func TestTriggerHybridRevalidation_ConcurrentCalls(t *testing.T) {
	ResetHybridRegistry()
	t.Cleanup(ResetHybridRegistry)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 0,
		StaleIfError:    true,
	}
	blob := []byte("test blob")
	etag := "md-test"

	RegisterHybridSnapshot(context.Background(), "markdown", "test", blob, etag, config)

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10

	for range numGoroutines {
		wg.Go(func() {
			TriggerHybridRevalidation(ctx, "markdown", "test")
		})
	}

	wg.Wait()
}

func TestNewDefaultHybridRegistry(t *testing.T) {
	registry := newDefaultHybridRegistry()
	if registry == nil {
		t.Fatal("newDefaultHybridRegistry() returned nil")
	}
}

func TestNewDefaultHybridRegistry_WithRuntimeRegistry(t *testing.T) {
	mockRegistry := &MockRuntimeProviderRegistry{}
	registry := newDefaultHybridRegistry(withHybridRuntimeRegistry(mockRegistry))

	if registry == nil {
		t.Fatal("newDefaultHybridRegistry() returned nil")
	}

	r := mustCastToDefaultHybridRegistry(t, registry)
	if r.runtimeRegistry != mockRegistry {
		t.Error("withHybridRuntimeRegistry did not inject the mock registry")
	}
}

func TestNewDefaultHybridRegistry_WithEncoder(t *testing.T) {
	encoder := &MockEncoder{
		EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}
	registry := newDefaultHybridRegistry(withHybridEncoder(encoder))

	if registry == nil {
		t.Fatal("newDefaultHybridRegistry() returned nil")
	}

	r := mustCastToDefaultHybridRegistry(t, registry)
	if r.encoder != encoder {
		t.Error("withHybridEncoder did not inject the mock encoder")
	}
}

func TestNewDefaultHybridRegistry_MultipleOptions(t *testing.T) {
	mockRegistry := &MockRuntimeProviderRegistry{}
	encoder := &MockEncoder{
		EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}

	registry := newDefaultHybridRegistry(
		withHybridRuntimeRegistry(mockRegistry),
		withHybridEncoder(encoder),
	)

	if registry == nil {
		t.Fatal("newDefaultHybridRegistry() returned nil")
	}

	r := mustCastToDefaultHybridRegistry(t, registry)
	if r.runtimeRegistry != mockRegistry {
		t.Error("runtimeRegistry not injected")
	}
	if r.encoder != encoder {
		t.Error("encoder not injected")
	}
}

func TestNewDefaultHybridRegistry_WithHybridClock(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	registry := newDefaultHybridRegistry(withHybridClock(mockClock))

	if registry == nil {
		t.Fatal("newDefaultHybridRegistry() returned nil")
	}

	r := mustCastToDefaultHybridRegistry(t, registry)
	if r.clock != mockClock {
		t.Error("withHybridClock did not inject the mock clock")
	}

	if r.clock.Now() != baseTime {
		t.Errorf("Expected time %v, got %v", baseTime, r.clock.Now())
	}
}

func Test_setHybridClock(t *testing.T) {

	originalClock := defaultHybridClock
	defer func() {
		defaultHybridClock = originalClock
	}()

	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	setHybridClock(mockClock)

	if defaultHybridClock != mockClock {
		t.Error("setHybridClock did not set the package-level clock")
	}

	setHybridClock(clock.RealClock())
	if defaultHybridClock == mockClock {
		t.Error("setHybridClock did not reset to real clock")
	}
}

func Test_setHybridClock_AffectsRegistration(t *testing.T) {
	ResetHybridRegistry()

	originalClock := defaultHybridClock
	defer func() {
		defaultHybridClock = originalClock
	}()

	baseTime := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)
	setHybridClock(mockClock)

	config := collection_dto.HybridConfig{
		RevalidationTTL: 1 * time.Minute,
		StaleIfError:    true,
	}

	RegisterHybridSnapshot(context.Background(), "test", "collection", []byte("data"), "etag", config)

	_, needsRevalidation := GetHybridBlob(context.Background(), "test", "collection")
	if needsRevalidation {
		t.Error("Expected no revalidation needed immediately after registration")
	}

	mockClock.Advance(2 * time.Minute)

	_, needsRevalidation = GetHybridBlob(context.Background(), "test", "collection")
	if !needsRevalidation {
		t.Error("Expected revalidation needed after TTL expired")
	}
}

func TestDefaultHybridRegistry_Register(t *testing.T) {
	ResetHybridRegistry()

	registry := newDefaultHybridRegistry()
	config := collection_dto.DefaultHybridConfig()

	registry.Register(context.Background(), "provider", "collection", []byte("data"), "etag", config)

	if !HasHybridCollection("provider", "collection") {
		t.Error("Expected collection to be registered")
	}
}

func TestDefaultHybridRegistry_GetBlob(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	blob := []byte("interface test blob")
	RegisterHybridSnapshot(context.Background(), "provider", "collection", blob, "etag", config)

	registry := newDefaultHybridRegistry()
	retrievedBlob, _ := registry.GetBlob(context.Background(), "provider", "collection")

	if string(retrievedBlob) != string(blob) {
		t.Errorf("Expected blob %q, got %q", blob, retrievedBlob)
	}
}

func TestDefaultHybridRegistry_GetETag(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	RegisterHybridSnapshot(context.Background(), "provider", "collection", []byte("data"), "test-etag", config)

	registry := newDefaultHybridRegistry()
	etag := registry.GetETag("provider", "collection")

	if etag != "test-etag" {
		t.Errorf("Expected etag 'test-etag', got %q", etag)
	}
}

func TestDefaultHybridRegistry_Has(t *testing.T) {
	ResetHybridRegistry()

	registry := newDefaultHybridRegistry()

	if registry.Has("provider", "collection") {
		t.Error("Expected Has to return false for unregistered collection")
	}

	config := collection_dto.DefaultHybridConfig()
	RegisterHybridSnapshot(context.Background(), "provider", "collection", []byte("data"), "etag", config)

	if !registry.Has("provider", "collection") {
		t.Error("Expected Has to return true for registered collection")
	}
}

func TestDefaultHybridRegistry_List(t *testing.T) {
	ResetHybridRegistry()

	config := collection_dto.DefaultHybridConfig()
	RegisterHybridSnapshot(context.Background(), "provider1", "coll1", []byte("data"), "etag1", config)
	RegisterHybridSnapshot(context.Background(), "provider2", "coll2", []byte("data"), "etag2", config)

	registry := newDefaultHybridRegistry()
	keys := registry.List()

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestHybridError_Error(t *testing.T) {
	err := &hybridError{message: "test error message"}

	if err.Error() != "test error message" {
		t.Errorf("Expected 'test error message', got %q", err.Error())
	}
}

func TestErrProviderNotFound(t *testing.T) {
	err := errProviderNotFound("test-provider")

	expectedMessage := "hybrid provider not found: test-provider"
	if err.Error() != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, err.Error())
	}
}

func TestErrProviderNotHybridCapable(t *testing.T) {
	err := errProviderNotHybridCapable("test-provider")

	expectedMessage := "provider does not support hybrid mode: test-provider"
	if err.Error() != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, err.Error())
	}
}

func TestGetHybridCapableProviderFromRegistry_ProviderNotFound(t *testing.T) {
	ResetRuntimeProviderRegistry()

	mockRegistry := NewDefaultRuntimeProviderRegistry()

	_, err := getHybridCapableProviderFromRegistry("nonexistent", mockRegistry)
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestGetHybridCapableProviderFromRegistry_NotHybridCapable(t *testing.T) {
	ResetRuntimeProviderRegistry()

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "non-hybrid" },
	}
	_ = RegisterProvider(provider)

	mockRegistry := NewDefaultRuntimeProviderRegistry()

	_, err := getHybridCapableProviderFromRegistry("non-hybrid", mockRegistry)
	if err == nil {
		t.Error("Expected error for non-hybrid-capable provider")
	}
}

func TestApplyRevalidationResultWithEncoder_Error(t *testing.T) {

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "test",
		Config:         collection_dto.HybridConfig{StaleIfError: true},
	}

	result := &collection_dto.HybridRevalidationResult{
		Error:         errProviderNotFound("test"),
		RevalidatedAt: fixedTime,
	}

	newVal, err := applyRevalidationResult(context.Background(), entry, result, nil)
	if err != nil {
		t.Fatalf("applyRevalidationResult() returned error: %v", err)
	}

	if newVal.LastRevalidated != result.RevalidatedAt {
		t.Error("Expected LastRevalidated to be updated")
	}
}

func TestApplyRevalidationResultWithEncoder_NoChange(t *testing.T) {

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "test",
		CurrentBlob:    []byte("original"),
		CurrentETag:    "original-etag",
	}

	result := &collection_dto.HybridRevalidationResult{
		ETagChanged:   false,
		RevalidatedAt: fixedTime,
	}

	newVal, err := applyRevalidationResult(context.Background(), entry, result, nil)
	if err != nil {
		t.Fatalf("applyRevalidationResult() returned error: %v", err)
	}

	if string(newVal.CurrentBlob) != "original" {
		t.Error("Expected blob to remain unchanged")
	}
	if newVal.CurrentETag != "original-etag" {
		t.Error("Expected etag to remain unchanged")
	}
}

func TestApplyRevalidationResultWithEncoder_EmptyItems(t *testing.T) {

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "test",
		CurrentBlob:    []byte("original"),
	}

	result := &collection_dto.HybridRevalidationResult{
		ETagChanged:   true,
		NewItems:      []collection_dto.ContentItem{},
		RevalidatedAt: fixedTime,
	}

	newVal, err := applyRevalidationResult(context.Background(), entry, result, nil)
	if err != nil {
		t.Fatalf("applyRevalidationResult() returned error: %v", err)
	}

	if string(newVal.CurrentBlob) != "original" {
		t.Error("Expected blob to remain unchanged for empty items")
	}
}

func TestApplyRevalidationResultWithEncoder_Success(t *testing.T) {

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "test",
		CurrentBlob:    []byte("original"),
		CurrentETag:    "original-etag",
	}

	encoder := &MockEncoder{
		EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
			return []byte("new-encoded-data"), nil
		},
	}

	result := &collection_dto.HybridRevalidationResult{
		ETagChanged: true,
		NewETag:     "new-etag",
		NewItems: []collection_dto.ContentItem{
			{ID: "item-1"},
		},
		RevalidatedAt: fixedTime,
	}

	newVal, err := applyRevalidationResult(context.Background(), entry, result, encoder)
	if err != nil {
		t.Fatalf("applyRevalidationResult() returned error: %v", err)
	}

	if string(newVal.CurrentBlob) != "new-encoded-data" {
		t.Errorf("Expected blob 'new-encoded-data', got %q", newVal.CurrentBlob)
	}
	if newVal.CurrentETag != "new-etag" {
		t.Errorf("Expected etag 'new-etag', got %q", newVal.CurrentETag)
	}
}

func TestEncodeItemsToBlob(t *testing.T) {
	encoder := &MockEncoder{
		EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
			return []byte("encoded"), nil
		},
	}

	items := []collection_dto.ContentItem{{ID: "test"}}

	blob, err := encodeItemsToBlob(items, encoder)
	if err != nil {
		t.Fatalf("encodeItemsToBlob() failed: %v", err)
	}

	if string(blob) != "encoded" {
		t.Errorf("Expected 'encoded', got %q", blob)
	}
}

type mockHybridCapableProvider struct {
	validateETagFunc func(ctx context.Context, collectionName, expectedETag string) (string, bool, error)
	fetchStaticFunc  func(ctx context.Context, collectionName string) ([]collection_dto.ContentItem, error)
}

func (m *mockHybridCapableProvider) ValidateETag(ctx context.Context, collectionName, expectedETag string) (string, bool, error) {
	if m.validateETagFunc != nil {
		return m.validateETagFunc(ctx, collectionName, expectedETag)
	}
	return expectedETag, false, nil
}

func (m *mockHybridCapableProvider) FetchStaticContent(ctx context.Context, collectionName string) ([]collection_dto.ContentItem, error) {
	if m.fetchStaticFunc != nil {
		return m.fetchStaticFunc(ctx, collectionName)
	}
	return nil, nil
}

func TestValidateAndFetchIfChanged_ETagUnchanged(t *testing.T) {
	provider := &mockHybridCapableProvider{
		validateETagFunc: func(_ context.Context, _, expectedETag string) (string, bool, error) {
			return expectedETag, false, nil
		},
	}

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "collection",
		CurrentETag:    "etag-123",
	}

	result := &collection_dto.HybridRevalidationResult{}

	finalResult := validateAndFetchIfChanged(context.Background(), provider, entry, result)

	if finalResult.ETagChanged {
		t.Error("Expected ETagChanged to be false")
	}

	if finalResult.NewETag != "etag-123" {
		t.Errorf("Expected NewETag 'etag-123', got %q", finalResult.NewETag)
	}

	if finalResult.NewItems != nil {
		t.Error("Expected NewItems to be nil when ETag unchanged")
	}
}

func TestValidateAndFetchIfChanged_ETagValidationError(t *testing.T) {
	expectedErr := errProviderNotFound("test")
	provider := &mockHybridCapableProvider{
		validateETagFunc: func(_ context.Context, _, _ string) (string, bool, error) {
			return "", false, expectedErr
		},
	}

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "collection",
		CurrentETag:    "etag-123",
	}

	result := &collection_dto.HybridRevalidationResult{}

	finalResult := validateAndFetchIfChanged(context.Background(), provider, entry, result)

	if finalResult.Error == nil {
		t.Error("Expected error from validation failure")
	}
}

func TestValidateAndFetchIfChanged_ETagChanged(t *testing.T) {
	provider := &mockHybridCapableProvider{
		validateETagFunc: func(_ context.Context, _, _ string) (string, bool, error) {
			return "new-etag", true, nil
		},
		fetchStaticFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
			return []collection_dto.ContentItem{
				{ID: "item-1"},
				{ID: "item-2"},
			}, nil
		},
	}

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "collection",
		CurrentETag:    "old-etag",
	}

	result := &collection_dto.HybridRevalidationResult{}

	finalResult := validateAndFetchIfChanged(context.Background(), provider, entry, result)

	if !finalResult.ETagChanged {
		t.Error("Expected ETagChanged to be true")
	}

	if finalResult.NewETag != "new-etag" {
		t.Errorf("Expected NewETag 'new-etag', got %q", finalResult.NewETag)
	}

	if len(finalResult.NewItems) != 2 {
		t.Errorf("Expected 2 new items, got %d", len(finalResult.NewItems))
	}
}

func TestFetchFreshContent_Success(t *testing.T) {
	expectedItems := []collection_dto.ContentItem{
		{ID: "item-1"},
		{ID: "item-2"},
		{ID: "item-3"},
	}

	provider := &mockHybridCapableProvider{
		fetchStaticFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
			return expectedItems, nil
		},
	}

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "collection",
		CurrentETag:    "old-etag",
	}

	result := &collection_dto.HybridRevalidationResult{}

	finalResult := fetchFreshContent(context.Background(), provider, entry, "new-etag", result)

	if finalResult.Error != nil {
		t.Fatalf("fetchFreshContent() returned error: %v", finalResult.Error)
	}

	if len(finalResult.NewItems) != 3 {
		t.Errorf("Expected 3 items, got %d", len(finalResult.NewItems))
	}
}

func TestFetchFreshContent_Error(t *testing.T) {
	expectedErr := errProviderNotFound("test")
	provider := &mockHybridCapableProvider{
		fetchStaticFunc: func(_ context.Context, _ string) ([]collection_dto.ContentItem, error) {
			return nil, expectedErr
		},
	}

	entry := HybridCacheValue{
		ProviderName:   "test",
		CollectionName: "collection",
		CurrentETag:    "old-etag",
	}

	result := &collection_dto.HybridRevalidationResult{}

	finalResult := fetchFreshContent(context.Background(), provider, entry, "new-etag", result)

	if finalResult.Error == nil {
		t.Error("Expected error from fetch failure")
	}

	if finalResult.NewItems != nil {
		t.Error("Expected NewItems to be nil on error")
	}
}

func TestRevalidateHybridCollectionWithDeps_ProviderNotFound(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	entry := HybridCacheValue{
		ProviderName:   "nonexistent",
		CollectionName: "collection",
		CurrentETag:    "etag",
	}

	mockRegistry := &MockRuntimeProviderRegistry{
		GetFunc: func(name string) (RuntimeProvider, error) {
			return nil, errProviderNotFound(name)
		},
	}

	result := revalidateCollection(
		context.Background(),
		entry,
		mockRegistry,
		mockClock,
	)

	if result.Error == nil {
		t.Error("Expected error for nonexistent provider")
	}

	if result.RevalidatedAt != baseTime {
		t.Errorf("Expected RevalidatedAt %v, got %v", baseTime, result.RevalidatedAt)
	}
}

func TestRevalidateHybridCollectionWithDeps_ProviderNotHybridCapable(t *testing.T) {
	ResetRuntimeProviderRegistry()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	provider := &MockRuntimeProvider{
		NameFunc: func() string { return "non-hybrid" },
	}
	_ = RegisterProvider(provider)

	entry := HybridCacheValue{
		ProviderName:   "non-hybrid",
		CollectionName: "collection",
		CurrentETag:    "etag",
	}

	mockRegistry := NewDefaultRuntimeProviderRegistry()

	result := revalidateCollection(
		context.Background(),
		entry,
		mockRegistry,
		mockClock,
	)

	if result.Error == nil {
		t.Error("Expected error for non-hybrid-capable provider")
	}
}

func TestTryResetHybridRegistry_Success(t *testing.T) {
	if err := TryResetHybridRegistry(); err != nil {
		t.Fatalf("TryResetHybridRegistry returned unexpected error: %v", err)
	}

	config := collection_dto.DefaultHybridConfig()
	RegisterHybridSnapshot(context.Background(), "provider", "collection", []byte("data"), "etag", config)

	if !HasHybridCollection("provider", "collection") {
		t.Fatal("expected collection to be registered before reset")
	}

	if err := TryResetHybridRegistry(); err != nil {
		t.Fatalf("TryResetHybridRegistry returned unexpected error: %v", err)
	}

	if HasHybridCollection("provider", "collection") {
		t.Error("expected collection to be cleared after reset")
	}
}

func TestResetHybridRegistry_NoPanicOnRepeatedReset(t *testing.T) {
	for range 3 {
		ResetHybridRegistry()
	}
}
