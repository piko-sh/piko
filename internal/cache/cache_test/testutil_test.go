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

package cache_test

import (
	"sync"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

func RunConcurrent(t *testing.T, n int, callback func(int)) {
	t.Helper()

	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(index int) {
			defer wg.Done()
			callback(index)
		}(i)
	}

	wg.Wait()
}

func AssertStats(t *testing.T, expected, actual cache_dto.Stats) {
	t.Helper()

	if actual.Hits != expected.Hits {
		t.Errorf("Hits: got %d, want %d", actual.Hits, expected.Hits)
	}
	if actual.Misses != expected.Misses {
		t.Errorf("Misses: got %d, want %d", actual.Misses, expected.Misses)
	}
	if actual.LoadSuccessCount != expected.LoadSuccessCount {
		t.Errorf("LoadSuccessCount: got %d, want %d", actual.LoadSuccessCount, expected.LoadSuccessCount)
	}
	if actual.LoadFailureCount != expected.LoadFailureCount {
		t.Errorf("LoadFailureCount: got %d, want %d", actual.LoadFailureCount, expected.LoadFailureCount)
	}
	if actual.Evictions != expected.Evictions {
		t.Errorf("Evictions: got %d, want %d", actual.Evictions, expected.Evictions)
	}
}

func AssertStatsRange(t *testing.T, actual cache_dto.Stats, minHits, maxHits, minMisses, maxMisses uint64) {
	t.Helper()

	if actual.Hits < minHits || actual.Hits > maxHits {
		t.Errorf("Hits %d not in range [%d, %d]", actual.Hits, minHits, maxHits)
	}
	if actual.Misses < minMisses || actual.Misses > maxMisses {
		t.Errorf("Misses %d not in range [%d, %d]", actual.Misses, minMisses, maxMisses)
	}
}

func Equal[T comparable](t *testing.T, got, want T, message string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: got %v, want %v", message, got, want)
	}
}

func NotEqual[T comparable](t *testing.T, got, notWant T, message string) {
	t.Helper()

	if got == notWant {
		t.Errorf("%s: got %v, but did not want %v", message, got, notWant)
	}
}

func Nil(t *testing.T, got any, message string) {
	t.Helper()

	if got != nil {
		t.Errorf("%s: expected nil, got %v", message, got)
	}
}

func NotNil(t *testing.T, got any, message string) {
	t.Helper()

	if got == nil {
		t.Errorf("%s: expected non-nil value", message)
	}
}

func ErrorContains(t *testing.T, err error, substring string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error containing %q, got nil", substring)
		return
	}

	if !contains(err.Error(), substring) {
		t.Errorf("error %q does not contain %q", err.Error(), substring)
	}
}

func NoError(t *testing.T, err error, message string) {
	t.Helper()

	if err != nil {
		t.Errorf("%s: unexpected error: %v", message, err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexSubstring(s, substr) >= 0)
}

func indexSubstring(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
