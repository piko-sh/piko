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

package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestAddCaseCountsCompositionsAcrossFilesOnce(t *testing.T) {
	var population Population
	c := casefile.Case{Files: map[string]string{"pages/order.pk": "<template><form><input></form></template>", "components/notes.pkc": "<template name=\"notes-field\"><svg><path/></svg></template>", "cmd/main.go": "package main"}}
	require.NoError(t, population.AddCase(t.Context(), &c))
	assert.Equal(t, Frequency{Projects: 1}, population.Compositions["form + go + svg"])
	assert.Equal(t, Frequency{Files: 1, Projects: 1}, population.Compositions["pkc + svg"])
}

func TestCompositionDimensionsIgnoresMalformedFacts(t *testing.T) {
	dimensions := compositionDimensions(".pk", map[string]bool{"binding": true, "binding/tag": true, "tag/form": true})
	assert.Equal(t, map[string]bool{"pk": true, "form": true}, dimensions)
}

func TestIsSourceSelectsInventoriedTypes(t *testing.T) {
	assert.True(t, isSource("pages/main.pk"))
	assert.True(t, isSource("components/card.pkc"))
	assert.True(t, isSource("cmd/main.go"))
	assert.False(t, isSource("cmd/main_test.go"))
	assert.False(t, isSource("styles/site.css"))
}
