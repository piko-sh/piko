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

//go:build crosslang

package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummariseHandlesEmptyInput(t *testing.T) {
	aggregate := Summarise("bench", RunnerPiko, ModeEndToEnd, nil, nil, nil, 0)
	assert.Equal(t, 0, aggregate.Runs)
	assert.Equal(t, int64(0), aggregate.MedianNanos)
}

func TestSummariseOddCountTakesMiddle(t *testing.T) {
	aggregate := Summarise("bench", RunnerPiko, ModeEndToEnd, []int64{10, 20, 30}, nil, nil, 0)
	assert.Equal(t, 3, aggregate.Runs)
	assert.Equal(t, int64(20), aggregate.MedianNanos)
	assert.Equal(t, int64(20), aggregate.MeanNanos)
	assert.Equal(t, int64(10), aggregate.MinNanos)
}

func TestSummariseEvenCountAveragesMiddleTwo(t *testing.T) {
	aggregate := Summarise("bench", RunnerPiko, ModeEndToEnd, []int64{10, 20, 30, 40}, nil, nil, 0)
	assert.Equal(t, int64(25), aggregate.MedianNanos)
}

func TestSummariseSinglePointStddevIsZero(t *testing.T) {
	aggregate := Summarise("bench", RunnerPiko, ModeEndToEnd, []int64{42}, nil, nil, 0)
	assert.Equal(t, int64(0), aggregate.StddevNanos)
}

func TestPercentileP95MatchesNearestRank(t *testing.T) {
	sorted := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.Equal(t, int64(10), percentileInt64(sorted, 0.95))
	assert.Equal(t, int64(5), percentileInt64(sorted, 0.5))
	assert.Equal(t, int64(1), percentileInt64(sorted, 0.0))
	assert.Equal(t, int64(10), percentileInt64(sorted, 1.0))
}

func TestSummariseSortsCopyDoesNotMutateInput(t *testing.T) {
	samples := []int64{40, 10, 30, 20}
	_ = Summarise("bench", RunnerPiko, ModeEndToEnd, samples, nil, nil, 0)
	assert.Equal(t, []int64{40, 10, 30, 20}, samples)
}

func TestSummariseRSSUsesMedianAcrossSamples(t *testing.T) {
	aggregate := Summarise("bench", RunnerPiko, ModeEndToEnd,
		[]int64{100, 200, 300},
		[]int64{1024, 2048, 4096},
		nil,
		0,
	)
	assert.Equal(t, int64(2048), aggregate.PeakRSSKB)
}
