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

package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

type namedSpamDetector struct {
	spamdetect_domain.Detector
	name string
}

func (d *namedSpamDetector) Name() string { return d.name }

func (*namedSpamDetector) Signals() []spamdetect_dto.Signal { return nil }

func (*namedSpamDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.DetectorPriority(0)
}

type stubFeedbackStore struct {
	spamdetect_domain.FeedbackStore
}

func TestGetSpamDetectService_DisabledByDefault(t *testing.T) {
	t.Parallel()

	service, err := NewContainer().GetSpamDetectService()

	require.NoError(t, err)
	assert.False(t, service.IsEnabled(context.Background()))
}

func TestGetSpamDetectService_BuildsBuiltInDetectorsFromConfig(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	c := NewContainer()
	c.serverConfig.Security.SpamDetectEnabled = new(true)
	c.serverConfig.Security.SpamDetectBlocklistPatterns = []string{"casino"}
	c.serverConfig.Security.SpamDetectScoreThreshold = new(0.75)
	c.SetSpamDetectFeedbackStore(&stubFeedbackStore{})

	service, err := c.GetSpamDetectService()

	require.NoError(t, err)
	assert.True(t, service.IsEnabled(context.Background()))
	assert.NotEmpty(t, service.GetDetectors(context.Background()))
}

func TestGetSpamDetectService_RejectsAnInvalidBlocklistPattern(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	c := NewContainer()
	c.serverConfig.Security.SpamDetectEnabled = new(true)
	c.serverConfig.Security.SpamDetectBlocklistPatterns = []string{"("}

	service, err := c.GetSpamDetectService()

	require.Error(t, err)
	assert.Nil(t, service)
}

func TestGetSpamDetectService_RegistersDetectorsFromOptions(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	c := NewContainer()
	c.AddSpamDetector("custom", &namedSpamDetector{name: "custom"})
	c.SetSpamDetectFeedbackStore(&stubFeedbackStore{})

	service, err := c.GetSpamDetectService()

	require.NoError(t, err)
	assert.Equal(t, []string{"custom"}, service.GetDetectors(context.Background()))
}

func TestGetSpamDetectService_RejectsANilDetector(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	c := NewContainer()
	c.AddSpamDetector("broken", nil)

	service, err := c.GetSpamDetectService()

	require.ErrorIs(t, err, spamdetect_dto.ErrDetectorNil)
	assert.Nil(t, service)
}

func TestSetSpamDetectService_TakesPrecedence(t *testing.T) {
	t.Parallel()

	override := spamdetect_domain.NewDisabledSpamDetectService()
	c := NewContainer()
	c.SetSpamDetectService(override)

	service, err := c.GetSpamDetectService()

	require.NoError(t, err)
	assert.Same(t, override, service)
}
