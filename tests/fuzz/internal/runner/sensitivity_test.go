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

package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestSensitivityDistinguishesControlsAndIndependentCorruptions(t *testing.T) {
	for _, index := range []uint64{0, 1, 7, 64} {
		testCase := generate.New("oracle-sensitivity", 7, index)
		trials, err := MeasureSensitivity(&testCase)
		require.NoError(t, err, "%d: %+v", index, trials)
		names := map[string]bool{}
		var controls, kills int
		for _, mutation := range trials {
			require.False(t, names[mutation.Name], "duplicate mutation %+v", mutation)
			require.True(t, mutation.Passed, "invalid mutation %+v", mutation)
			require.NotEqual(t, mutation.Control, mutation.Detected, "invalid mutation %+v", mutation)
			names[mutation.Name] = true
			if mutation.Control {
				controls++
			} else {
				kills++
			}
		}
		assert.Equal(t, 2, controls, "index %d clean controls", index)
		assert.Equal(t, 15, kills, "index %d kills", index)
		testCase.Probes = nil
		_, err = MeasureSensitivity(&testCase)
		assert.Error(t, err, "missing saved model passed")
	}
}
