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

package pml_adapters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMediaQueryCollector(t *testing.T) {
	t.Parallel()

	collector := NewMediaQueryCollector()
	require.NotNil(t, collector)
}

func TestMediaQueryCollector_RegisterClass(t *testing.T) {
	t.Parallel()

	t.Run("registers a class successfully", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-50", "width: 100% !important; max-width: 100%;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "pml-col-50")
		assert.Contains(t, css, "width: 100% !important; max-width: 100%;")
	})

	t.Run("ignores empty class name", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("", "width: 100%;")
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("ignores empty mobile styles", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-50", "")
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("ignores both empty", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("", "")
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("deduplicates same class", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-50", "width: 100% !important;")
		collector.RegisterClass("pml-col-50", "width: 100% !important;")
		collector.RegisterClass("pml-col-50", "width: 100% !important;")
		css := collector.GenerateCSS("")
		count := strings.Count(css, "pml-col-50")
		assert.Equal(t, 1, count)
	})

	t.Run("last registration wins", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-50", "width: 50%;")
		collector.RegisterClass("pml-col-50", "width: 100% !important;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "width: 100% !important;")
		assert.NotContains(t, css, "width: 50%;")
	})
}

func TestMediaQueryCollector_RegisterFluidClass(t *testing.T) {
	t.Parallel()

	t.Run("registers a fluid class", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterFluidClass("pml-fluid-mobile", "width: 100% !important;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "pml-fluid-mobile")
	})

	t.Run("ignores empty class name", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterFluidClass("", "width: 100%;")
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("ignores empty styles", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterFluidClass("pml-fluid-mobile", "")
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("deduplicates fluid classes", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterFluidClass("pml-fluid-mobile", "width: 100% !important;")
		collector.RegisterFluidClass("pml-fluid-mobile", "width: 100% !important;")
		css := collector.GenerateCSS("")
		count := strings.Count(css, "pml-fluid-mobile")
		assert.Equal(t, 1, count)
	})
}

func TestMediaQueryCollector_GenerateCSS(t *testing.T) {
	t.Parallel()

	t.Run("empty collector returns empty string", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		css := collector.GenerateCSS("")
		assert.Empty(t, css)
	})

	t.Run("default breakpoint is 480px", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("test", "width: 100%;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "max-width: 480px")
	})

	t.Run("custom breakpoint", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("test", "width: 100%;")
		css := collector.GenerateCSS("600px")
		assert.Contains(t, css, "max-width: 600px")
	})

	t.Run("classes are sorted alphabetically", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("zebra", "width: 100%;")
		collector.RegisterClass("alpha", "width: 100%;")
		collector.RegisterClass("middle", "width: 100%;")
		css := collector.GenerateCSS("")

		alphaIndex := strings.Index(css, "alpha")
		middleIndex := strings.Index(css, "middle")
		zebraIndex := strings.Index(css, "zebra")
		assert.True(t, alphaIndex < middleIndex && middleIndex < zebraIndex)
	})

	t.Run("classes get dot prefix", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("my-class", "width: 100%;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, ".my-class")
	})

	t.Run("classes with dot are not re-prefixed", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("table.full-width", "width: 100%;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "table.full-width { width: 100%; }")
		assert.NotContains(t, css, ".table.full-width")
	})

	t.Run("mixed classes and fluid classes together", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-50", "width: 100% !important;")
		collector.RegisterFluidClass("pml-fluid-mobile", "width: 100% !important;")
		css := collector.GenerateCSS("")
		assert.Contains(t, css, "pml-col-50")
		assert.Contains(t, css, "pml-fluid-mobile")
		assert.Contains(t, css, "@media only screen")
		assert.True(t, strings.HasSuffix(css, "}"))
	})

	t.Run("media query wrapper structure", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("test-class", "display: block;")
		css := collector.GenerateCSS("480px")
		assert.True(t, strings.HasPrefix(css, "@media only screen and (max-width: 480px) {\n"))
		assert.True(t, strings.HasSuffix(css, "}"))
	})

	t.Run("multiple classes produce correct CSS", func(t *testing.T) {
		t.Parallel()
		collector := NewMediaQueryCollector()
		collector.RegisterClass("pml-col-25", "width: 25% !important;")
		collector.RegisterClass("pml-col-50", "width: 50% !important;")
		collector.RegisterClass("pml-col-100", "width: 100% !important;")
		css := collector.GenerateCSS("")

		assert.Contains(t, css, ".pml-col-25 { width: 25% !important; }")
		assert.Contains(t, css, ".pml-col-50 { width: 50% !important; }")
		assert.Contains(t, css, ".pml-col-100 { width: 100% !important; }")
	})
}

func TestMSOConditionalCollector_LastRegistrationWins(t *testing.T) {
	t.Parallel()

	collector := NewMSOConditionalCollector()
	collector.RegisterStyle("ul", "margin: 10px;")
	collector.RegisterStyle("ul", "margin: 0 !important;")
	result := collector.GenerateConditionalBlock()
	assert.Contains(t, result, "margin: 0 !important;")
	assert.NotContains(t, result, "margin: 10px;")
}

func TestMSOConditionalCollector_StructuralFormat(t *testing.T) {
	t.Parallel()

	collector := NewMSOConditionalCollector()
	collector.RegisterStyle("td", "padding: 0;")
	result := collector.GenerateConditionalBlock()

	assert.True(t, strings.HasPrefix(result, "<!--[if mso]>\n<style type=\"text/css\">\n"))
	assert.True(t, strings.HasSuffix(result, "</style>\n<![endif]-->"))
	assert.Contains(t, result, "  td {padding: 0;}")
}
