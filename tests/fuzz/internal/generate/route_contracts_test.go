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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestRouteContractsModelEveryBulletinStage(t *testing.T) {
	c, err := Build("route-contracts", 7, 0, "")
	require.NoError(t, err)
	requireParseableCase(t, c)
	stages := map[string]casefile.Probe{}
	for _, probe := range c.Probes {
		if strings.HasPrefix(probe.Path, "/bulletins/") {
			stages[probe.Stage] = probe
		}
	}
	require.Len(t, stages, 5)
	published := stages["published"]
	require.NotNil(t, published.Semantic)
	assert.Equal(t, []string{"outer", "inner"}, published.ResponseHeaders[bulletinTrace])
	token := strings.TrimPrefix(published.Path, "/bulletins/")
	assert.True(t, strings.HasPrefix(token, "edition-"))

	moved := stages["middleware-stop"]
	assert.Equal(t, http.StatusSeeOther, moved.Status)
	assert.Equal(t, "/bulletins/featured", moved.Location)
	assert.Equal(t, published.Path+"?move=yes", moved.Path)

	assert.Contains(t, stages["literal-precedence"].ResponseHeaders, bulletinTrace)
	assert.Nil(t, stages["literal-precedence"].ResponseHeaders[bulletinTrace])
	assert.Equal(t, "/bulletins/archive/2026/"+token, stages["catch-all"].Path)

	archived := stages["typed-error"]
	assert.Equal(t, http.StatusForbidden, archived.Status)
	require.Len(t, archived.ByMode, 2)
	assert.Equal(t, http.StatusForbidden, archived.ByMode["compiled"].Status)
	assert.Contains(t, c.Files, "pages/!403.pk")
	assert.Contains(t, c.Files, "pages/bulletins/{slug}.pk")
}

func TestEditionErrorPolicyPrefixesInterpretedMessage(t *testing.T) {
	policy := editionErrorPolicy("edition-4")
	require.Len(t, policy, 2)
	message := func(mode string) string {
		t.Helper()
		elements := policy[mode].Semantic.Elements
		require.NotEmpty(t, elements, mode)
		return elements[len(elements)-1].Name
	}
	assert.Equal(t, "Archived edition-4", message("compiled"))
	assert.Equal(t, "renderPage: runPage failed for pages/bulletins/{slug}.pk: Archived edition-4", message("interp"))
	assert.Equal(t, []string{"noindex"}, policy["interp"].Semantic.Head["robots"])
}

func TestExpressionBoundariesSwapRowsOnReverse(t *testing.T) {
	c := casefile.Case{Files: map[string]string{}}
	addExpressionBoundaries(&c, Project{Count: 1})
	require.Len(t, c.Probes, 2)
	owners := func(probe casefile.Probe) []string {
		var result []string
		for _, output := range probe.Output {
			if output.ID == "template" {
				result = append(result, output.Text)
			}
		}
		return result
	}
	assert.Equal(t, []string{"edition zero: 0", "edition negative: -3"}, owners(c.Probes[0]))
	assert.Equal(t, []string{"edition negative: -3", "edition zero: 0"}, owners(c.Probes[1]))
	last := c.Probes[1].Output[len(c.Probes[1].Output)-1]
	assert.Equal(t, "skip", last.ID)
	assert.Equal(t, "omitted", last.Text)
	assert.Equal(t, "skipped", last.Value)
	assert.Contains(t, c.Files["pages/edition-expressions.pk"], "-3")
}
