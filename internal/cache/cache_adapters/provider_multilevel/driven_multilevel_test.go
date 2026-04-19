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

package provider_multilevel

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_dto"
)

func newTestAdapter() (
	*MultiLevelAdapter[string, string],
	*provider_mock.MockAdapter[string, string],
	*provider_mock.MockAdapter[string, string],
) {
	l1 := provider_mock.NewMockAdapter[string, string]()
	l2 := provider_mock.NewMockAdapter[string, string]()
	adapter := NewMultiLevelAdapter[string, string](context.Background(), "test", l1, l2, Config{
		MaxConsecutiveFailures: 5,
		OpenStateTimeout:       30 * time.Second,
	})
	return adapter, l1, l2
}

func waitForAsync() {
	runtime.Gosched()
	time.Sleep(50 * time.Millisecond)
}

func TestGetIfPresent_L1Hit(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "value1")

	value, ok, err := adapter.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected hit, got miss")
	}
	if value != "value1" {
		t.Errorf("got %q, want %q", value, "value1")
	}

	l2Calls := l2.GetSetCalls()
	if len(l2Calls) != 0 {
		t.Errorf("expected no L2 set calls, got %d", len(l2Calls))
	}
}

func TestGetIfPresent_L1Miss_L2Hit(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l2.Set(ctx, "key1", "from-l2")

	value, ok, err := adapter.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected hit from L2, got miss")
	}
	if value != "from-l2" {
		t.Errorf("got %q, want %q", value, "from-l2")
	}

	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok {
		t.Fatal("expected L1 back-population, key not found")
	}
	if l1Val != "from-l2" {
		t.Errorf("L1 value: got %q, want %q", l1Val, "from-l2")
	}
}

func TestGetIfPresent_BothMiss(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	_, ok, err := adapter.GetIfPresent(ctx, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected miss, got hit")
	}
}

func TestGet_L1Hit_LoaderNotCalled(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "cached")

	loaderCalled := false
	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, _ string) (string, error) {
		loaderCalled = true
		return "loaded", nil
	})

	value, err := adapter.Get(ctx, "key1", loader)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "cached" {
		t.Errorf("got %q, want %q", value, "cached")
	}
	if loaderCalled {
		t.Error("loader should not have been called on L1 hit")
	}
}

func TestGet_L1Miss_L2Hit(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l2.Set(ctx, "key1", "from-l2")

	loaderCalled := false
	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, _ string) (string, error) {
		loaderCalled = true
		return "loaded", nil
	})

	value, err := adapter.Get(ctx, "key1", loader)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "from-l2" {
		t.Errorf("got %q, want %q", value, "from-l2")
	}
	if loaderCalled {
		t.Error("loader should not have been called when L2 has the value")
	}

	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok {
		t.Fatal("expected L1 back-population")
	}
	if l1Val != "from-l2" {
		t.Errorf("L1 value: got %q, want %q", l1Val, "from-l2")
	}
}

func TestGet_BothMiss_LoaderCalled(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	value, err := adapter.Get(ctx, "key1", cache_dto.LoaderFunc[string, string](
		func(_ context.Context, key string) (string, error) {
			return "loaded-" + key, nil
		},
	))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "loaded-key1" {
		t.Errorf("got %q, want %q", value, "loaded-key1")
	}

	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok {
		t.Fatal("expected value in L1 after load")
	}
	if l1Val != "loaded-key1" {
		t.Errorf("L1 value: got %q, want %q", l1Val, "loaded-key1")
	}

	waitForAsync()
	l2Val, l2Ok, l2Err := l2.GetIfPresent(ctx, "key1")
	if l2Err != nil {
		t.Fatalf("unexpected L2 error: %v", l2Err)
	}
	if !l2Ok {
		t.Fatal("expected async write-back to L2")
	}
	if l2Val != "loaded-key1" {
		t.Errorf("L2 value: got %q, want %q", l2Val, "loaded-key1")
	}
}

func TestGet_LoaderError(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	expectedErr := errors.New("load failed")
	_, err := adapter.Get(ctx, "key1", cache_dto.LoaderFunc[string, string](
		func(_ context.Context, _ string) (string, error) {
			return "", expectedErr
		},
	))
	if err == nil {
		t.Fatal("expected error from loader, got nil")
	}
}

func TestSet_WritesToBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	if err := adapter.Set(ctx, "key1", "value1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok || l1Val != "value1" {
		t.Errorf("L1: got (%q, %v), want (%q, true)", l1Val, l1Ok, "value1")
	}

	l2Val, l2Ok, l2Err := l2.GetIfPresent(ctx, "key1")
	if l2Err != nil {
		t.Fatalf("unexpected L2 error: %v", l2Err)
	}
	if !l2Ok || l2Val != "value1" {
		t.Errorf("L2: got (%q, %v), want (%q, true)", l2Val, l2Ok, "value1")
	}
}

func TestSet_PropagatesTags(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	if err := adapter.Set(ctx, "key1", "value1", "tag-a", "tag-b"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	l1Calls := l1.GetSetCalls()
	if len(l1Calls) == 0 {
		t.Fatal("expected L1 set call")
	}
	lastL1Call := l1Calls[len(l1Calls)-1]
	if len(lastL1Call.Tags) != 2 || lastL1Call.Tags[0] != "tag-a" || lastL1Call.Tags[1] != "tag-b" {
		t.Errorf("L1 tags: got %v, want [tag-a tag-b]", lastL1Call.Tags)
	}

	l2Calls := l2.GetSetCalls()
	if len(l2Calls) == 0 {
		t.Fatal("expected L2 set call")
	}
	lastL2Call := l2Calls[len(l2Calls)-1]
	if len(lastL2Call.Tags) != 2 || lastL2Call.Tags[0] != "tag-a" || lastL2Call.Tags[1] != "tag-b" {
		t.Errorf("L2 tags: got %v, want [tag-a tag-b]", lastL2Call.Tags)
	}
}

func TestSetWithTTL_WritesToBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	err := adapter.SetWithTTL(ctx, "key1", "value1", 5*time.Minute)
	if err != nil {
		t.Fatalf("SetWithTTL failed: %v", err)
	}

	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok || l1Val != "value1" {
		t.Errorf("L1: got (%q, %v), want (%q, true)", l1Val, l1Ok, "value1")
	}

	l2Val, l2Ok, l2Err := l2.GetIfPresent(ctx, "key1")
	if l2Err != nil {
		t.Fatalf("unexpected L2 error: %v", l2Err)
	}
	if !l2Ok || l2Val != "value1" {
		t.Errorf("L2: got (%q, %v), want (%q, true)", l2Val, l2Ok, "value1")
	}
}

func TestInvalidate_RemovesFromBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "v")
	_ = l2.Set(ctx, "key1", "v")

	if err := adapter.Invalidate(ctx, "key1"); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	if _, ok, _ := l1.GetIfPresent(ctx, "key1"); ok {
		t.Error("expected key removed from L1")
	}
	if _, ok, _ := l2.GetIfPresent(ctx, "key1"); ok {
		t.Error("expected key removed from L2")
	}
}

func TestInvalidateByTags_RemovesFromBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "v", "mytag")
	_ = l2.Set(ctx, "key1", "v", "mytag")

	count, err := adapter.InvalidateByTags(ctx, "mytag")
	if err != nil {
		t.Fatalf("InvalidateByTags failed: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one invalidation from L1")
	}

	if _, ok, _ := l1.GetIfPresent(ctx, "key1"); ok {
		t.Error("expected key removed from L1")
	}

	if _, ok, _ := l2.GetIfPresent(ctx, "key1"); ok {
		t.Error("expected key removed from L2")
	}
}

func TestInvalidateAll_ClearsBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l1.Set(ctx, "b", "2")
	_ = l2.Set(ctx, "a", "1")
	_ = l2.Set(ctx, "b", "2")

	if err := adapter.InvalidateAll(ctx); err != nil {
		t.Fatalf("InvalidateAll failed: %v", err)
	}

	if l1.EstimatedSize() != 0 {
		t.Errorf("expected L1 empty, got %d entries", l1.EstimatedSize())
	}
	if l2.GetInvalidateAllCount() == 0 {
		t.Error("expected InvalidateAll called on L2")
	}
}

func TestBulkGet_AllInL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l1.Set(ctx, "b", "2")

	loaderCalled := false
	results, err := adapter.BulkGet(ctx, []string{"a", "b"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
			loaderCalled = true
			return nil, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if loaderCalled {
		t.Error("loader should not be called when all keys in L1")
	}
}

func TestBulkGet_L1Miss_L2Hit(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l2.Set(ctx, "b", "2")

	results, err := adapter.BulkGet(ctx, []string{"a", "b"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
			return nil, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if results["a"] != "1" {
		t.Errorf("key 'a': got %q, want %q", results["a"], "1")
	}
	if results["b"] != "2" {
		t.Errorf("key 'b': got %q, want %q", results["b"], "2")
	}

	waitForAsync()
	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "b")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok {
		t.Fatal("expected L1 back-population for key 'b'")
	}
	if l1Val != "2" {
		t.Errorf("L1 back-populated value: got %q, want %q", l1Val, "2")
	}
}

func TestBulkGet_AllMiss_LoaderCalled(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	results, err := adapter.BulkGet(ctx, []string{"x", "y"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
			out := make(map[string]string, len(keys))
			for _, k := range keys {
				out[k] = "loaded-" + k
			}
			return out, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if results["x"] != "loaded-x" || results["y"] != "loaded-y" {
		t.Errorf("unexpected results: %v", results)
	}

	if v, ok, _ := l1.GetIfPresent(ctx, "x"); !ok || v != "loaded-x" {
		t.Errorf("L1 key 'x': got (%q, %v), want (%q, true)", v, ok, "loaded-x")
	}

	waitForAsync()
	if v, ok, _ := l2.GetIfPresent(ctx, "x"); !ok || v != "loaded-x" {
		t.Errorf("L2 key 'x': got (%q, %v), want (%q, true)", v, ok, "loaded-x")
	}
}

func TestBulkGet_EmptyKeys(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	results, err := adapter.BulkGet(ctx, []string{},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
			t.Fatal("loader should not be called for empty keys")
			return nil, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestBulkGet_LoaderError(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	expectedErr := errors.New("bulk load failed")
	results, err := adapter.BulkGet(ctx, []string{"a"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
			return nil, expectedErr
		}))
	if err == nil {
		t.Fatal("expected error from bulk loader, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected empty results on error, got %d", len(results))
	}
}

func TestBulkSet_WritesToBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	err := adapter.BulkSet(ctx, map[string]string{"a": "1", "b": "2"})
	if err != nil {
		t.Fatalf("BulkSet failed: %v", err)
	}

	for _, key := range []string{"a", "b"} {
		if _, ok, _ := l1.GetIfPresent(ctx, key); !ok {
			t.Errorf("expected key %q in L1", key)
		}
		if _, ok, _ := l2.GetIfPresent(ctx, key); !ok {
			t.Errorf("expected key %q in L2", key)
		}
	}
}

func TestBulkSet_EmptyItems(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	err := adapter.BulkSet(ctx, map[string]string{})
	if err != nil {
		t.Fatalf("BulkSet with empty items should succeed, got: %v", err)
	}
}

func TestBulkSet_WithTags(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	err := adapter.BulkSet(ctx, map[string]string{"a": "1"}, "tag-x")
	if err != nil {
		t.Fatalf("BulkSet failed: %v", err)
	}

	if _, ok, _ := l1.GetIfPresent(ctx, "a"); !ok {
		t.Error("expected key 'a' in L1")
	}
	if _, ok, _ := l2.GetIfPresent(ctx, "a"); !ok {
		t.Error("expected key 'a' in L2")
	}

	count, tagErr := l1.InvalidateByTags(ctx, "tag-x")
	if tagErr != nil {
		t.Fatalf("InvalidateByTags failed: %v", tagErr)
	}
	if count == 0 {
		t.Error("expected tag invalidation to remove entries from L1")
	}
}

func TestCompute_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "old")

	value, ok, err := adapter.Compute(ctx, "key1", func(oldValue string, found bool) (string, cache_dto.ComputeAction) {
		if !found || oldValue != "old" {
			t.Errorf("expected found=true, oldValue=%q", oldValue)
		}
		return "new", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	if !ok || value != "new" {
		t.Errorf("Compute: got (%q, %v), want (%q, true)", value, ok, "new")
	}
}

func TestComputeIfAbsent_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	value, computed, err := adapter.ComputeIfAbsent(ctx, "key1", func() string { return "computed" })
	if err != nil {
		t.Fatalf("ComputeIfAbsent failed: %v", err)
	}
	if !computed || value != "computed" {
		t.Errorf("ComputeIfAbsent: got (%q, %v), want (%q, true)", value, computed, "computed")
	}
}

func TestComputeIfPresent_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "old")

	value, ok, err := adapter.ComputeIfPresent(ctx, "key1", func(oldValue string) (string, cache_dto.ComputeAction) {
		return "updated", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("ComputeIfPresent failed: %v", err)
	}
	if !ok || value != "updated" {
		t.Errorf("ComputeIfPresent: got (%q, %v), want (%q, true)", value, ok, "updated")
	}
}

func TestComputeWithTTL_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	value, ok, err := adapter.ComputeWithTTL(ctx, "key1", func(_ string, _ bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{
			Value:  "ttl-value",
			Action: cache_dto.ComputeActionSet,
			TTL:    time.Minute,
		}
	})
	if err != nil {
		t.Fatalf("ComputeWithTTL failed: %v", err)
	}
	if !ok || value != "ttl-value" {
		t.Errorf("ComputeWithTTL: got (%q, %v), want (%q, true)", value, ok, "ttl-value")
	}
}

func TestEstimatedSize_DelegatesToL1(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l1.Set(ctx, "b", "2")
	_ = l2.Set(ctx, "c", "3")

	size := adapter.EstimatedSize()
	if size != 2 {
		t.Errorf("expected EstimatedSize=2 (L1 only), got %d", size)
	}
}

func TestGetMaximum_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	l1.SetMaximum(1000)

	if adapter.GetMaximum() != 1000 {
		t.Errorf("expected GetMaximum=1000, got %d", adapter.GetMaximum())
	}
}

func TestSetMaximum_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	adapter.SetMaximum(500)

	if l1.GetMaximum() != 500 {
		t.Errorf("expected L1 maximum=500, got %d", l1.GetMaximum())
	}
}

func TestWeightedSize_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")

	size := adapter.WeightedSize()
	l1Size := l1.WeightedSize()
	if size != l1Size {
		t.Errorf("expected WeightedSize=%d (from L1), got %d", l1Size, size)
	}
}

func TestStats_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_, _, _ = l1.GetIfPresent(ctx, "a")

	stats := adapter.Stats()
	l1Stats := l1.Stats()
	if stats.Hits != l1Stats.Hits {
		t.Errorf("expected Hits=%d (from L1), got %d", l1Stats.Hits, stats.Hits)
	}
}

func TestGetEntry_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "value1")

	entry, ok, err := adapter.GetEntry(ctx, "key1")
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if !ok {
		t.Fatal("expected entry found")
	}
	if entry.Key != "key1" || entry.Value != "value1" {
		t.Errorf("got entry (%q, %q), want (%q, %q)", entry.Key, entry.Value, "key1", "value1")
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	_, ok, err := adapter.GetEntry(ctx, "missing")
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if ok {
		t.Error("expected entry not found")
	}
}

func TestProbeEntry_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "value1")

	entry, ok, err := adapter.ProbeEntry(ctx, "key1")
	if err != nil {
		t.Fatalf("ProbeEntry failed: %v", err)
	}
	if !ok {
		t.Fatal("expected entry found")
	}
	if entry.Key != "key1" || entry.Value != "value1" {
		t.Errorf("got (%q, %q), want (%q, %q)", entry.Key, entry.Value, "key1", "value1")
	}
}

func TestSetExpiresAfter_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "value1")

	if err := adapter.SetExpiresAfter(ctx, "key1", 5*time.Minute); err != nil {
		t.Fatalf("SetExpiresAfter failed: %v", err)
	}

	if _, ok, _ := l1.GetIfPresent(ctx, "key1"); !ok {
		t.Error("expected key still present after SetExpiresAfter")
	}
}

func TestSetRefreshableAfter_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "value1")

	if err := adapter.SetRefreshableAfter(ctx, "key1", 2*time.Minute); err != nil {
		t.Fatalf("SetRefreshableAfter failed: %v", err)
	}
}

func TestRefresh_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "old")

	refreshChannel := adapter.Refresh(ctx, "key1", cache_dto.LoaderFunc[string, string](
		func(_ context.Context, _ string) (string, error) {
			return "refreshed", nil
		},
	))

	result := <-refreshChannel
	if result.Err != nil {
		t.Fatalf("Refresh failed: %v", result.Err)
	}
	if result.Value != "refreshed" {
		t.Errorf("got %q, want %q", result.Value, "refreshed")
	}
}

func TestBulkRefresh_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l1.Set(ctx, "b", "2")

	adapter.BulkRefresh(ctx, []string{"a", "b"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
			out := make(map[string]string, len(keys))
			for _, k := range keys {
				out[k] = "refreshed-" + k
			}
			return out, nil
		}))

	waitForAsync()
}

func TestSupportsSearch_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()

	if adapter.SupportsSearch() {
		t.Error("expected SupportsSearch=false from mock L1")
	}
}

func TestGetSchema_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()

	if adapter.GetSchema() != nil {
		t.Error("expected nil schema from mock L1")
	}
}

func TestGet_AsyncWritebackCompletesAfterContextCancellation(t *testing.T) {
	adapter, _, l2 := newTestAdapter()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	value, err := adapter.Get(ctx, "key1", cache_dto.LoaderFunc[string, string](
		func(_ context.Context, key string) (string, error) {
			return "loaded-" + key, nil
		},
	))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "loaded-key1" {
		t.Errorf("got %q, want %q", value, "loaded-key1")
	}

	waitForAsync()
	l2Val, l2Ok, l2Err := l2.GetIfPresent(context.Background(), "key1")
	if l2Err != nil {
		t.Fatalf("unexpected L2 error: %v", l2Err)
	}
	if !l2Ok {
		t.Fatal("expected async L2 writeback to complete despite cancelled caller context")
	}
	if l2Val != "loaded-key1" {
		t.Errorf("L2 value: got %q, want %q", l2Val, "loaded-key1")
	}
}

func TestBulkGet_L2BackPopulationCompletesAfterContextCancellation(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	bgCtx := context.Background()
	_ = l2.Set(bgCtx, "a", "from-l2")

	ctx, cancel := context.WithCancelCause(bgCtx)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	results, err := adapter.BulkGet(ctx, []string{"a"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
			t.Fatal("loader should not be called when value is in L2")
			return nil, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if results["a"] != "from-l2" {
		t.Errorf("got %q, want %q", results["a"], "from-l2")
	}

	waitForAsync()
	l1Val, l1Ok, l1Err := l1.GetIfPresent(bgCtx, "a")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok {
		t.Fatal("expected L1 back-population to complete despite cancelled caller context")
	}
	if l1Val != "from-l2" {
		t.Errorf("L1 value: got %q, want %q", l1Val, "from-l2")
	}
}

func TestBulkGet_L2WritebackCompletesAfterContextCancellation(t *testing.T) {
	adapter, _, l2 := newTestAdapter()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	results, err := adapter.BulkGet(ctx, []string{"x"},
		cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, keys []string) (map[string]string, error) {
			out := make(map[string]string, len(keys))
			for _, k := range keys {
				out[k] = "loaded-" + k
			}
			return out, nil
		}))
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}
	if results["x"] != "loaded-x" {
		t.Errorf("got %q, want %q", results["x"], "loaded-x")
	}

	waitForAsync()
	l2Val, l2Ok, l2Err := l2.GetIfPresent(context.Background(), "x")
	if l2Err != nil {
		t.Fatalf("unexpected L2 error: %v", l2Err)
	}
	if !l2Ok {
		t.Fatal("expected async L2 writeback to complete despite cancelled caller context")
	}
	if l2Val != "loaded-x" {
		t.Errorf("L2 value: got %q, want %q", l2Val, "loaded-x")
	}
}

func TestSearch_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	_, err := adapter.Search(ctx, "query", nil)
	if err == nil {
		t.Error("expected error from mock L1 (ErrSearchNotSupported)")
	}
}

func TestQuery_DelegatesToL1(t *testing.T) {
	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	_, err := adapter.Query(ctx, nil)
	if err == nil {
		t.Error("expected error from mock L1 (ErrSearchNotSupported)")
	}
}

func TestClose_ClosesBothLevels(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()

	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_ = l1.Close(ctx)
	_ = l2.Close(ctx)
}

func TestClose_BothFail_ReturnsJoinedError(t *testing.T) {
	t.Parallel()

	l1Err := errors.New("l1 close failed")
	l2Err := errors.New("l2 close failed")

	l1 := &closeFailingMock{
		MockAdapter: provider_mock.NewMockAdapter[string, string](),
		closeErr:    l1Err,
	}
	l2 := &closeFailingMock{
		MockAdapter: provider_mock.NewMockAdapter[string, string](),
		closeErr:    l2Err,
	}

	adapter := NewMultiLevelAdapter[string, string](context.Background(), "joined-close",
		l1, l2, Config{MaxConsecutiveFailures: 5, OpenStateTimeout: 30 * time.Second})

	err := adapter.Close(context.Background())
	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	if !errors.Is(err, l1Err) {
		t.Errorf("expected error to wrap l1 close failure, got %v", err)
	}
	if !errors.Is(err, l2Err) {
		t.Errorf("expected error to wrap l2 close failure, got %v", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	adapter, _, _ := newTestAdapter()
	ctx := context.Background()

	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("second Close must remain a safe no-op: %v", err)
	}
	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("third Close must remain a safe no-op: %v", err)
	}
}

type closeFailingMock struct {
	*provider_mock.MockAdapter[string, string]
	closeErr error
}

func (m *closeFailingMock) Close(_ context.Context) error {
	return m.closeErr
}

func TestGetIfPresent_L2Error_ReturnsL1Value(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "key1", "l1-value")
	l2.SetError(errors.New("l2 down"))

	value, ok, err := adapter.GetIfPresent(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected L1 hit despite L2 error")
	}
	if value != "l1-value" {
		t.Errorf("got %q, want %q", value, "l1-value")
	}
}

func TestSet_L2Error_StillWritesToL1(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	l2.SetError(errors.New("l2 down"))

	_ = adapter.Set(ctx, "key1", "value1")

	l2.SetError(nil)
	l1Val, l1Ok, l1Err := l1.GetIfPresent(ctx, "key1")
	if l1Err != nil {
		t.Fatalf("unexpected L1 error: %v", l1Err)
	}
	if !l1Ok || l1Val != "value1" {
		t.Errorf("L1 should still have value despite L2 error: got (%q, %v)", l1Val, l1Ok)
	}
}

func TestAll_DelegatesToL1(t *testing.T) {
	adapter, l1, l2 := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "a", "1")
	_ = l2.Set(ctx, "b", "2")

	count := 0
	for k, v := range adapter.All() {
		count++
		if k != "a" || v != "1" {
			t.Errorf("unexpected entry: (%q, %q)", k, v)
		}
	}
	if count != 1 {
		t.Errorf("expected 1 entry from L1, got %d", count)
	}
}

func TestKeys_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "x", "1")
	_ = l1.Set(ctx, "y", "2")

	keys := make(map[string]struct{})
	for k := range adapter.Keys() {
		keys[k] = struct{}{}
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestValues_DelegatesToL1(t *testing.T) {
	adapter, l1, _ := newTestAdapter()
	ctx := context.Background()
	_ = l1.Set(ctx, "x", "val-x")

	values := make([]string, 0)
	for v := range adapter.Values() {
		values = append(values, v)
	}
	if len(values) != 1 || values[0] != "val-x" {
		t.Errorf("expected [val-x], got %v", values)
	}
}
