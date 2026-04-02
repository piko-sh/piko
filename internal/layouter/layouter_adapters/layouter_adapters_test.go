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

package layouter_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestFontKey_Equality(t *testing.T) {
	t.Parallel()

	t.Run("identical keys are equal", func(t *testing.T) {
		t.Parallel()

		keyA := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}
		keyB := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}

		assert.Equal(t, keyA, keyB)
	})

	t.Run("different family makes keys unequal", func(t *testing.T) {
		t.Parallel()

		keyA := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}
		keyB := fontKey{family: "Helvetica", weight: 400, style: layouter_domain.FontStyleNormal}

		assert.NotEqual(t, keyA, keyB)
	})

	t.Run("different weight makes keys unequal", func(t *testing.T) {
		t.Parallel()

		keyA := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}
		keyB := fontKey{family: "Arial", weight: 700, style: layouter_domain.FontStyleNormal}

		assert.NotEqual(t, keyA, keyB)
	})

	t.Run("different style makes keys unequal", func(t *testing.T) {
		t.Parallel()

		keyA := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}
		keyB := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleItalic}

		assert.NotEqual(t, keyA, keyB)
	})
}

func TestFontKey_AsMapKey(t *testing.T) {
	t.Parallel()

	keyMap := make(map[fontKey]string)

	keyA := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}
	keyB := fontKey{family: "Arial", weight: 700, style: layouter_domain.FontStyleNormal}
	keyC := fontKey{family: "Arial", weight: 400, style: layouter_domain.FontStyleNormal}

	keyMap[keyA] = "normal"
	keyMap[keyB] = "bold"

	assert.Equal(t, "normal", keyMap[keyC])
	assert.Equal(t, "bold", keyMap[keyB])
	assert.Len(t, keyMap, 2)
}

func TestMockFontMetrics_MeasureText_Default(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{}
	font := layouter_domain.FontDescriptor{Family: "Arial", Weight: 400}

	result := mock.MeasureText(font, 12.0, "Hello", layouter_domain.DirectionLTR)

	assert.InDelta(t, 5*12.0*mockCharacterWidthRatio, result, 0.001)
}

func TestMockFontMetrics_MeasureText_CustomFunc(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{
		MeasureTextFunc: func(_ layouter_domain.FontDescriptor, _ float64, _ string) float64 {
			return 42.0
		},
	}

	result := mock.MeasureText(layouter_domain.FontDescriptor{}, 12.0, "test", layouter_domain.DirectionLTR)

	assert.Equal(t, 42.0, result)
}

func TestMockFontMetrics_ShapeText_Default(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{}
	font := layouter_domain.FontDescriptor{Family: "Arial", Weight: 400}

	result := mock.ShapeText(font, 10.0, "Hi", layouter_domain.DirectionLTR)

	require.Len(t, result, 2)
	assert.InDelta(t, 10.0*mockCharacterWidthRatio, result[0].XAdvance, 0.001)
	assert.Equal(t, 0, result[0].ClusterIndex)
	assert.Equal(t, 1, result[0].RuneCount)
	assert.Equal(t, 1, result[1].ClusterIndex)
}

func TestMockFontMetrics_ShapeText_CustomFunc(t *testing.T) {
	t.Parallel()

	customGlyphs := []layouter_domain.GlyphPosition{{GlyphID: 99, XAdvance: 7.5}}
	mock := &MockFontMetrics{
		ShapeTextFunc: func(_ layouter_domain.FontDescriptor, _ float64, _ string) []layouter_domain.GlyphPosition {
			return customGlyphs
		},
	}

	result := mock.ShapeText(layouter_domain.FontDescriptor{}, 10.0, "A", layouter_domain.DirectionLTR)

	require.Len(t, result, 1)
	assert.Equal(t, uint16(99), result[0].GlyphID)
}

func TestMockFontMetrics_GetMetrics_Default(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{}
	font := layouter_domain.FontDescriptor{Family: "Arial", Weight: 400}

	metrics := mock.GetMetrics(font, 10.0)

	assert.InDelta(t, 10.0*mockAscentRatio, metrics.Ascent, 0.001)
	assert.InDelta(t, 10.0*mockDescentRatio, metrics.Descent, 0.001)
	assert.Equal(t, 0.0, metrics.LineGap)
	assert.InDelta(t, 10.0*mockCapHeightRatio, metrics.CapHeight, 0.001)
	assert.InDelta(t, 10.0*mockXHeightRatio, metrics.XHeight, 0.001)
	assert.Equal(t, mockUnitsPerEm, metrics.UnitsPerEm)
}

func TestMockFontMetrics_GetMetrics_CustomFunc(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{
		GetMetricsFunc: func(_ layouter_domain.FontDescriptor, _ float64) layouter_domain.FontMetrics {
			return layouter_domain.FontMetrics{Ascent: 99.0}
		},
	}

	metrics := mock.GetMetrics(layouter_domain.FontDescriptor{}, 10.0)

	assert.Equal(t, 99.0, metrics.Ascent)
}

func TestMockFontMetrics_ResolveFallback_Default(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{}
	font := layouter_domain.FontDescriptor{Family: "Arial", Weight: 400}

	result := mock.ResolveFallback(font, 'A')

	assert.Equal(t, font, result)
}

func TestMockFontMetrics_ResolveFallback_CustomFunc(t *testing.T) {
	t.Parallel()

	fallbackFont := layouter_domain.FontDescriptor{Family: "Fallback", Weight: 400}
	mock := &MockFontMetrics{
		ResolveFallbackFunc: func(_ layouter_domain.FontDescriptor, _ rune) layouter_domain.FontDescriptor {
			return fallbackFont
		},
	}

	result := mock.ResolveFallback(layouter_domain.FontDescriptor{Family: "Arial"}, 'A')

	assert.Equal(t, fallbackFont, result)
}

func TestMockFontMetrics_SplitGraphemeClusters(t *testing.T) {
	t.Parallel()

	mock := &MockFontMetrics{}

	t.Run("empty string returns empty slice", func(t *testing.T) {
		t.Parallel()

		result := mock.SplitGraphemeClusters("")

		assert.Empty(t, result)
	})

	t.Run("ASCII text splits into single-rune clusters", func(t *testing.T) {
		t.Parallel()

		result := mock.SplitGraphemeClusters("abc")

		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
}

func TestMockFontMetrics_ImplementsPort(t *testing.T) {
	t.Parallel()

	var _ layouter_domain.FontMetricsPort = (*MockFontMetrics)(nil)
}

func TestMockImageResolver_Default(t *testing.T) {
	t.Parallel()

	mock := &MockImageResolver{}

	width, height, err := mock.GetImageDimensions(context.Background(), "test.png")

	require.NoError(t, err)
	assert.Equal(t, mockImageWidth, width)
	assert.Equal(t, mockImageHeight, height)
}

func TestMockImageResolver_CustomFunc(t *testing.T) {
	t.Parallel()

	mock := &MockImageResolver{
		GetImageDimensionsFunc: func(_ context.Context, source string) (float64, float64, error) {
			if source == "wide.png" {
				return 200.0, 50.0, nil
			}
			return 0, 0, errors.New("not found")
		},
	}

	t.Run("custom func returns custom dimensions", func(t *testing.T) {
		t.Parallel()

		width, height, err := mock.GetImageDimensions(context.Background(), "wide.png")

		require.NoError(t, err)
		assert.Equal(t, 200.0, width)
		assert.Equal(t, 50.0, height)
	})

	t.Run("custom func returns error", func(t *testing.T) {
		t.Parallel()

		_, _, err := mock.GetImageDimensions(context.Background(), "missing.png")

		assert.Error(t, err)
	})
}

func TestMockImageResolver_ImplementsPort(t *testing.T) {
	t.Parallel()

	var _ layouter_domain.ImageResolverPort = (*MockImageResolver)(nil)
}

func TestNewCSSResolutionAdapter_DefaultRootFontSize(t *testing.T) {
	t.Parallel()

	adapter := NewCSSResolutionAdapter(0)

	assert.InDelta(t, defaultRootFontSize, adapter.rootFontSize, 0.001)
}

func TestNewCSSResolutionAdapter_NegativeRootFontSize(t *testing.T) {
	t.Parallel()

	adapter := NewCSSResolutionAdapter(-5.0)

	assert.InDelta(t, defaultRootFontSize, adapter.rootFontSize, 0.001)
}

func TestNewCSSResolutionAdapter_CustomRootFontSize(t *testing.T) {
	t.Parallel()

	adapter := NewCSSResolutionAdapter(16.0)

	assert.InDelta(t, 16.0, adapter.rootFontSize, 0.001)
}

func TestCSSResolutionAdapter_SetViewportDimensions(t *testing.T) {
	t.Parallel()

	adapter := NewCSSResolutionAdapter(12.0)
	adapter.SetViewportDimensions(800.0, 600.0)

	assert.InDelta(t, 800.0, adapter.viewportWidth, 0.001)
	assert.InDelta(t, 600.0, adapter.viewportHeight, 0.001)
}

func TestCSSResolutionAdapter_ImplementsPort(t *testing.T) {
	t.Parallel()

	var _ layouter_domain.StylesheetPort = (*CSSResolutionAdapter)(nil)
}

func TestParsePseudoType(t *testing.T) {
	t.Parallel()

	t.Run("before", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, layouter_domain.PseudoBefore, parsePseudoType("before"))
	})

	t.Run("after", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, layouter_domain.PseudoAfter, parsePseudoType("after"))
	})

	t.Run("unknown returns PseudoNone", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, layouter_domain.PseudoNone, parsePseudoType("unknown"))
	})

	t.Run("empty returns PseudoNone", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, layouter_domain.PseudoNone, parsePseudoType(""))
	})
}

func TestGetAttributeValue(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing attribute", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "main"},
				{Name: "id", Value: "header"},
			},
		}

		assert.Equal(t, "main", getAttributeValue(node, "class"))
		assert.Equal(t, "header", getAttributeValue(node, "id"))
	})

	t.Run("returns empty for missing attribute", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "main"},
			},
		}

		assert.Equal(t, "", getAttributeValue(node, "style"))
	})

	t.Run("returns empty for node with no attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{}

		assert.Equal(t, "", getAttributeValue(node, "class"))
	})
}

func TestNormaliseDimensionValue(t *testing.T) {
	t.Parallel()

	t.Run("bare number gets px suffix", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "100px", normaliseDimensionValue("100"))
	})

	t.Run("percentage stays as-is", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "50%", normaliseDimensionValue("50%"))
	})

	t.Run("non-numeric value stays as-is", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "auto", normaliseDimensionValue("auto"))
	})

	t.Run("float number gets px suffix", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "12.5px", normaliseDimensionValue("12.5"))
	})
}

func TestMapFontSizeToCSS(t *testing.T) {
	t.Parallel()

	t.Run("size 1 maps to x-small", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x-small", mapFontSizeToCSS("1"))
	})

	t.Run("size 2 maps to small", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "small", mapFontSizeToCSS("2"))
	})

	t.Run("size 3 maps to medium", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "medium", mapFontSizeToCSS("3"))
	})

	t.Run("size 4 maps to large", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "large", mapFontSizeToCSS("4"))
	})

	t.Run("size 5 maps to x-large", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x-large", mapFontSizeToCSS("5"))
	})

	t.Run("size 6 maps to xx-large", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "xx-large", mapFontSizeToCSS("6"))
	})

	t.Run("size 7 maps to xxx-large", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "xxx-large", mapFontSizeToCSS("7"))
	})

	t.Run("unknown size returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", mapFontSizeToCSS("0"))
	})

	t.Run("non-numeric returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", mapFontSizeToCSS("large"))
	})
}

func TestFindAncestorByTag(t *testing.T) {
	t.Parallel()

	t.Run("finds direct parent", func(t *testing.T) {
		t.Parallel()

		parent := &ast_domain.TemplateNode{TagName: "table"}
		child := &ast_domain.TemplateNode{TagName: "td"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{
			child: parent,
		}

		result := findAncestorByTag(child, "table", parentMap)

		assert.Equal(t, parent, result)
	})

	t.Run("finds grandparent", func(t *testing.T) {
		t.Parallel()

		grandparent := &ast_domain.TemplateNode{TagName: "table"}
		parent := &ast_domain.TemplateNode{TagName: "tr"}
		child := &ast_domain.TemplateNode{TagName: "td"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{
			child:  parent,
			parent: grandparent,
		}

		result := findAncestorByTag(child, "table", parentMap)

		assert.Equal(t, grandparent, result)
	})

	t.Run("returns nil when no matching ancestor", func(t *testing.T) {
		t.Parallel()

		parent := &ast_domain.TemplateNode{TagName: "div"}
		child := &ast_domain.TemplateNode{TagName: "span"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{
			child: parent,
		}

		result := findAncestorByTag(child, "table", parentMap)

		assert.Nil(t, result)
	})

	t.Run("returns nil for root node", func(t *testing.T) {
		t.Parallel()

		child := &ast_domain.TemplateNode{TagName: "div"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}

		result := findAncestorByTag(child, "table", parentMap)

		assert.Nil(t, result)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		t.Parallel()

		parent := &ast_domain.TemplateNode{TagName: "TABLE"}
		child := &ast_domain.TemplateNode{TagName: "td"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{
			child: parent,
		}

		result := findAncestorByTag(child, "table", parentMap)

		assert.Equal(t, parent, result)
	})
}

func TestBuildNodeProperties(t *testing.T) {
	t.Parallel()

	t.Run("nil CSS props and no presentational attrs returns nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{TagName: "div"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}

		result := buildNodeProperties(node, nil, parentMap)

		assert.Nil(t, result)
	})

	t.Run("CSS props only returns CSS props", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{TagName: "div"}
		cssProps := map[string]string{"color": "red"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}

		result := buildNodeProperties(node, cssProps, parentMap)

		assert.Equal(t, "red", result["color"])
	})

	t.Run("presentational attrs only returns those", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName: "img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "100"},
			},
		}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}

		result := buildNodeProperties(node, nil, parentMap)

		assert.Equal(t, "100px", result["width"])
	})

	t.Run("CSS props override presentational attrs", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName: "img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "100"},
			},
		}
		cssProps := map[string]string{"width": "200px"}
		parentMap := map[*ast_domain.TemplateNode]*ast_domain.TemplateNode{}

		result := buildNodeProperties(node, cssProps, parentMap)

		assert.Equal(t, "200px", result["width"])
	})
}

func TestMapDimensionAttributes(t *testing.T) {
	t.Parallel()

	t.Run("img element maps width and height", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "200"},
				{Name: "height", Value: "100"},
			},
		}
		properties := make(map[string]string)

		mapDimensionAttributes(node, properties, "img")

		assert.Equal(t, "200px", properties["width"])
		assert.Equal(t, "100px", properties["height"])
	})

	t.Run("div element does not map dimensions", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "200"},
			},
		}
		properties := make(map[string]string)

		mapDimensionAttributes(node, properties, "div")

		assert.Empty(t, properties)
	})

	t.Run("table element maps dimensions", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "50%"},
			},
		}
		properties := make(map[string]string)

		mapDimensionAttributes(node, properties, "table")

		assert.Equal(t, "50%", properties["width"])
	})
}

func TestMapAlignAttribute(t *testing.T) {
	t.Parallel()

	t.Run("table align center sets auto margins", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "align", Value: "center"}},
		}
		properties := make(map[string]string)

		mapAlignAttribute(node, properties, "table")

		assert.Equal(t, "auto", properties["margin-left"])
		assert.Equal(t, "auto", properties["margin-right"])
	})

	t.Run("td align sets text-align", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "align", Value: "right"}},
		}
		properties := make(map[string]string)

		mapAlignAttribute(node, properties, "td")

		assert.Equal(t, "right", properties["text-align"])
	})

	t.Run("no align attribute does nothing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{}
		properties := make(map[string]string)

		mapAlignAttribute(node, properties, "td")

		assert.Empty(t, properties)
	})
}

func TestMapBgcolourAttribute(t *testing.T) {
	t.Parallel()

	t.Run("body bgcolor maps to background-color", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "bgcolor", Value: "#ff0000"}},
		}
		properties := make(map[string]string)

		mapBgcolourAttribute(node, properties, "body")

		assert.Equal(t, "#ff0000", properties["background-color"])
	})

	t.Run("span bgcolor does nothing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "bgcolor", Value: "#ff0000"}},
		}
		properties := make(map[string]string)

		mapBgcolourAttribute(node, properties, "span")

		assert.Empty(t, properties)
	})
}

func TestMapCellspacingAttribute(t *testing.T) {
	t.Parallel()

	t.Run("table cellspacing maps to border-spacing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "cellspacing", Value: "5"}},
		}
		properties := make(map[string]string)

		mapCellspacingAttribute(node, properties, "table")

		assert.Equal(t, "5px", properties["border-spacing"])
	})

	t.Run("non-table element does nothing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "cellspacing", Value: "5"}},
		}
		properties := make(map[string]string)

		mapCellspacingAttribute(node, properties, "div")

		assert.Empty(t, properties)
	})

	t.Run("non-numeric cellspacing does nothing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "cellspacing", Value: "abc"}},
		}
		properties := make(map[string]string)

		mapCellspacingAttribute(node, properties, "table")

		assert.Empty(t, properties)
	})
}

func TestMapFontElementAttributes(t *testing.T) {
	t.Parallel()

	t.Run("font colour maps to color", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "color", Value: "blue"}},
		}
		properties := make(map[string]string)

		mapFontElementAttributes(node, properties, "font")

		assert.Equal(t, "blue", properties["color"])
	})

	t.Run("font size maps to font-size", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "size", Value: "4"}},
		}
		properties := make(map[string]string)

		mapFontElementAttributes(node, properties, "font")

		assert.Equal(t, "large", properties["font-size"])
	})

	t.Run("non-font element does nothing", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{{Name: "color", Value: "blue"}},
		}
		properties := make(map[string]string)

		mapFontElementAttributes(node, properties, "div")

		assert.Empty(t, properties)
	})
}

func TestGoTextFontMetrics_NoFonts_FallbackMeasure(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	width := metrics.MeasureText(
		layouter_domain.FontDescriptor{Family: "Nonexistent", Weight: 400},
		10.0,
		"Hello",
		layouter_domain.DirectionLTR,
	)

	assert.InDelta(t, 5*10.0*fallbackAdvanceFraction, width, 0.001)
}

func TestGoTextFontMetrics_NoFonts_FallbackGetMetrics(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	fontMetrics := metrics.GetMetrics(
		layouter_domain.FontDescriptor{Family: "Nonexistent", Weight: 400},
		10.0,
	)

	assert.InDelta(t, 10.0*defaultAscentFraction, fontMetrics.Ascent, 0.001)
	assert.InDelta(t, 10.0*defaultDescentFraction, fontMetrics.Descent, 0.001)
	assert.Equal(t, defaultUnitsPerEm, fontMetrics.UnitsPerEm)
}

func TestGoTextFontMetrics_NoFonts_GetFontData_NotFound(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	data, found := metrics.GetFontData(layouter_domain.FontDescriptor{Family: "Nonexistent"})

	assert.False(t, found)
	assert.Nil(t, data)
}

func TestGoTextFontMetrics_NoFonts_GetFontFace_Nil(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	face := metrics.GetFontFace(layouter_domain.FontDescriptor{Family: "Nonexistent"})

	assert.Nil(t, face)
}

func TestGoTextFontMetrics_NoFonts_ResolveFallback_ReturnsSame(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	font := layouter_domain.FontDescriptor{Family: "Missing", Weight: 400}
	result := metrics.ResolveFallback(font, 'A')

	assert.Equal(t, font, result)
}

func TestGoTextFontMetrics_SplitGraphemeClusters(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	t.Run("empty string returns nil", func(t *testing.T) {
		t.Parallel()

		result := metrics.SplitGraphemeClusters("")

		assert.Nil(t, result)
	})

	t.Run("ASCII text splits correctly", func(t *testing.T) {
		t.Parallel()

		result := metrics.SplitGraphemeClusters("abc")

		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
}

func TestGoTextFontMetrics_MeasureText_EmptyString(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	width := metrics.MeasureText(
		layouter_domain.FontDescriptor{Family: "Nonexistent"},
		12.0,
		"",
		layouter_domain.DirectionLTR,
	)

	assert.Equal(t, 0.0, width)
}

func TestGoTextFontMetrics_ShapeText_NoFont_Fallback(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	glyphs := metrics.ShapeText(
		layouter_domain.FontDescriptor{Family: "Nonexistent"},
		10.0,
		"AB",
		layouter_domain.DirectionLTR,
	)

	require.Len(t, glyphs, 2)
	assert.InDelta(t, 10.0*fallbackAdvanceFraction, glyphs[0].XAdvance, 0.001)
}

func TestGoTextFontMetrics_ShapeText_EmptyString(t *testing.T) {
	t.Parallel()

	metrics, err := NewGoTextFontMetrics(nil)
	require.NoError(t, err)

	glyphs := metrics.ShapeText(
		layouter_domain.FontDescriptor{Family: "Nonexistent"},
		10.0,
		"",
		layouter_domain.DirectionLTR,
	)

	assert.Empty(t, glyphs)
}

func TestMapDirection(t *testing.T) {
	t.Parallel()

	t.Run("RTL maps correctly", func(t *testing.T) {
		t.Parallel()

		result := mapDirection(layouter_domain.DirectionRTL)

		assert.NotZero(t, result)
	})

	t.Run("LTR maps correctly", func(t *testing.T) {
		t.Parallel()

		result := mapDirection(layouter_domain.DirectionLTR)

		assert.NotNil(t, result)
	})
}
