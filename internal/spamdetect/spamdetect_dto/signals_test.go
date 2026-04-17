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

	"github.com/stretchr/testify/assert"
)

func TestSignal_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		signal Signal
		want   string
	}{
		{SignalGibberish, "gibberish"},
		{SignalLinkDensity, "link_density"},
		{SignalBlocklist, "blocklist"},
		{SignalHoneypot, "honeypot"},
		{SignalTiming, "timing"},
		{SignalRepetition, "repetition"},
		{Signal("custom"), "custom"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.signal.String())
	}
}

func TestPriority_Ordering(t *testing.T) {
	t.Parallel()
	assert.Less(t, int(PriorityCritical), int(PriorityHigh))
	assert.Less(t, int(PriorityHigh), int(PriorityNormal))
}

func TestDetectorMode_Values(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DetectorMode(0), DetectorModeSync)
	assert.Equal(t, DetectorMode(1), DetectorModeAsync)
	assert.NotEqual(t, DetectorModeSync, DetectorModeAsync)
}
