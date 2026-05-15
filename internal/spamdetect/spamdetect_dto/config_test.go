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

package spamdetect_dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultServiceConfig_Defaults(t *testing.T) {
	t.Parallel()
	config := DefaultServiceConfig()
	assert.InDelta(t, defaultScoreThreshold, config.ScoreThreshold, 0.0001)
	assert.Equal(t, defaultTimeout, config.Timeout)
	assert.Equal(t, DefaultFeedbackCacheSize, config.FeedbackCacheSize)
	assert.Nil(t, config.DetectorWeights)
}

func TestDefaultServiceConfig_Independent(t *testing.T) {
	t.Parallel()
	first := DefaultServiceConfig()
	second := DefaultServiceConfig()
	first.ScoreThreshold = 0.9
	first.Timeout = time.Second
	first.FeedbackCacheSize = 1
	assert.InDelta(t, defaultScoreThreshold, second.ScoreThreshold, 0.0001)
	assert.Equal(t, defaultTimeout, second.Timeout)
	assert.Equal(t, DefaultFeedbackCacheSize, second.FeedbackCacheSize)
}

func TestMaxDetectorCount_Constant(t *testing.T) {
	t.Parallel()
	assert.Equal(t, maxDetectorCount, MaxDetectorCount())
	assert.Greater(t, MaxDetectorCount(), 0)
}

func TestMaxFeedbackCacheSize_Constant(t *testing.T) {
	t.Parallel()
	assert.Equal(t, maxFeedbackCacheSize, MaxFeedbackCacheSize())
	assert.Greater(t, MaxFeedbackCacheSize(), DefaultFeedbackCacheSize)
}
