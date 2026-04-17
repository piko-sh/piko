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

package spamdetect_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

func TestDisabledSpamDetectService_AllMethods(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	assert.False(t, service.IsEnabled(context.Background()))

	_, err := service.Analyse(context.Background(), &spamdetect_dto.Submission{}, &spamdetect_dto.Schema{})
	assert.ErrorIs(t, err, spamdetect_dto.ErrSpamDetectDisabled)

	assert.ErrorIs(t, service.RegisterDetector(context.Background(), "test", &mockDetector{}), spamdetect_dto.ErrSpamDetectDisabled)
	assert.Empty(t, service.GetDetectors(context.Background()))
	assert.False(t, service.HasDetector(context.Background(), "test"))
	assert.Empty(t, service.ListDetectors(context.Background()))
	assert.NoError(t, service.HealthCheck(context.Background()))
	assert.NoError(t, service.Close(context.Background()))
}

func TestDisabledSpamDetectService_FeedbackMethods(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	service.SetFeedbackStore(nil)
	assert.ErrorIs(t, service.ReportSpam(context.Background(), "id"), spamdetect_dto.ErrSpamDetectDisabled)
	assert.ErrorIs(t, service.ReportHam(context.Background(), "id"), spamdetect_dto.ErrSpamDetectDisabled)
}

func TestDisabledSpamDetectService_SetFeedbackStore(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	service.SetFeedbackStore(nil)
	service.SetFeedbackStore(&mockFeedbackStore{})
}
