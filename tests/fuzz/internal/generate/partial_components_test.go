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

func TestPartialComponentRecipe(t *testing.T) {
	variants := map[string]bool{}
	for index := range uint64(4) {
		c := New("partial-components", 8, index)
		trace := c.Expected.Client.PartialComponents
		require.Len(t, trace.States, 16)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash())
		require.True(t, c.RequiresProjectInterpreterHost(), "interpreted mode omitted the native action")
		require.Contains(t, c.Files["partials/record-rows.pk"], `<record-chip :recordkey="row.ID" :label="row.Label" :score="row.Score">`)
		require.GreaterOrEqual(t, len(trace.States[1].Log), 2)
		variants[trace.Input.Locale+trace.States[1].Log[:2]] = true
	}
	assert.Len(t, variants, 4)
}
