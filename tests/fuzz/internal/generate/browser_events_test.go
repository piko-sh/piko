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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
)

func TestEventsAppTopologyAndReplay(t *testing.T) {
	rotations := map[int]bool{}
	deltas := map[int]bool{}
	labels := map[string]bool{}
	for index := range uint64(12) {
		c := New("browser-events", 7, index)
		trace := c.Expected.Client.Events
		require.NotNil(t, trace, "index %d", index)
		require.Len(t, trace.States, 12)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash())
		rotations[trace.Input.Rotate] = true
		deltas[trace.Input.Delta] = true
		labels[trace.Input.Label] = true
		for _, file := range []string{"pages/events.pk", "components/fuzz-owner.pkc", "components/fuzz-emitter.pkc"} {
			document, err := sfcparser.Parse([]byte(c.Files[file]))
			require.NoError(t, err, file)
			tree, err := ast_domain.Parse(t.Context(), document.Template, file, nil)
			require.NoError(t, err, file)
			require.Empty(t, tree.Diagnostics, file)
			ast_domain.PutTree(tree)
		}
		for _, branch := range []string{"emitter", "owner"} {
			assert.Contains(t, c.Files["components/fuzz-"+branch+".pkc"], "@/lib/event-"+branch)
			assert.Contains(t, c.Files["lib/event-"+branch+".ts"], "./event-serial.js")
		}
		assert.True(t, slices.Contains(c.Features, "slot-reassignment"))
		assert.NotEmpty(t, c.Files["lib/event-serial.ts"])
	}
	assert.Len(t, rotations, 4)
	assert.GreaterOrEqual(t, len(deltas), 2)
	assert.GreaterOrEqual(t, len(labels), 2)
}
