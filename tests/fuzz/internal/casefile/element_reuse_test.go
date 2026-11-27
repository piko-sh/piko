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

package casefile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpectedElementReuseHandlesEachFilterChangeOnceWithTheShownListing(t *testing.T) {
	trace := ExpectedLifecycle(LifecycleInput{ElementReuse: true})

	assert.Equal(t, []string{"reuse-mount", "reuse-filter-orders", "reuse-link-invoices", "reuse-link-orders", "reuse-history-back"},
		lifecycleSteps(trace))
	last := trace.States[len(trace.States)-1]
	assert.Equal(t, "/reuse-invoices", last.Path)
	assert.Equal(t, "orders,invoices,orders,invoices", last.Applied)
	assert.Equal(t, 4, last.Deliveries)
	assert.Equal(t, 4, last.Connected)
	assert.Equal(t, 3, last.Cleaned)
	assert.Equal(t, PageScriptCounts{}, last.Journey)
	assert.True(t, last.SameDocument)
}
