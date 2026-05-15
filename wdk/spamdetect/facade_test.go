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

package spamdetect_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
	"piko.sh/piko/wdk/spamdetect"
)

func TestFacade_TypeAliases_AreInterchangeable(t *testing.T) {
	t.Parallel()

	schema := spamdetect.MustNewSchema(
		spamdetect.TextField("message", spamdetect.SignalGibberish),
	)
	require.NotNil(t, schema)

	submission := &spamdetect.Submission{}
	assert.IsType(t, &spamdetect_dto.Submission{}, submission)
}

func TestFacade_NewSchema_Errors(t *testing.T) {
	t.Parallel()
	entries := make([]spamdetect.SchemaEntry, 0)
	for index := range spamdetect_dto.MaxDetectorCount() + 10 {
		entries = append(entries, spamdetect.Language("lang_"+string(rune('a'+index%26))))
	}
	_, err := spamdetect.NewSchema(entries...)
	assert.Error(t, err)
}

func TestFacade_MustNewSchema_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		recovered := recover()
		assert.NotNil(t, recovered, "MustNewSchema must panic on invalid schema")
	}()
	entries := make([]spamdetect.SchemaEntry, 0)
	for index := range spamdetect_dto.MaxDetectorCount() + 10 {
		entries = append(entries, spamdetect.Language("lang_"+string(rune('a'+index%26))))
	}
	_ = spamdetect.MustNewSchema(entries...)
}

func TestFacade_Analyse_NilSubmission_ReturnsSafeError(t *testing.T) {
	t.Parallel()
	schema := spamdetect.MustNewSchema(spamdetect.TextField("message", spamdetect.SignalGibberish))
	_, err := spamdetect.Analyse(context.Background(), nil, schema)
	require.Error(t, err)
	assert.True(t, errors.Is(err, spamdetect_dto.ErrSubmissionNil))
	var safeErr safeerror.Error
	if assert.True(t, errors.As(err, &safeErr)) {
		assert.NotEmpty(t, safeErr.SafeMessage())
	}
}

func TestFacade_Analyse_NilSchema_ReturnsSafeError(t *testing.T) {
	t.Parallel()
	submission := &spamdetect.Submission{}
	_, err := spamdetect.Analyse(context.Background(), submission, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, spamdetect_dto.ErrSchemaNil))
}

func TestFacade_Sentinels_Reexported(t *testing.T) {
	t.Parallel()
	assert.Equal(t, spamdetect.ErrSpamDetectDisabled, spamdetect_dto.ErrSpamDetectDisabled)
	assert.Equal(t, spamdetect.ErrSpamDetected, spamdetect_dto.ErrSpamDetected)
	assert.Equal(t, spamdetect.ErrAllDetectorsFailed, spamdetect_dto.ErrAllDetectorsFailed)
	assert.Equal(t, spamdetect.ErrNoMatchingDetectors, spamdetect_dto.ErrNoMatchingDetectors)
	assert.Equal(t, spamdetect.ErrDetectorUnavailable, spamdetect_dto.ErrDetectorUnavailable)
}

func TestFacade_NewSpamDetectService_RoundTrip(t *testing.T) {
	t.Parallel()
	service, err := spamdetect.NewSpamDetectService(nil,
		spamdetect.WithScoreThreshold(0.5),
	)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.False(t, service.IsEnabled(context.Background()))
}

func TestFacade_DisabledService_RoundTrip(t *testing.T) {
	t.Parallel()
	service := spamdetect.NewDisabledSpamDetectService()
	require.NotNil(t, service)
	assert.False(t, service.IsEnabled(context.Background()))
	_, err := service.Analyse(context.Background(), &spamdetect.Submission{}, &spamdetect.Schema{})
	assert.ErrorIs(t, err, spamdetect.ErrSpamDetectDisabled)
}
