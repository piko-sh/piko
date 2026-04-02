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
	"iter"
	"maps"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

type testCacheValue struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type mockByteCache struct {
	data             map[string][]byte
	expiresAfter     map[string]time.Duration
	refreshableAfter map[string]time.Duration
	invalidatedTags  []string
	maximum          uint64
	weightedSize     uint64
	mu               sync.Mutex
	closed           bool
}

func newMockByteCache() *mockByteCache {
	return &mockByteCache{
		data:             make(map[string][]byte),
		expiresAfter:     make(map[string]time.Duration),
		refreshableAfter: make(map[string]time.Duration),
	}
}

func (m *mockByteCache) GetIfPresent(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *mockByteCache) Get(ctx context.Context, key string, loader cache_dto.Loader[string, []byte]) ([]byte, error) {
	m.mu.Lock()
	v, ok := m.data[key]
	m.mu.Unlock()
	if ok {
		return v, nil
	}
	if loader == nil {
		return nil, errors.New("key not found and no loader provided")
	}
	loaded, err := loader.Load(ctx, key)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.data[key] = loaded
	m.mu.Unlock()
	return loaded, nil
}

func (m *mockByteCache) Set(_ context.Context, key string, value []byte, _ ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockByteCache) SetWithTTL(_ context.Context, key string, value []byte, _ time.Duration, _ ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockByteCache) BulkSet(_ context.Context, items map[string][]byte, _ ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	maps.Copy(m.data, items)
	return nil
}

func (m *mockByteCache) Invalidate(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockByteCache) InvalidateByTags(_ context.Context, tags ...string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidatedTags = append(m.invalidatedTags, tags...)
	return 0, nil
}

func (m *mockByteCache) InvalidateAll(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	return nil
}

func (m *mockByteCache) Compute(_ context.Context, key string, computeFunction func(oldValue []byte, found bool) (newValue []byte, action cache_dto.ComputeAction)) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, found := m.data[key]
	newVal, action := computeFunction(old, found)
	switch action {
	case cache_dto.ComputeActionSet:
		m.data[key] = newVal
		return newVal, true, nil
	case cache_dto.ComputeActionDelete:
		delete(m.data, key)
		return nil, false, nil
	default:
		if found {
			return old, true, nil
		}
		return nil, false, nil
	}
}

func (m *mockByteCache) ComputeIfAbsent(_ context.Context, key string, computeFunction func() []byte) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.data[key]; ok {
		return existing, false, nil
	}
	value := computeFunction()
	m.data[key] = value
	return value, true, nil
}

func (m *mockByteCache) ComputeIfPresent(_ context.Context, key string, computeFunction func(oldValue []byte) (newValue []byte, action cache_dto.ComputeAction)) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, found := m.data[key]
	if !found {
		return nil, false, nil
	}
	newVal, action := computeFunction(old)
	switch action {
	case cache_dto.ComputeActionSet:
		m.data[key] = newVal
		return newVal, true, nil
	case cache_dto.ComputeActionDelete:
		delete(m.data, key)
		return nil, false, nil
	default:
		return old, true, nil
	}
}

func (m *mockByteCache) ComputeWithTTL(_ context.Context, key string, computeFunction func(oldValue []byte, found bool) cache_dto.ComputeResult[[]byte]) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, found := m.data[key]
	result := computeFunction(old, found)
	switch result.Action {
	case cache_dto.ComputeActionSet:
		m.data[key] = result.Value
		return result.Value, true, nil
	case cache_dto.ComputeActionDelete:
		delete(m.data, key)
		return nil, false, nil
	default:
		if found {
			return old, true, nil
		}
		return nil, false, nil
	}
}

func (m *mockByteCache) BulkGet(ctx context.Context, keys []string, bulkLoader cache_dto.BulkLoader[string, []byte]) (map[string][]byte, error) {
	m.mu.Lock()
	result := make(map[string][]byte)
	var missing []string
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			result[k] = v
		} else {
			missing = append(missing, k)
		}
	}
	m.mu.Unlock()

	if len(missing) > 0 && bulkLoader != nil {
		loaded, err := bulkLoader.BulkLoad(ctx, missing)
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		for k, v := range loaded {
			m.data[k] = v
			result[k] = v
		}
		m.mu.Unlock()
	}
	return result, nil
}

func (m *mockByteCache) BulkRefresh(ctx context.Context, keys []string, bulkLoader cache_dto.BulkLoader[string, []byte]) {
	if bulkLoader == nil {
		return
	}
	loaded, err := bulkLoader.BulkLoad(ctx, keys)
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	maps.Copy(m.data, loaded)
}

func (m *mockByteCache) Refresh(ctx context.Context, key string, loader cache_dto.Loader[string, []byte]) <-chan cache_dto.LoadResult[[]byte] {
	resultChannel := make(chan cache_dto.LoadResult[[]byte], 1)
	go func() {
		defer close(resultChannel)
		value, err := loader.Load(ctx, key)
		if err != nil {
			resultChannel <- cache_dto.LoadResult[[]byte]{Err: err}
			return
		}
		m.mu.Lock()
		m.data[key] = value
		m.mu.Unlock()
		resultChannel <- cache_dto.LoadResult[[]byte]{Value: value}
	}()
	return resultChannel
}

func (m *mockByteCache) All() iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()
		for k, v := range m.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (m *mockByteCache) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()
		for k := range m.data {
			if !yield(k) {
				return
			}
		}
	}
}

func (m *mockByteCache) Values() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, v := range m.data {
			if !yield(v) {
				return
			}
		}
	}
}

func (m *mockByteCache) GetEntry(_ context.Context, key string) (cache_dto.Entry[string, []byte], bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return cache_dto.Entry[string, []byte]{}, false, nil
	}
	return cache_dto.Entry[string, []byte]{
		Key:   key,
		Value: v,
	}, true, nil
}

func (m *mockByteCache) ProbeEntry(ctx context.Context, key string) (cache_dto.Entry[string, []byte], bool, error) {
	return m.GetEntry(ctx, key)
}

func (m *mockByteCache) EstimatedSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.data)
}

func (m *mockByteCache) Stats() cache_dto.Stats { return cache_dto.Stats{} }
func (m *mockByteCache) GetMaximum() uint64     { return m.maximum }
func (m *mockByteCache) SetMaximum(size uint64) { m.maximum = size }
func (m *mockByteCache) WeightedSize() uint64   { return m.weightedSize }
func (m *mockByteCache) SetExpiresAfter(_ context.Context, key string, d time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expiresAfter[key] = d
	return nil
}
func (m *mockByteCache) SetRefreshableAfter(_ context.Context, key string, d time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshableAfter[key] = d
	return nil
}
func (m *mockByteCache) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
func (m *mockByteCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, []byte], error) {
	return cache_dto.SearchResult[string, []byte]{}, ErrSearchNotSupported
}
func (m *mockByteCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, []byte], error) {
	return cache_dto.SearchResult[string, []byte]{}, ErrSearchNotSupported
}
func (m *mockByteCache) SupportsSearch() bool               { return false }
func (m *mockByteCache) GetSchema() *cache_dto.SearchSchema { return nil }

var _ Cache[string, []byte] = (*mockByteCache)(nil)

type wrapperTestEncoder struct {
	marshalErr      error
	unmarshalErr    error
	marshalCalled   int
	unmarshalCalled int
}

func (e *wrapperTestEncoder) Marshal(v testCacheValue) ([]byte, error) {
	e.marshalCalled++
	if e.marshalErr != nil {
		return nil, e.marshalErr
	}
	return CacheAPI.Marshal(v)
}

func (e *wrapperTestEncoder) Unmarshal(data []byte, target *testCacheValue) error {
	e.unmarshalCalled++
	if e.unmarshalErr != nil {
		return e.unmarshalErr
	}
	return CacheAPI.Unmarshal(data, target)
}

func newTestWrapper(provider *mockByteCache) *transformerWrapper[string, testCacheValue] {
	return newTransformerWrapper[string, testCacheValue](
		provider,
		NewTransformerRegistry(),
		nil,
	)
}

func newTestWrapperWithTransformer(provider *mockByteCache) (*transformerWrapper[string, testCacheValue], *mockTransformer) {
	registry := NewTransformerRegistry()
	transformer := newMockTransformer("mock", 100)
	if err := registry.Register(context.Background(), transformer); err != nil {
		panic(err)
	}
	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{"mock"},
		TransformerOptions:  make(map[string]any),
	}
	wrapper := newTransformerWrapper[string, testCacheValue](
		provider,
		registry,
		config,
	)
	return wrapper, transformer
}

func newTestWrapperWithEncoder(provider *mockByteCache, enc *wrapperTestEncoder) *transformerWrapper[string, testCacheValue] {
	return newTransformerWrapper[string, testCacheValue](
		provider,
		NewTransformerRegistry(),
		nil,
		withWrapperEncoder[string, testCacheValue](enc),
	)
}

func TestEncodeValue_DefaultJSON(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	data, err := w.encodeValue(testCacheValue{Name: "alice", Count: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty encoded data")
	}
}

func TestEncodeValue_CustomEncoder(t *testing.T) {
	enc := &wrapperTestEncoder{}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	_, err := w.encodeValue(testCacheValue{Name: "bob", Count: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc.marshalCalled != 1 {
		t.Errorf("expected marshal called once, got %d", enc.marshalCalled)
	}
}

func TestEncodeValue_EncoderError(t *testing.T) {
	enc := &wrapperTestEncoder{marshalErr: errors.New("encode boom")}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	_, err := w.encodeValue(testCacheValue{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "failed to encode cache value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeValue_DefaultJSON(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	original := testCacheValue{Name: "charlie", Count: 3}
	data, _ := w.encodeValue(original)
	decoded, err := w.decodeValue(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Count != original.Count {
		t.Errorf("decoded %+v != original %+v", decoded, original)
	}
}

func TestDecodeValue_CustomEncoder(t *testing.T) {
	enc := &wrapperTestEncoder{}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	data, _ := CacheAPI.Marshal(testCacheValue{Name: "test"})
	_, err := w.decodeValue(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc.unmarshalCalled != 1 {
		t.Errorf("expected unmarshal called once, got %d", enc.unmarshalCalled)
	}
}

func TestDecodeValue_EncoderError(t *testing.T) {
	enc := &wrapperTestEncoder{unmarshalErr: errors.New("decode boom")}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	_, err := w.decodeValue([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "failed to decode cache value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeValue_JSONFallbackError(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, err := w.decodeValue([]byte(`not-valid-json`))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestTransformAndWrap_NilConfig(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	input := []byte("hello")
	output, err := w.transformAndWrap(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("expected passthrough, got %q", output)
	}
}

func TestTransformAndWrap_EmptyTransformers(t *testing.T) {
	provider := newMockByteCache()
	config := &cache_dto.TransformConfig{
		EnabledTransformers: []string{},
		TransformerOptions:  make(map[string]any),
	}
	w := newTransformerWrapper[string, testCacheValue](
		provider, NewTransformerRegistry(), config,
	)
	input := []byte("hello")
	output, err := w.transformAndWrap(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("expected passthrough, got %q", output)
	}
}

func TestTransformAndWrap_WithTransformers(t *testing.T) {
	provider := newMockByteCache()
	w, _ := newTestWrapperWithTransformer(provider)
	input := []byte("hello")
	output, err := w.transformAndWrap(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) == string(input) {
		t.Error("expected transformed output to differ from input")
	}
	if !IsTransformedValue(output) {
		t.Error("expected output to be a TransformedValue")
	}
}

func TestTransformAndWrap_TransformError(t *testing.T) {
	provider := newMockByteCache()
	w, transformer := newTestWrapperWithTransformer(provider)
	transformer.failMode = "transform"
	_, err := w.transformAndWrap(context.Background(), []byte("hello"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "transformation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnwrapAndReverse_PlainData(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	plain := []byte(`{"name":"alice"}`)
	output, err := w.unwrapAndReverse(context.Background(), plain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != string(plain) {
		t.Error("expected plain data to pass through")
	}
}

func TestUnwrapAndReverse_TransformedData(t *testing.T) {
	provider := newMockByteCache()
	w, _ := newTestWrapperWithTransformer(provider)
	input := []byte("hello")
	wrapped, err := w.transformAndWrap(context.Background(), input)
	if err != nil {
		t.Fatalf("wrap error: %v", err)
	}
	reversed, err := w.unwrapAndReverse(context.Background(), wrapped)
	if err != nil {
		t.Fatalf("unwrap error: %v", err)
	}
	if string(reversed) != string(input) {
		t.Errorf("round trip failed: got %q, want %q", reversed, input)
	}
}

func TestTransformRoundTrip(t *testing.T) {
	provider := newMockByteCache()
	w, _ := newTestWrapperWithTransformer(provider)
	original := testCacheValue{Name: "roundtrip", Count: 42}
	encoded, err := w.encodeValue(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	wrapped, err := w.transformAndWrap(context.Background(), encoded)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	unwrapped, err := w.unwrapAndReverse(context.Background(), wrapped)
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	decoded, err := w.decodeValue(unwrapped)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Name != original.Name || decoded.Count != original.Count {
		t.Errorf("round trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestGetIfPresent_Found(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	original := testCacheValue{Name: "found", Count: 1}
	encoded, _ := CacheAPI.Marshal(original)
	_ = provider.Set(context.Background(), "key1", encoded)

	value, ok, _ := w.GetIfPresent(context.Background(), "key1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if value.Name != "found" || value.Count != 1 {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestGetIfPresent_NotFound(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.GetIfPresent(context.Background(), "missing")
	if ok {
		t.Error("expected key not found")
	}
}

func TestGetIfPresent_DecodeError(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "bad", []byte("not-json"))
	_, ok, _ := w.GetIfPresent(context.Background(), "bad")
	if ok {
		t.Error("expected false on decode error")
	}
}

func TestGet_WithLoader(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	loader := cache_dto.LoaderFunc[string, testCacheValue](func(_ context.Context, key string) (testCacheValue, error) {
		return testCacheValue{Name: "loaded-" + key, Count: 10}, nil
	})

	value, err := w.Get(context.Background(), "k1", loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value.Name != "loaded-k1" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestGet_CacheHit(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)

	original := testCacheValue{Name: "cached", Count: 5}
	encoded, _ := CacheAPI.Marshal(original)
	_ = provider.Set(context.Background(), "k1", encoded)

	loaderCalled := false
	loader := cache_dto.LoaderFunc[string, testCacheValue](func(_ context.Context, _ string) (testCacheValue, error) {
		loaderCalled = true
		return testCacheValue{}, nil
	})

	value, err := w.Get(context.Background(), "k1", loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaderCalled {
		t.Error("loader should not be called on cache hit")
	}
	if value.Name != "cached" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestGet_LoaderError(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	loader := cache_dto.LoaderFunc[string, testCacheValue](func(_ context.Context, _ string) (testCacheValue, error) {
		return testCacheValue{}, errors.New("loader failed")
	})

	_, err := w.Get(context.Background(), "k1", loader)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "loader failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSet_StoresTransformedValue(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = w.Set(context.Background(), "key1", testCacheValue{Name: "stored", Count: 7})

	value, ok, _ := w.GetIfPresent(context.Background(), "key1")
	if !ok {
		t.Fatal("expected key to be found after Set")
	}
	if value.Name != "stored" || value.Count != 7 {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestSet_WithTransformers(t *testing.T) {
	provider := newMockByteCache()
	w, _ := newTestWrapperWithTransformer(provider)
	_ = w.Set(context.Background(), "key1", testCacheValue{Name: "transformed", Count: 8})

	raw, ok, _ := provider.GetIfPresent(context.Background(), "key1")
	if !ok {
		t.Fatal("expected key in provider")
	}
	if !IsTransformedValue(raw) {
		t.Error("expected stored value to be a TransformedValue")
	}

	value, ok, _ := w.GetIfPresent(context.Background(), "key1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if value.Name != "transformed" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestSetWithTTL_Success(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	err := w.SetWithTTL(context.Background(), "key1", testCacheValue{Name: "ttl"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	value, ok, _ := w.GetIfPresent(context.Background(), "key1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if value.Name != "ttl" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestSetWithTTL_EncodeError(t *testing.T) {
	enc := &wrapperTestEncoder{marshalErr: errors.New("encode boom")}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	err := w.SetWithTTL(context.Background(), "key1", testCacheValue{}, time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "encoding failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBulkSet_MultipleItems(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	items := map[string]testCacheValue{
		"a": {Name: "alpha", Count: 1},
		"b": {Name: "beta", Count: 2},
	}
	err := w.BulkSet(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.EstimatedSize() != 2 {
		t.Errorf("expected 2 items, got %d", provider.EstimatedSize())
	}
}

func TestBulkSet_EmptyMap(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	err := w.BulkSet(context.Background(), map[string]testCacheValue{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidate_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "key1", []byte("data"))
	_ = w.Invalidate(context.Background(), "key1")
	if _, ok, _ := provider.GetIfPresent(context.Background(), "key1"); ok {
		t.Error("expected key to be invalidated")
	}
}

func TestInvalidateByTags_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_, _ = w.InvalidateByTags(context.Background(), "tag1", "tag2")
	if len(provider.invalidatedTags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(provider.invalidatedTags))
	}
}

func TestInvalidateAll_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "a", []byte("1"))
	_ = provider.Set(context.Background(), "b", []byte("2"))
	_ = w.InvalidateAll(context.Background())
	if provider.EstimatedSize() != 0 {
		t.Errorf("expected empty cache, got %d items", provider.EstimatedSize())
	}
}

func TestCompute_SetAction(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	value, ok, _ := w.Compute(context.Background(), "key1", func(_ testCacheValue, found bool) (testCacheValue, cache_dto.ComputeAction) {
		if found {
			t.Error("expected found=false for new key")
		}
		return testCacheValue{Name: "computed", Count: 99}, cache_dto.ComputeActionSet
	})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value.Name != "computed" || value.Count != 99 {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestCompute_NoopAction(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.Compute(context.Background(), "key1", func(_ testCacheValue, _ bool) (testCacheValue, cache_dto.ComputeAction) {
		return testCacheValue{}, cache_dto.ComputeActionNoop
	})
	if ok {
		t.Error("expected ok=false for noop on absent key")
	}
}

func TestCompute_DeleteAction(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)

	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "existing"})
	_ = provider.Set(context.Background(), "key1", encoded)

	_, ok, _ := w.Compute(context.Background(), "key1", func(_ testCacheValue, _ bool) (testCacheValue, cache_dto.ComputeAction) {
		return testCacheValue{}, cache_dto.ComputeActionDelete
	})
	if ok {
		t.Error("expected ok=false after delete")
	}
	if _, found, _ := provider.GetIfPresent(context.Background(), "key1"); found {
		t.Error("expected key to be deleted from provider")
	}
}

func TestComputeIfAbsent_AbsentKey(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	value, computed, _ := w.ComputeIfAbsent(context.Background(), "key1", func() testCacheValue {
		return testCacheValue{Name: "new", Count: 1}
	})
	if !computed {
		t.Error("expected computed=true")
	}
	if value.Name != "new" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestComputeIfAbsent_PresentKey(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "existing", Count: 5})
	_ = provider.Set(context.Background(), "key1", encoded)

	computeCalled := false
	value, computed, _ := w.ComputeIfAbsent(context.Background(), "key1", func() testCacheValue {
		computeCalled = true
		return testCacheValue{Name: "should-not-be-used"}
	})
	if computed {
		t.Error("expected computed=false for existing key")
	}
	if computeCalled {
		t.Error("compute function should not have been called")
	}
	if value.Name != "existing" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestComputeIfPresent_PresentKey(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "old", Count: 1})
	_ = provider.Set(context.Background(), "key1", encoded)

	value, ok, _ := w.ComputeIfPresent(context.Background(), "key1", func(old testCacheValue) (testCacheValue, cache_dto.ComputeAction) {
		return testCacheValue{Name: "updated", Count: old.Count + 1}, cache_dto.ComputeActionSet
	})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value.Name != "updated" || value.Count != 2 {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestComputeIfPresent_AbsentKey(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.ComputeIfPresent(context.Background(), "missing", func(old testCacheValue) (testCacheValue, cache_dto.ComputeAction) {
		t.Error("should not be called for absent key")
		return testCacheValue{}, cache_dto.ComputeActionNoop
	})
	if ok {
		t.Error("expected ok=false for absent key")
	}
}

func TestComputeWithTTL_SetAction(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	value, ok, _ := w.ComputeWithTTL(context.Background(), "key1", func(_ testCacheValue, found bool) cache_dto.ComputeResult[testCacheValue] {
		return cache_dto.ComputeResult[testCacheValue]{
			Value:  testCacheValue{Name: "ttl-computed", Count: 3},
			Action: cache_dto.ComputeActionSet,
			TTL:    5 * time.Minute,
		}
	})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if value.Name != "ttl-computed" {
		t.Errorf("unexpected value: %+v", value)
	}
}

func TestComputeWithTTL_NoopAction(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.ComputeWithTTL(context.Background(), "key1", func(_ testCacheValue, _ bool) cache_dto.ComputeResult[testCacheValue] {
		return cache_dto.ComputeResult[testCacheValue]{Action: cache_dto.ComputeActionNoop}
	})
	if ok {
		t.Error("expected ok=false for noop on absent key")
	}
}

func TestBulkGet_AllCached(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	for _, key := range []string{"a", "b"} {
		encoded, _ := CacheAPI.Marshal(testCacheValue{Name: key, Count: 1})
		_ = provider.Set(context.Background(), key, encoded)
	}

	loader := cache_dto.BulkLoaderFunc[string, testCacheValue](func(_ context.Context, keys []string) (map[string]testCacheValue, error) {
		t.Error("bulk loader should not be called when all keys are cached")
		return nil, nil
	})

	results, err := w.BulkGet(context.Background(), []string{"a", "b"}, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestBulkGet_AllMissing(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	loader := cache_dto.BulkLoaderFunc[string, testCacheValue](func(_ context.Context, keys []string) (map[string]testCacheValue, error) {
		result := make(map[string]testCacheValue, len(keys))
		for _, k := range keys {
			result[k] = testCacheValue{Name: "loaded-" + k}
		}
		return result, nil
	})

	results, err := w.BulkGet(context.Background(), []string{"x", "y"}, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results["x"].Name != "loaded-x" {
		t.Errorf("unexpected value: %+v", results["x"])
	}
}

func TestBulkGet_LoaderError(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	loader := cache_dto.BulkLoaderFunc[string, testCacheValue](func(_ context.Context, _ []string) (map[string]testCacheValue, error) {
		return nil, errors.New("bulk load failed")
	})

	_, err := w.BulkGet(context.Background(), []string{"a"}, loader)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBulkRefresh_CallsProvider(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	loaderCalled := false
	loader := cache_dto.BulkLoaderFunc[string, testCacheValue](func(_ context.Context, keys []string) (map[string]testCacheValue, error) {
		loaderCalled = true
		result := make(map[string]testCacheValue, len(keys))
		for _, k := range keys {
			result[k] = testCacheValue{Name: "refreshed-" + k}
		}
		return result, nil
	})

	w.BulkRefresh(context.Background(), []string{"r1"}, loader)
	if !loaderCalled {
		t.Error("expected loader to be called")
	}
}

func TestRefresh_Success(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	loader := cache_dto.LoaderFunc[string, testCacheValue](func(_ context.Context, key string) (testCacheValue, error) {
		return testCacheValue{Name: "refreshed-" + key}, nil
	})

	refreshChannel := w.Refresh(context.Background(), "r1", loader)
	select {
	case result := <-refreshChannel:
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		if result.Value.Name != "refreshed-r1" {
			t.Errorf("unexpected value: %+v", result.Value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for refresh result")
	}
}

func TestRefresh_LoaderError(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	loader := cache_dto.LoaderFunc[string, testCacheValue](func(_ context.Context, _ string) (testCacheValue, error) {
		return testCacheValue{}, errors.New("refresh failed")
	})

	refreshChannel := w.Refresh(context.Background(), "r1", loader)
	select {
	case result := <-refreshChannel:
		if result.Err == nil {
			t.Fatal("expected error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for refresh result")
	}
}

func TestAll_IteratesAndDecodes(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	for _, name := range []string{"a", "b", "c"} {
		encoded, _ := CacheAPI.Marshal(testCacheValue{Name: name})
		_ = provider.Set(context.Background(), name, encoded)
	}

	count := 0
	for _, v := range w.All() {
		if v.Name == "" {
			t.Error("expected decoded value to have a name")
		}
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 items, got %d", count)
	}
}

func TestAll_SkipsDecodeErrors(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "good"})
	_ = provider.Set(context.Background(), "good", encoded)
	_ = provider.Set(context.Background(), "bad", []byte("not-json"))

	count := 0
	for _, v := range w.All() {
		if v.Name != "good" {
			t.Errorf("unexpected value: %+v", v)
		}
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 item (bad one skipped), got %d", count)
	}
}

func TestKeys_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "k1", []byte("v1"))
	_ = provider.Set(context.Background(), "k2", []byte("v2"))

	keys := make(map[string]bool)
	for k := range w.Keys() {
		keys[k] = true
	}
	if len(keys) != 2 || !keys["k1"] || !keys["k2"] {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestValues_DecodesAllValues(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	for _, name := range []string{"x", "y"} {
		encoded, _ := CacheAPI.Marshal(testCacheValue{Name: name})
		_ = provider.Set(context.Background(), name, encoded)
	}

	count := 0
	for v := range w.Values() {
		if v.Name == "" {
			t.Error("expected decoded value")
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 values, got %d", count)
	}
}

func TestValues_SkipsDecodeErrors(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "ok"})
	_ = provider.Set(context.Background(), "ok", encoded)
	_ = provider.Set(context.Background(), "bad", []byte("nope"))

	count := 0
	for range w.Values() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 value (bad skipped), got %d", count)
	}
}

func TestGetEntry_Found(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "entry", Count: 5})
	_ = provider.Set(context.Background(), "key1", encoded)

	entry, ok, _ := w.GetEntry(context.Background(), "key1")
	if !ok {
		t.Fatal("expected entry to be found")
	}
	if entry.Key != "key1" {
		t.Errorf("expected key 'key1', got %q", entry.Key)
	}
	if entry.Value.Name != "entry" || entry.Value.Count != 5 {
		t.Errorf("unexpected value: %+v", entry.Value)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.GetEntry(context.Background(), "missing")
	if ok {
		t.Error("expected not found")
	}
}

func TestGetEntry_DecodeError(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "bad", []byte("not-json"))
	_, ok, _ := w.GetEntry(context.Background(), "bad")
	if ok {
		t.Error("expected false on decode error")
	}
}

func TestProbeEntry_Found(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	encoded, _ := CacheAPI.Marshal(testCacheValue{Name: "probed"})
	_ = provider.Set(context.Background(), "key1", encoded)

	entry, ok, _ := w.ProbeEntry(context.Background(), "key1")
	if !ok {
		t.Fatal("expected entry to be found")
	}
	if entry.Value.Name != "probed" {
		t.Errorf("unexpected value: %+v", entry.Value)
	}
}

func TestProbeEntry_NotFound(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, ok, _ := w.ProbeEntry(context.Background(), "missing")
	if ok {
		t.Error("expected not found")
	}
}

func TestEstimatedSize_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = provider.Set(context.Background(), "a", []byte("1"))
	_ = provider.Set(context.Background(), "b", []byte("2"))
	if w.EstimatedSize() != 2 {
		t.Errorf("expected 2, got %d", w.EstimatedSize())
	}
}

func TestStats_Passthrough(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_ = w.Stats()
}

func TestClose_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = w.Close(context.Background())
	if !provider.closed {
		t.Error("expected provider to be closed")
	}
}

func TestGetMaximum_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	provider.maximum = 1000
	w := newTestWrapper(provider)
	if w.GetMaximum() != 1000 {
		t.Errorf("expected 1000, got %d", w.GetMaximum())
	}
}

func TestSetMaximum_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	w.SetMaximum(500)
	if provider.maximum != 500 {
		t.Errorf("expected 500, got %d", provider.maximum)
	}
}

func TestWeightedSize_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	provider.weightedSize = 42
	w := newTestWrapper(provider)
	if w.WeightedSize() != 42 {
		t.Errorf("expected 42, got %d", w.WeightedSize())
	}
}

func TestSetExpiresAfter_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = w.SetExpiresAfter(context.Background(), "key1", 10*time.Minute)
	if provider.expiresAfter["key1"] != 10*time.Minute {
		t.Error("expected expiry to be forwarded")
	}
}

func TestSetRefreshableAfter_Passthrough(t *testing.T) {
	provider := newMockByteCache()
	w := newTestWrapper(provider)
	_ = w.SetRefreshableAfter(context.Background(), "key1", 5*time.Minute)
	if provider.refreshableAfter["key1"] != 5*time.Minute {
		t.Error("expected refreshable to be forwarded")
	}
}

func TestSearch_ReturnsUnsupported(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, err := w.Search(context.Background(), "query", nil)
	if !errors.Is(err, ErrSearchNotSupported) {
		t.Errorf("expected ErrSearchNotSupported, got %v", err)
	}
}

func TestQuery_ReturnsUnsupported(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	_, err := w.Query(context.Background(), nil)
	if !errors.Is(err, ErrSearchNotSupported) {
		t.Errorf("expected ErrSearchNotSupported, got %v", err)
	}
}

func TestSupportsSearch_ReturnsFalse(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	if w.SupportsSearch() {
		t.Error("expected false")
	}
}

func TestGetSchema_ReturnsNil(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	if w.GetSchema() != nil {
		t.Error("expected nil")
	}
}

func TestNewTransformerWrapper_DefaultEncoder(t *testing.T) {
	w := newTestWrapper(newMockByteCache())
	if w.encoder != nil {
		t.Error("expected nil encoder (JSON fallback)")
	}
}

func TestNewTransformerWrapper_WithEncoder(t *testing.T) {
	enc := &wrapperTestEncoder{}
	w := newTestWrapperWithEncoder(newMockByteCache(), enc)
	if w.encoder == nil {
		t.Error("expected encoder to be set")
	}
}

func TestWithWrapperEncoder_FunctionalOption(t *testing.T) {
	enc := &wrapperTestEncoder{}
	opt := withWrapperEncoder[string, testCacheValue](enc)
	config := &wrapperConfig[string, testCacheValue]{}
	opt(config)
	if config.encoder == nil {
		t.Error("expected encoder to be set on config")
	}
}
