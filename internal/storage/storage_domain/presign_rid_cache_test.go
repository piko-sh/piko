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

package storage_domain

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/wdk/clock"
)

func TestPresignRIDCache_Add(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *PresignRIDCache)
		rid      string
		expected bool
	}{
		{
			name:     "add new rid returns true",
			setup:    func(_ *PresignRIDCache) {},
			rid:      "unique-rid-123",
			expected: true,
		},
		{
			name: "add existing rid returns false",
			setup: func(c *PresignRIDCache) {
				c.Add("existing-rid", time.Now().Add(1*time.Hour))
			},
			rid:      "existing-rid",
			expected: false,
		},
		{
			name: "add different rids returns true for each",
			setup: func(c *PresignRIDCache) {
				c.Add("rid-1", time.Now().Add(1*time.Hour))
			},
			rid:      "rid-2",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
			defer cache.Stop()

			tc.setup(cache)

			result := cache.Add(tc.rid, time.Now().Add(1*time.Hour))
			if result != tc.expected {
				t.Errorf("Add(%q) = %v, want %v", tc.rid, result, tc.expected)
			}
		})
	}
}

func TestPresignRIDCache_Has(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
	defer cache.Stop()

	if cache.Has("unknown-rid") {
		t.Error("Has() should return false for unknown rid")
	}

	cache.Add("known-rid", time.Now().Add(1*time.Hour))

	if !cache.Has("known-rid") {
		t.Error("Has() should return true for known rid")
	}
}

func TestPresignRIDCache_Count(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
	defer cache.Stop()

	if cache.Count() != 0 {
		t.Errorf("expected count 0, got %d", cache.Count())
	}

	cache.Add("rid-1", time.Now().Add(1*time.Hour))
	if cache.Count() != 1 {
		t.Errorf("expected count 1, got %d", cache.Count())
	}

	cache.Add("rid-2", time.Now().Add(1*time.Hour))
	if cache.Count() != 2 {
		t.Errorf("expected count 2, got %d", cache.Count())
	}

	cache.Add("rid-1", time.Now().Add(1*time.Hour))
	if cache.Count() != 2 {
		t.Errorf("expected count 2 after duplicate, got %d", cache.Count())
	}
}

func TestPresignRIDCache_Clear(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
	defer cache.Stop()

	cache.Add("rid-1", time.Now().Add(1*time.Hour))
	cache.Add("rid-2", time.Now().Add(1*time.Hour))
	cache.Add("rid-3", time.Now().Add(1*time.Hour))

	if cache.Count() != 3 {
		t.Fatalf("expected count 3, got %d", cache.Count())
	}

	cache.Clear()

	if cache.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", cache.Count())
	}

	if cache.Has("rid-1") {
		t.Error("Has() should return false after clear")
	}

	if !cache.Add("rid-1", time.Now().Add(1*time.Hour)) {
		t.Error("Add() should return true after clear")
	}
}

func TestPresignRIDCache_ReplayProtection(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
	defer cache.Stop()

	rid := "replay-test-rid"
	expiry := time.Now().Add(1 * time.Hour)

	if !cache.Add(rid, expiry) {
		t.Error("first Add() should return true")
	}

	for i := range 10 {
		if cache.Add(rid, expiry) {
			t.Errorf("replay attempt %d should return false", i+1)
		}
	}
}

func TestPresignRIDCache_ConcurrentAccess(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 1*time.Hour)
	defer cache.Stop()

	var wg sync.WaitGroup
	goroutines := 100
	ridsPerGoroutine := 100

	successCount := make(chan int, goroutines)

	for range goroutines {
		wg.Go(func() {
			count := 0
			for range ridsPerGoroutine {
				rid := "shared-rid"
				if cache.Add(rid, time.Now().Add(1*time.Hour)) {
					count++
				}
			}
			successCount <- count
		})
	}

	wg.Wait()
	close(successCount)

	total := 0
	for count := range successCount {
		total += count
	}

	if total != 1 {
		t.Errorf("expected exactly 1 successful add for shared rid, got %d", total)
	}
}

func TestPresignRIDCache_ExpiryCleanup(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := clock.NewMockClock(now)
	baseline := mock.TimerCount()

	cache := NewPresignRIDCache(context.Background(), 50*time.Millisecond, WithPresignRIDCacheClock(mock))
	defer cache.Stop()

	if !mock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for cleanup ticker setup")
	}

	cache.Add("short-lived-1", now.Add(10*time.Millisecond))
	cache.Add("short-lived-2", now.Add(10*time.Millisecond))
	cache.Add("long-lived", now.Add(1*time.Hour))

	if cache.Count() != 3 {
		t.Fatalf("expected count 3, got %d", cache.Count())
	}

	mock.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	if cache.Count() != 1 {
		t.Errorf("expected count 1 after cleanup, got %d", cache.Count())
	}

	if cache.Has("short-lived-1") {
		t.Error("short-lived-1 should have been purged")
	}
	if cache.Has("short-lived-2") {
		t.Error("short-lived-2 should have been purged")
	}
	if !cache.Has("long-lived") {
		t.Error("long-lived should still exist")
	}
}

func TestPresignRIDCache_Stop(t *testing.T) {
	cache := NewPresignRIDCache(context.Background(), 50*time.Millisecond)

	cache.Add("test-rid", time.Now().Add(1*time.Hour))

	cache.Stop()

	cache.Stop()
	cache.Stop()

	if !cache.Has("test-rid") {
		t.Error("rid should still be readable after Stop")
	}
}

func TestPresignRIDCache_DefaultCleanupInterval(t *testing.T) {

	cache := NewPresignRIDCache(context.Background(), 0)
	defer cache.Stop()

	if !cache.Add("test-rid", time.Now().Add(1*time.Hour)) {
		t.Error("Add should work with default cleanup interval")
	}
}
