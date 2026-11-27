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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestBrowserLifecycleAlternatesReleaseOrderAndLinkPlacement(t *testing.T) {
	variants := map[[2]bool]bool{}
	for index := range uint64(4) {
		c, err := Build("browser-lifecycle", 7, index, "")
		require.NoError(t, err)
		requireParseableCase(t, c)
		lifecycle := c.Expected.Client.Lifecycle
		require.NotNil(t, lifecycle, "index %d", index)
		assert.Equal(t, (7+index)%2 != 0, lifecycle.Input.Reverse)
		assert.Equal(t, (7+index)/2%2 != 0, lifecycle.Input.LayoutLinks)
		assert.False(t, lifecycle.Input.InlineScripts)
		assert.False(t, lifecycle.Input.ElementReuse)
		assert.NotEmpty(t, lifecycle.Input.Draft)
		assert.NotEmpty(t, lifecycle.States)
		variants[[2]bool{lifecycle.Input.Reverse, lifecycle.Input.LayoutLinks}] = true
		for _, file := range []string{"pages/journey.pk", "pages/journey-away.pk", "partials/journey-list.pk", "partials/journey-layout.pk", "components/fuzz-watch.pkc"} {
			assert.Contains(t, c.Files, file)
		}
		for _, page := range []string{"pages/journey.pk", "pages/journey-away.pk"} {
			assert.Contains(t, c.Files[page], `<piko:partial is="layout">`, page)
			assert.Contains(t, c.Files[page], `p-ref="bound"`, page)
			assert.Contains(t, c.Files[page], "function pageCounters()", page)
		}
		links := c.Files["partials/journey-layout.pk"] + c.Files["pages/journey.pk"] + c.Files["pages/journey-away.pk"]
		for _, link := range []string{`<piko:a href="/journey" id="link-journey">`, `<piko:a href="/journey-away" id="link-away">`} {
			assert.Equal(t, 1, strings.Count(links, link), link)
			assert.Equal(t, lifecycle.Input.LayoutLinks, strings.Contains(c.Files["partials/journey-layout.pk"], link), link)
		}
		assert.Contains(t, c.Features, "listener-cleanup")
		assert.Contains(t, c.Features, "page-script-lifecycle")
		matrices := Matrices(&c)
		require.Len(t, matrices, 1)
		assert.Equal(t, matrixBrowser, matrices[0].Environment)
		assert.Equal(t, 2, matrices[0].Feasible)
	}
	assert.Len(t, variants, 4)
}

func TestBrowserLifecycleInlineWritesPageScriptsWithoutFunctions(t *testing.T) {
	c, err := Build("browser-lifecycle-inline", 7, 0, "")
	require.NoError(t, err)
	requireParseableCase(t, c)
	lifecycle := c.Expected.Client.Lifecycle
	require.NotNil(t, lifecycle)
	assert.True(t, lifecycle.Input.InlineScripts)
	assert.Equal(t, casefile.ExpectedLifecycle(lifecycle.Input).States, lifecycle.States)
	for _, page := range []string{"pages/journey.pk", "pages/journey-away.pk"} {
		assert.Contains(t, c.Files[page], "pk.onConnected", page)
		assert.NotContains(t, c.Files[page], "function ", page)
	}
	assert.Contains(t, c.Features, "inline-page-script")
}

func TestBrowserElementReuseRendersStructurallyIdenticalListings(t *testing.T) {
	placements := map[bool]bool{}
	for index := range uint64(2) {
		c, err := Build("browser-element-reuse", 7, index, "")
		require.NoError(t, err)
		requireParseableCase(t, c)
		lifecycle := c.Expected.Client.Lifecycle
		require.NotNil(t, lifecycle)
		assert.True(t, lifecycle.Input.ElementReuse)
		placements[lifecycle.Input.LayoutLinks] = true
		first, second := c.Files["pages/reuse-orders.pk"], c.Files["pages/reuse-invoices.pk"]
		require.NotEmpty(t, first)
		require.NotEmpty(t, second)
		assert.Equal(t, strings.ReplaceAll(first, "orders", "invoices"), strings.ReplaceAll(second, "orders", "invoices"),
			"the listings differ in more than their name, so the morph might not reuse their elements")
		assert.Contains(t, first, `<listing-pager owner="orders" filter_input="#orders-filter">`)
		assert.Contains(t, c.Files, "components/listing-pager.pkc")
		assert.Contains(t, c.Features, "morph-element-reuse")
		assert.Empty(t, Matrices(&c))
	}
	assert.Len(t, placements, 2)
}
