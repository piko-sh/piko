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

func TestMarkupJourneyMeasuresFourFamilies(t *testing.T) {
	for _, target := range []string{"markup", "firefox-markup"} {
		c, err := Build(target, 7, 1, "")
		require.NoError(t, err, target)
		plan := c.Expected.DOM
		require.NotNil(t, plan, target)
		assert.Equal(t, "/markup", plan.Path)
		assert.Equal(t, boardHost, plan.ReadyHost)
		assert.Contains(t, c.Expected.AdditionalComponents, boardHost)
		assert.Contains(t, c.Files["pages/main.pk"], "<markup-board hidden></markup-board>")
		page := c.Files["pages/markup.pk"]
		assert.Contains(t, page, `<h1 data-fuzz-element="title">`)
		assert.Contains(t, page, "/_piko/assets/"+casefile.Module+"/lib/markup-icon.svg")
		assert.NotContains(t, page, "@/lib/")

		var families *casefile.DOMStep
		for index := range plan.Steps {
			if plan.Steps[index].ID == "content-models" {
				families = &plan.Steps[index]
			}
		}
		require.NotNil(t, families, target)
		matrices := Matrices(&c)
		require.Len(t, matrices, 1)
		assert.Len(t, matrices[0].Cells, 4)
		assert.Equal(t, len(markupFamilies), matrices[0].Feasible)
		for _, family := range matrices[0].Cells {
			assert.Contains(t, page, `id="leaf-`+family+`"`)
			assert.Contains(t, c.Features, "html-family-"+family)
		}
		for _, field := range families.Fields {
			assert.Empty(t, field.Query.Host, field.ID)
		}
	}
}

func TestMarkupFamiliesReadMultimediaDefaults(t *testing.T) {
	c := casefile.Case{Files: map[string]string{"pages/markup.pk": "<article></article>"}, Expected: casefile.Expected{DOM: &casefile.DOMPlan{}}, Index: 3, Seed: 2}
	addMarkupFamilies(&c, "  spaced   title ")
	require.Len(t, c.Expected.DOM.Steps, 1)
	fields := c.Expected.DOM.Steps[0].Fields
	var media bool
	for _, field := range fields {
		if field.ID == "media-defaults" {
			media = true
			assert.Equal(t, []string{"true/true/none", "true/false/none"}, field.Want)
		}
		if field.ID == "multimedia-text" {
			assert.Equal(t, []string{"Case multimedia spaced title"}, field.Want)
		}
	}
	assert.True(t, media)
	assert.True(t, strings.HasSuffix(c.Files["pages/markup.pk"], "</article>"))
}

func TestMarkupTableSwapsEveryReadWhenReversed(t *testing.T) {
	forward := markupTable("One", "Two", 3, false)
	reversed := markupTable("One", "Two", 3, true)
	require.Len(t, reversed, len(forward))
	byID := func(fields []casefile.DOMField) map[string][]string {
		values := map[string][]string{}
		for _, field := range fields {
			values[field.ID] = field.Want
		}
		return values
	}
	want, got := byID(forward), byID(reversed)
	assert.Equal(t, []string{"One", "Two"}, want["names"])
	assert.Equal(t, []string{"Two", "One"}, got["names"])
	assert.Equal(t, []string{"4", "3"}, got["quantities"])
	assert.Equal(t, []string{"b", "a"}, got["row-keys"])
	assert.Equal(t, want["rows"], got["rows"])
}
