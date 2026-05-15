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

package spamdetect_provider_builtin_detectors_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/spamdetect/spamdetect_provider_builtin_detectors"
)

func TestFacade_NewHoneypotDetector(t *testing.T) {
	t.Parallel()
	detector := spamdetect_provider_builtin_detectors.NewHoneypotDetector()
	require.NotNil(t, detector)
	assert.Equal(t, "honeypot", detector.Name())
}

func TestFacade_NewGibberishDetector(t *testing.T) {
	t.Parallel()
	detector := spamdetect_provider_builtin_detectors.NewGibberishDetector(0.6, nil)
	require.NotNil(t, detector)
	assert.Equal(t, "gibberish", detector.Name())
}

func TestFacade_NewLinkDensityDetector(t *testing.T) {
	t.Parallel()
	detector := spamdetect_provider_builtin_detectors.NewLinkDensityDetector(3)
	require.NotNil(t, detector)
	assert.Equal(t, "link_density", detector.Name())
}

func TestFacade_NewBlocklistDetector_ValidPatterns(t *testing.T) {
	t.Parallel()
	detector, err := spamdetect_provider_builtin_detectors.NewBlocklistDetector([]string{`foo`, `bar`})
	require.NoError(t, err)
	require.NotNil(t, detector)
	assert.Equal(t, "blocklist", detector.Name())
}

func TestFacade_NewBlocklistDetector_InvalidPattern(t *testing.T) {
	t.Parallel()
	_, err := spamdetect_provider_builtin_detectors.NewBlocklistDetector([]string{`[invalid`})
	assert.Error(t, err)
}

func TestFacade_NewTimingDetector(t *testing.T) {
	t.Parallel()
	detector := spamdetect_provider_builtin_detectors.NewTimingDetector(2 * time.Second)
	require.NotNil(t, detector)
	assert.Equal(t, "timing", detector.Name())
}

func TestFacade_NewRepetitionDetector_NilCache(t *testing.T) {
	t.Parallel()
	detector := spamdetect_provider_builtin_detectors.NewRepetitionDetector(nil, time.Minute, true)
	require.NotNil(t, detector)
	assert.Equal(t, "repetition", detector.Name())
}
