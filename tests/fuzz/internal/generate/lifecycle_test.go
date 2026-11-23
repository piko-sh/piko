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

func TestBrowserLifecycleAlternatesReleaseOrder(t *testing.T) {
	orders := map[bool]bool{}
	for index := range uint64(4) {
		c, err := Build("browser-lifecycle", 7, index, "")
		require.NoError(t, err)
		requireParseableCase(t, c)
		lifecycle := c.Expected.Client.Lifecycle
		require.NotNil(t, lifecycle, "index %d", index)
		assert.Equal(t, (7+index)%2 != 0, lifecycle.Input.Reverse)
		assert.NotEmpty(t, lifecycle.Input.Draft)
		assert.NotEmpty(t, lifecycle.States)
		orders[lifecycle.Input.Reverse] = true
		for _, file := range []string{"pages/journey.pk", "pages/journey-away.pk", "partials/journey-list.pk", "components/fuzz-watch.pkc"} {
			assert.Contains(t, c.Files, file)
		}
		assert.Contains(t, c.Features, "listener-cleanup")
		matrices := Matrices(&c)
		require.Len(t, matrices, 1)
		assert.Equal(t, matrixBrowser, matrices[0].Environment)
		assert.Equal(t, 2, matrices[0].Feasible)
	}
	assert.Len(t, orders, 2)
}
