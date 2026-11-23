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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestSourceEquivalenceRoutesAgreeExceptTheChangedRewrite(t *testing.T) {
	c, err := Build("source-equivalence", 7, 2, "")
	require.NoError(t, err)
	requireParseableCase(t, c)
	stages := equivalenceNumbers(t, c.Probes)
	require.Len(t, stages, 5)
	base := stages["rewrite-base"]
	for _, variant := range []string{"aliased", "long", "split"} {
		assert.Equal(t, base, stages["rewrite-"+variant], variant)
	}
	assert.Equal(t, base+1, stages["rewrite-changed"])
	assert.Contains(t, c.Files, "model/equivalence/scale.go")
	assert.Contains(t, c.Files, "pages/guards.pk")

	guards := map[string][2]string{}
	for _, probe := range c.Probes {
		if len(probe.Output) == 2 && probe.Output[0].ID == "enabled" {
			guards[probe.Path] = [2]string{probe.Output[0].Text, probe.Output[1].Text}
		}
	}
	assert.Equal(t, map[string][2]string{
		"/guards?case=none":     {"absent", "yes"},
		"/guards?case=present":  {"active", "no"},
		"/guards?case=disabled": {"absent", "yes"},
	}, guards)
	assert.Contains(t, c.Features, "short-circuit-guard")
}

func TestWasmEquivalenceOmitsTheSplitPackage(t *testing.T) {
	c, err := Build("wasm", 7, 2, "")
	require.NoError(t, err)
	stages := equivalenceNumbers(t, c.Probes)
	assert.Len(t, stages, 4)
	assert.NotContains(t, stages, "rewrite-split")
	assert.NotContains(t, c.Files, "model/equivalence/scale.go")
	assert.NotContains(t, c.Files, "pages/guards.pk")
	for _, probe := range c.Probes {
		if strings.HasPrefix(probe.Stage, "rewrite-") {
			assert.False(t, probe.BrowserProgram, probe.Stage)
		}
	}
}

func equivalenceNumbers(t *testing.T, probes []casefile.Probe) map[string]int {
	t.Helper()
	numbers := map[string]int{}
	for _, probe := range probes {
		if len(probe.Output) == 0 || probe.Output[0].ID != "number" {
			continue
		}
		number, err := strconv.Atoi(probe.Output[0].Text)
		require.NoError(t, err, probe.Stage)
		numbers[probe.Stage] = number
	}
	return numbers
}
