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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnalysisResult_ZeroValue(t *testing.T) {
	t.Parallel()
	var result AnalysisResult
	assert.Empty(t, result.SubmissionID)
	assert.Empty(t, result.DetectorResults)
	assert.Empty(t, result.FieldResults)
	assert.Empty(t, result.FormReasons)
	assert.Empty(t, result.PendingDetectors)
	assert.Empty(t, result.TruncatedFields)
	assert.Equal(t, time.Duration(0), result.Duration)
	assert.InDelta(t, 0.0, result.Score, 0.0001)
	assert.InDelta(t, 0.0, result.Threshold, 0.0001)
	assert.False(t, result.IsSpam)
	assert.False(t, result.PendingAsync)
	assert.False(t, result.Truncated)
}

func TestAnalysisResult_TruncatedFieldsSurfaced(t *testing.T) {
	t.Parallel()
	result := AnalysisResult{
		Truncated:       true,
		TruncatedFields: []string{"message"},
	}
	assert.True(t, result.Truncated)
	assert.Equal(t, []string{"message"}, result.TruncatedFields)
}

func TestDetectorResult_ErrorExcludesFromAggregation(t *testing.T) {
	t.Parallel()
	result := DetectorResult{
		Detector: "blocklist",
		Score:    0.5,
		Error:    errors.New("upstream failure"),
	}
	assert.Error(t, result.Error)
	assert.Equal(t, "blocklist", result.Detector)
}

func TestFieldResult_ZeroValue(t *testing.T) {
	t.Parallel()
	var result FieldResult
	assert.Empty(t, result.Key)
	assert.Equal(t, FieldType(""), result.Type)
	assert.Empty(t, result.Reasons)
	assert.InDelta(t, 0.0, result.Score, 0.0001)
}

func TestSubmissionRecord_ZeroValue(t *testing.T) {
	t.Parallel()
	var record SubmissionRecord
	assert.Nil(t, record.Submission)
	assert.Nil(t, record.Result)
	assert.True(t, record.ReportedAt.IsZero())
	assert.Empty(t, record.SubmissionID)
	assert.False(t, record.IsSpam)
}

func TestNewCleanDetectorResult(t *testing.T) {
	t.Parallel()
	result := NewCleanDetectorResult()
	assert.NotNil(t, result)
	assert.InDelta(t, 0.0, result.Score, 0.0001)
	assert.False(t, result.IsSpam)
	assert.Nil(t, result.FieldScores)
	assert.Nil(t, result.FieldReasons)
	assert.Nil(t, result.Reasons)
	assert.NoError(t, result.Error)
	assert.Empty(t, result.Detector)
	assert.Equal(t, time.Duration(0), result.Duration)
}

func TestNewFieldDetectorResult(t *testing.T) {
	t.Parallel()
	scores := map[string]float64{"message": 0.75}
	reasons := map[string][]string{"message": {"matched"}}
	result := NewFieldDetectorResult(0.75, true, scores, reasons)
	assert.InDelta(t, 0.75, result.Score, 0.0001)
	assert.True(t, result.IsSpam)
	assert.Equal(t, scores, result.FieldScores)
	assert.Equal(t, reasons, result.FieldReasons)
	assert.Nil(t, result.Reasons)
	assert.NoError(t, result.Error)
}

func TestNewReasonDetectorResult(t *testing.T) {
	t.Parallel()
	result := NewReasonDetectorResult(1.0, true, []string{"honeypot field was filled"})
	assert.InDelta(t, 1.0, result.Score, 0.0001)
	assert.True(t, result.IsSpam)
	assert.Equal(t, []string{"honeypot field was filled"}, result.Reasons)
	assert.Nil(t, result.FieldScores)
	assert.Nil(t, result.FieldReasons)
	assert.NoError(t, result.Error)
}
