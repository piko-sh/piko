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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchReceiptSurvivesSplitAndOversizedLogLines(t *testing.T) {
	var monitor catalogueWatchLog
	monitor.edit("edit-card", "edited")
	for _, chunk := range []string{
		strings.Repeat("unrelated", 1024),
		"\n" + `msg="Received file event" path=/app/partials/catalog/`,
		"card.pk\n" + `msg="Starting targeted rebuild" changed=partials/catalog/card.pk affected_count=10` + "\n" + `msg="Targeted rebuild complete" changed=partials/catalog/card.pk` + "\n",
	} {
		written, err := monitor.Write([]byte(chunk))
		require.NoError(t, err)
		require.Equal(t, len(chunk), written)
	}
	events := monitor.snapshot()
	require.Len(t, events, 4, "lost watcher evidence: %q", events)
	assert.Contains(t, events[0], "sha256=")
	assert.Contains(t, events[1], catalogueLeaf)
	assert.Contains(t, events[2], "affected_count=10")
	assert.Contains(t, events[3], "complete")
}
