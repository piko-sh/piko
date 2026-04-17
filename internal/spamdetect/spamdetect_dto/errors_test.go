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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinels_AreDistinct(t *testing.T) {
	t.Parallel()
	sentinels := []error{
		ErrSpamDetectDisabled,
		ErrSpamDetected,
		ErrAllDetectorsFailed,
		ErrNoMatchingDetectors,
		ErrDetectorUnavailable,
		ErrSubmissionNil,
		ErrSchemaNil,
		ErrDetectorNameEmpty,
		ErrDetectorNil,
		ErrTooManyDetectors,
		ErrSchemaTooManyFields,
		ErrSchemaDuplicateField,
		ErrSchemaInvalidThreshold,
		ErrSchemaCapExceeded,
		ErrBlocklistTooLarge,
		ErrBlocklistPatternInvalid,
		ErrBlocklistPatternTooLong,
		ErrUnexpectedDetectorResponse,
		ErrSubmissionIDGeneration,
	}
	for index, candidate := range sentinels {
		assert.NotNil(t, candidate)
		require.NotEmpty(t, candidate.Error())
		for otherIndex, other := range sentinels {
			if index == otherIndex {
				continue
			}
			assert.NotEqual(t, candidate.Error(), other.Error(),
				"sentinel %d duplicates sentinel %d", index, otherIndex)
		}
	}
}

func TestNewSpamDetectError_WrapsCause(t *testing.T) {
	t.Parallel()
	underlying := errors.New("inner")
	wrapped := NewSpamDetectError("analyse", "honeypot", underlying)

	assert.True(t, errors.Is(wrapped, underlying))
	assert.Equal(t, "analyse", wrapped.Operation)
	assert.Equal(t, "honeypot", wrapped.Detector)
	assert.Equal(t, underlying, wrapped.Unwrap())
}

func TestSpamDetectError_ErrorString(t *testing.T) {
	t.Parallel()
	underlying := errors.New("compute failed")
	wrapped := NewSpamDetectError("analyse", "blocklist", underlying)

	message := wrapped.Error()
	assert.True(t, strings.Contains(message, "analyse"))
	assert.True(t, strings.Contains(message, "blocklist"))
	assert.True(t, strings.Contains(message, "compute failed"))
}

func TestSpamDetectError_AsExtraction(t *testing.T) {
	t.Parallel()
	wrapped := NewSpamDetectError("feedback", "timing", errors.New("upstream gone"))

	var target *SpamDetectError
	require.True(t, errors.As(wrapped, &target))
	assert.Equal(t, "feedback", target.Operation)
	assert.Equal(t, "timing", target.Detector)
}
