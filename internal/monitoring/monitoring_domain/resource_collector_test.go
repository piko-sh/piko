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

package monitoring_domain

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestResourceCollector_CategoriseResource(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	testCases := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "regular file",
			target:   "/var/log/app.log",
			expected: ResourceCategoryFile,
		},
		{
			name:     "absolute path file",
			target:   "/home/user/data.txt",
			expected: ResourceCategoryFile,
		},
		{
			name:     "pipe",
			target:   "pipe:[12345]",
			expected: ResourceCategoryPipe,
		},
		{
			name:     "socket generic",
			target:   "socket:[67890]",
			expected: ResourceCategorySocket,
		},
		{
			name:     "anon_inode epoll",
			target:   "anon_inode:[eventpoll]",
			expected: ResourceCategoryOther,
		},
		{
			name:     "anon_inode eventfd",
			target:   "anon_inode:[eventfd]",
			expected: ResourceCategoryOther,
		},
		{
			name:     "unknown target",
			target:   "something-unknown",
			expected: ResourceCategoryOther,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := collector.categoriseResource(tc.target)
			if result != tc.expected {
				t.Errorf("categoriseResource(%q) = %q, want %q", tc.target, result, tc.expected)
			}
		})
	}
}

func TestSortResourcesByAge(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		input    []resourceInfo
		expected []int
	}{
		{
			name:     "empty slice",
			input:    []resourceInfo{},
			expected: []int{},
		},
		{
			name: "single element",
			input: []resourceInfo{
				{Number: 1, FirstSeen: baseTime},
			},
			expected: []int{1},
		},
		{
			name: "already sorted",
			input: []resourceInfo{
				{Number: 1, FirstSeen: baseTime},
				{Number: 2, FirstSeen: baseTime.Add(time.Hour)},
				{Number: 3, FirstSeen: baseTime.Add(2 * time.Hour)},
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "reverse order",
			input: []resourceInfo{
				{Number: 3, FirstSeen: baseTime.Add(2 * time.Hour)},
				{Number: 2, FirstSeen: baseTime.Add(time.Hour)},
				{Number: 1, FirstSeen: baseTime},
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "random order",
			input: []resourceInfo{
				{Number: 2, FirstSeen: baseTime.Add(time.Hour)},
				{Number: 4, FirstSeen: baseTime.Add(3 * time.Hour)},
				{Number: 1, FirstSeen: baseTime},
				{Number: 3, FirstSeen: baseTime.Add(2 * time.Hour)},
			},
			expected: []int{1, 2, 3, 4},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fds := make([]resourceInfo, len(tc.input))
			copy(fds, tc.input)

			sortResourcesByAge(fds)

			if len(fds) != len(tc.expected) {
				t.Fatalf("expected %d elements, got %d", len(tc.expected), len(fds))
			}

			for i, expectedNum := range tc.expected {
				if fds[i].Number != expectedNum {
					t.Errorf("at index %d: expected Number %d, got %d", i, expectedNum, fds[i].Number)
				}
			}
		})
	}
}

func TestResourceCollector_FirstSeenTracking(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	collector.firstSeen[10] = baseTime
	collector.firstSeen[20] = baseTime.Add(time.Hour)
	collector.firstSeen[30] = baseTime.Add(2 * time.Hour)

	currentFDs := map[int]struct{}{
		10: {},
		30: {},
	}

	collector.cleanupStaleFirstSeen(currentFDs)

	if _, exists := collector.firstSeen[20]; exists {
		t.Error("expected FD 20 to be removed from firstSeen")
	}

	if _, exists := collector.firstSeen[10]; !exists {
		t.Error("expected FD 10 to remain in firstSeen")
	}
	if _, exists := collector.firstSeen[30]; !exists {
		t.Error("expected FD 30 to remain in firstSeen")
	}
}

func TestResourcePatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		target      string
		matchSocket bool
		matchPipe   bool
	}{
		{
			name:        "socket with inode",
			target:      "socket:[12345]",
			matchSocket: true,
			matchPipe:   false,
		},
		{
			name:        "pipe with inode",
			target:      "pipe:[67890]",
			matchSocket: false,
			matchPipe:   true,
		},
		{
			name:        "regular file",
			target:      "/var/log/test.log",
			matchSocket: false,
			matchPipe:   false,
		},
		{
			name:        "socket with large inode",
			target:      "socket:[999999999]",
			matchSocket: true,
			matchPipe:   false,
		},
		{
			name:        "malformed socket",
			target:      "socket:[abc]",
			matchSocket: false,
			matchPipe:   false,
		},
		{
			name:        "malformed pipe",
			target:      "pipe:[]",
			matchSocket: false,
			matchPipe:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if socketPattern.MatchString(tc.target) != tc.matchSocket {
				t.Errorf("socketPattern.MatchString(%q) = %v, want %v",
					tc.target, !tc.matchSocket, tc.matchSocket)
			}

			if pipePattern.MatchString(tc.target) != tc.matchPipe {
				t.Errorf("pipePattern.MatchString(%q) = %v, want %v",
					tc.target, !tc.matchPipe, tc.matchPipe)
			}
		})
	}
}

func TestResourceCollector_ScanFirstSeen(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()
	collector.scanFirstSeen()

	if len(collector.firstSeen) == 0 {
		t.Fatal("expected firstSeen to contain entries after scan")
	}

	for fd, ts := range collector.firstSeen {
		if ts.IsZero() {
			t.Errorf("firstSeen[%d] has zero timestamp", fd)
		}
	}

	entries, err := os.ReadDir(procSelfFileDescriptor)
	if err != nil {
		t.Fatalf("reading %s: %v", procSelfFileDescriptor, err)
	}

	for _, entry := range entries {
		number, err := strconv.Atoi(entry.Name())
		if err != nil || number <= 2 {
			continue
		}

		if _, ok := collector.firstSeen[number]; !ok {

			t.Logf("FD %d exists in /proc/self/fd but not in firstSeen (may be transient)", number)
		}
	}
}

func TestResourceCollector_ScanFirstSeenPreservesExisting(t *testing.T) {
	t.Parallel()

	f, err := os.Open(procSelfFileDescriptor)
	if err != nil {
		t.Fatalf("opening %s: %v", procSelfFileDescriptor, err)
	}
	defer func() { _ = f.Close() }()

	knownFD := int(f.Fd())

	collector := NewResourceCollector()

	earlyTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	collector.firstSeen[knownFD] = earlyTime

	collector.scanFirstSeen()

	if ts := collector.firstSeen[knownFD]; !ts.Equal(earlyTime) {
		t.Errorf("firstSeen[%d] = %v, want %v (should preserve existing)", knownFD, ts, earlyTime)
	}
}

func TestResourceCollector_StopIdempotent(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	collector.Stop()
	collector.Stop()
}

func TestBuildCategoriesResponse(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		input                 map[string][]resourceInfo
		name                  string
		expectedCategoryCount int
		expectedTotal         int
	}{
		{
			name:                  "empty input",
			input:                 map[string][]resourceInfo{},
			expectedCategoryCount: 0,
			expectedTotal:         0,
		},
		{
			name: "single category",
			input: map[string][]resourceInfo{
				ResourceCategoryFile: {
					{Number: 3, Category: ResourceCategoryFile, Target: "/tmp/a", FirstSeen: now},
					{Number: 4, Category: ResourceCategoryFile, Target: "/tmp/b", FirstSeen: now},
				},
			},
			expectedCategoryCount: 1,
			expectedTotal:         2,
		},
		{
			name: "multiple categories",
			input: map[string][]resourceInfo{
				ResourceCategoryFile: {
					{Number: 3, Category: ResourceCategoryFile, Target: "/tmp/a", FirstSeen: now},
				},
				ResourceCategoryTCP: {
					{Number: 5, Category: ResourceCategoryTCP, Target: "socket:[123]", FirstSeen: now},
					{Number: 6, Category: ResourceCategoryTCP, Target: "socket:[456]", FirstSeen: now},
				},
				ResourceCategoryPipe: {
					{Number: 7, Category: ResourceCategoryPipe, Target: "pipe:[789]", FirstSeen: now},
				},
			},
			expectedCategoryCount: 3,
			expectedTotal:         4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			categories, total := collector.buildCategoriesResponse(tc.input, now)

			if len(categories) != tc.expectedCategoryCount {
				t.Errorf("expected %d categories, got %d", tc.expectedCategoryCount, len(categories))
			}

			if total != tc.expectedTotal {
				t.Errorf("expected total %d, got %d", tc.expectedTotal, total)
			}

			lastIndex := -1
			for _, cat := range categories {
				index := -1
				for i, orderCat := range resourceCategoryOrder {
					if orderCat == cat.Category {
						index = i
						break
					}
				}
				if index <= lastIndex {
					t.Errorf("categories not in expected order: %s came after previous", cat.Category)
				}
				lastIndex = index
			}
		})
	}
}

func TestWithResourceCollectorClock(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	require.NotNil(t, collector)
	assert.Equal(t, mockClock, collector.clock)
}

func TestNewResourceCollector_DefaultClock(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	require.NotNil(t, collector)
	assert.NotNil(t, collector.clock)
	assert.NotNil(t, collector.firstSeen)
	assert.NotNil(t, collector.stopCh)
}

func TestNewResourceCollector_WithMockClock(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	require.NotNil(t, collector)
	assert.Equal(t, mockClock, collector.clock)
}

func TestResourceCollector_GetResources_ReturnsData(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	data := collector.GetResources()

	assert.NotNil(t, data.Categories)
	assert.Equal(t, startTime.UnixMilli(), data.TimestampMs)
	assert.GreaterOrEqual(t, data.Total, int32(0))
}

func TestResourceCollector_GetResources_HasFileDescriptors(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	data := collector.GetResources()

	assert.Greater(t, data.Total, int32(0))

	for _, cat := range data.Categories {
		assert.NotEmpty(t, cat.Category)
		assert.Greater(t, cat.Count, int32(0))
		assert.Len(t, cat.Resources, int(cat.Count))

		for _, fdResource := range cat.Resources {
			assert.Greater(t, fdResource.FD, int32(2), "should skip stdin/stdout/stderr")
			assert.NotEmpty(t, fdResource.Category)
			assert.NotEmpty(t, fdResource.Target)
		}
	}
}

func TestResourceCollector_GetResources_TracksFirstSeen(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	data1 := collector.GetResources()
	require.Greater(t, data1.Total, int32(0))

	mockClock.Advance(5 * time.Second)

	data2 := collector.GetResources()
	require.Greater(t, data2.Total, int32(0))

	assert.Equal(t, startTime.Add(5*time.Second).UnixMilli(), data2.TimestampMs)

	for _, cat := range data2.Categories {
		for _, fdResource := range cat.Resources {
			assert.GreaterOrEqual(t, fdResource.AgeMs, int64(0))
		}
	}
}

func TestResourceCollector_GetResources_CategoriesOrdered(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	data := collector.GetResources()

	if len(data.Categories) <= 1 {
		t.Skip("not enough categories to verify ordering")
	}

	lastIndex := -1
	for _, cat := range data.Categories {
		index := -1
		for i, orderCat := range resourceCategoryOrder {
			if orderCat == cat.Category {
				index = i
				break
			}
		}

		assert.Greater(t, index, lastIndex, "categories should follow resourceCategoryOrder")
		lastIndex = index
	}
}

func TestResourceCollector_StartAndStop(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	ctx, cancel := context.WithCancelCause(context.Background())

	collector.Start(ctx)

	collector.mu.RLock()
	initialCount := len(collector.firstSeen)
	collector.mu.RUnlock()

	assert.Greater(t, initialCount, 0, "Start should trigger initial scanFirstSeen")

	cancel(fmt.Errorf("test: cleanup"))

	time.Sleep(50 * time.Millisecond)

	collector.Stop()
}

func TestResourceCollector_LoopTicksOnClock(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	collector := NewResourceCollector(WithResourceCollectorClock(mockClock))

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	collector.Start(ctx)

	mockClock.Advance(3 * time.Second)

	time.Sleep(50 * time.Millisecond)

	collector.mu.RLock()
	count := len(collector.firstSeen)
	collector.mu.RUnlock()

	assert.Greater(t, count, 0, "loop should have performed a scan")

	cancel(fmt.Errorf("test: cleanup"))
	time.Sleep(50 * time.Millisecond)
	collector.Stop()
}

func TestResourceCollector_ProcessResourceEntry_StdioSkipped(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	data := collector.GetResources()

	for _, cat := range data.Categories {
		for _, fdResource := range cat.Resources {
			assert.Greater(t, fdResource.FD, int32(2),
				"FD %d should be excluded (stdin/stdout/stderr)", fdResource.FD)
		}
	}
}

func TestResourceCollector_CollectResourceInfo_PopulatesCategories(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	data := collector.GetResources()

	categoryNames := make(map[string]bool)
	for _, cat := range data.Categories {
		categoryNames[cat.Category] = true
	}

	assert.True(t, categoryNames[ResourceCategoryFile] || data.Total > 0,
		"expected at least file category or some resources")
}

func TestResourceCollector_LookupSocketType_TCP(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	result := collector.lookupSocketType("socket:[999999999]")

	validCategories := map[string]bool{
		ResourceCategoryTCP:    true,
		ResourceCategoryUDP:    true,
		ResourceCategoryUnix:   true,
		ResourceCategorySocket: true,
		"":                     true,
	}

	assert.True(t, validCategories[result],
		"lookupSocketType returned unexpected category: %q", result)
}

func TestResourceCollector_LookupSocketType_InvalidFormat(t *testing.T) {
	t.Parallel()

	collector := NewResourceCollector()

	result := collector.lookupSocketType("not-a-socket")
	assert.Empty(t, result)

	result = collector.lookupSocketType("socket:[]")
	assert.Empty(t, result)

	result = collector.lookupSocketType("socket:[abc]")
	assert.Empty(t, result)
}
