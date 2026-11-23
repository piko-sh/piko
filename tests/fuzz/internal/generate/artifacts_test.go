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

package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedArtifactsHaveDistinctRetainedOracles(t *testing.T) {
	c := New("generated-artifacts", 7, 0)
	seen := map[string]string{}
	for _, probe := range c.Probes {
		if probe.Path != "/artifact" && probe.Path != "/unaffected" {
			continue
		}
		require.True(t, probe.Program, probe.Path)
		require.Len(t, probe.Output, 1, probe.Path)
		require.Contains(t, c.Files["pages"+probe.Path+".pk"], probe.Output[0].Text)
		seen[probe.Path] = probe.Output[0].Text
	}
	require.Len(t, seen, 2)
	assert.NotEqual(t, seen["/artifact"], seen["/unaffected"])
	assert.Contains(t, c.Files["actions/contract/echo.go"], "EchoAction")
}
