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

package cache_dto

import (
	"time"

	"piko.sh/piko/wdk/safeconv"
)

// StatsRecorder defines a provider-agnostic interface for recording cache
// statistics. It tracks hits, misses, evictions, and load timing metrics.
type StatsRecorder interface {
	// RecordHits records the given number of cache hits.
	//
	// Takes count (uint64) which is the number of hits to record.
	RecordHits(count uint64)

	// RecordMisses records the given number of cache misses.
	//
	// Takes count (uint64) which is the number of cache misses to record.
	RecordMisses(count uint64)

	// RecordLoadSuccess records a successful load operation.
	//
	// Takes loadTime (time.Duration) which is how long the load took.
	RecordLoadSuccess(loadTime time.Duration)

	// RecordLoadFailure records a failed load operation with its duration.
	//
	// Takes loadTime (time.Duration) which is how long the load took before it
	// failed.
	RecordLoadFailure(loadTime time.Duration)

	// RecordEviction records that an item was removed from the cache.
	RecordEviction()
}

// Stats holds a snapshot of cache performance metrics at a point in time.
type Stats struct {
	// Hits is the number of successful cache lookups since the cache was created.
	Hits uint64

	// Misses is the number of times a key was not found in the cache.
	Misses uint64

	// Evictions is the number of items removed from the cache since it was created.
	Evictions uint64

	// LoadSuccessCount is the number of times a value was loaded successfully.
	LoadSuccessCount uint64

	// LoadFailureCount is the total number of times a load operation has failed.
	LoadFailureCount uint64

	// TotalLoadTime is the total time spent loading values into the cache.
	TotalLoadTime time.Duration
}

// HitRatio returns the ratio of cache hits to total requests (hits + misses).
//
// Returns float64 which is 0.0 if there have been no requests.
func (s Stats) HitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0.0
	}
	return float64(s.Hits) / float64(total)
}

// MissRatio returns the ratio of cache misses to total requests (hits + misses).
//
// Returns float64 which is 0.0 if there have been no requests.
func (s Stats) MissRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0.0
	}
	return float64(s.Misses) / float64(total)
}

// AverageLoadPenalty returns the average time spent loading entries.
//
// Returns time.Duration which is zero if no load operations have occurred.
func (s Stats) AverageLoadPenalty() time.Duration {
	totalLoads := s.LoadSuccessCount + s.LoadFailureCount
	if totalLoads == 0 {
		return 0
	}
	return s.TotalLoadTime / time.Duration(safeconv.Uint64ToInt64(totalLoads))
}
