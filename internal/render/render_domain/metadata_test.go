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

package render_domain

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_dto"
)

func TestBuildSvgSpriteSheet_CollectsSymbols(t *testing.T) {
	svgHome := &ParsedSvgData{
		InnerHTML:  `<path d="M10 20v-6h4v6"/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
	}
	svgHome.CachedSymbol, svgHome.CachedDefs, _ = ComputeSymbolAndDefs("icon-home", svgHome)
	svgUser := &ParsedSvgData{
		InnerHTML:  `<circle cx="12" cy="12" r="4"/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
	}
	svgUser.CachedSymbol, svgUser.CachedDefs, _ = ComputeSymbolAndDefs("icon-user", svgUser)

	mockReg := newTestRegistryBuilder().
		withSVG("icon-home", `<path d="M10 20v-6h4v6"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		withSVG("icon-user", `<circle cx="12" cy="12" r="4"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
		svgSymbolEntry{id: "icon-home", data: svgHome},
		svgSymbolEntry{id: "icon-user", data: svgUser},
	)

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	assert.Contains(t, spriteSheet, `<symbol id="icon-home"`)
	assert.Contains(t, spriteSheet, `<symbol id="icon-user"`)
	assert.Contains(t, spriteSheet, `<path d="M10 20v-6h4v6"/>`)
	assert.Contains(t, spriteSheet, `<circle cx="12" cy="12" r="4"/>`)
}

func TestBuildSvgSpriteSheet_DeduplicatesSymbols(t *testing.T) {
	svgDup := &ParsedSvgData{
		InnerHTML:  `<path d="M0 0"/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
	}
	svgDup.CachedSymbol, svgDup.CachedDefs, _ = ComputeSymbolAndDefs("icon-dup", svgDup)

	mockReg := newTestRegistryBuilder().
		withSVG("icon-dup", `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	entry := svgSymbolEntry{id: "icon-dup", data: svgDup}
	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols, entry)

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	count := strings.Count(spriteSheet, `<symbol id="icon-dup"`)
	assert.Equal(t, 1, count, "symbol should appear exactly once")
}

func TestBuildSvgSpriteSheet_ReturnsEmptyForNoSymbols(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	ro := NewTestOrchestratorBuilder().Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)
	assert.Empty(t, spriteSheet)
}

func TestBuildSvgSpriteSheet_HandlesParallelProcessing(t *testing.T) {
	ClearSpriteSheetCacheForTesting()

	rb := newTestRegistryBuilder()
	expectedIDs := make([]string, 15)
	svgEntries := make(map[string]*ParsedSvgData, 15)
	for i := range 15 {
		id := "icon-" + string(rune('a'+i))
		expectedIDs[i] = id
		rb.withSVG(id, `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"})
		entry := &ParsedSvgData{
			InnerHTML:  `<path d="M0 0"/>`,
			Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
		}
		entry.CachedSymbol, entry.CachedDefs, _ = ComputeSymbolAndDefs(id, entry)
		svgEntries[id] = entry
	}
	mockReg := rb.build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	for _, id := range expectedIDs {
		rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
			svgSymbolEntry{id: id, data: svgEntries[id]})
	}

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	for _, id := range expectedIDs {
		assert.Contains(t, spriteSheet, `<symbol id="`+id+`"`, "missing symbol: %s", id)
	}
}

func TestAssembleSpriteSheet_FormatsCorrectly(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	symbols := []string{
		`<symbol id="icon-a" viewBox="0 0 24 24"><path d="A"/></symbol>`,
		`<symbol id="icon-b" viewBox="0 0 24 24"><path d="B"/></symbol>`,
	}

	result := assembleSpriteSheet(symbols, nil, rctx)

	assert.True(t, strings.HasPrefix(result, `<svg xmlns="http://www.w3.org/2000/svg" id="sprite"`))
	assert.True(t, strings.HasSuffix(result, `</svg>`))

	assert.Contains(t, result, `<symbol id="icon-a"`)
	assert.Contains(t, result, `<symbol id="icon-b"`)
}

func TestAssembleSpriteSheet_SkipsEmptySymbols(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	symbols := []string{
		`<symbol id="icon-a" viewBox="0 0 24 24"><path d="A"/></symbol>`,
		"",
		`<symbol id="icon-b" viewBox="0 0 24 24"><path d="B"/></symbol>`,
	}

	result := assembleSpriteSheet(symbols, nil, rctx)

	assert.Contains(t, result, `<symbol id="icon-a"`)
	assert.Contains(t, result, `<symbol id="icon-b"`)

	count := strings.Count(result, "<symbol")
	assert.Equal(t, 2, count)
}

func TestAssembleSpriteSheet_HandlesEmptySymbolList(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	result := assembleSpriteSheet([]string{}, nil, rctx)

	assert.Contains(t, result, `<svg xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, result, `</svg>`)
}

func TestCollectAndSortSVGIDs_SortsDeterministically(t *testing.T) {
	entries := []svgSymbolEntry{
		{id: "zebra"},
		{id: "apple"},
		{id: "mango"},
	}

	result := collectAndSortSVGIDs(entries)

	assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
}

func TestCollectAndSortSVGIDs_HandlesEmptySlice(t *testing.T) {
	result := collectAndSortSVGIDs(nil)
	assert.Empty(t, result)
}

func TestCollectAndSortSVGIDs_PreservesAllEntries(t *testing.T) {
	entries := []svgSymbolEntry{
		{id: "a"},
		{id: "b"},
		{id: "c"},
		{id: "d"},
		{id: "e"},
	}

	result := collectAndSortSVGIDs(entries)

	assert.Len(t, result, 5)

	assert.True(t, sort.StringsAreSorted(result))
}

func TestAddLinkHeaderIfUnique_DeduplicatesHeaders(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	header := render_dto.LinkHeader{
		URL: "/test.js",
		Rel: "preload",
		As:  "script",
	}

	rctx.addLinkHeaderIfUnique(header)
	rctx.addLinkHeaderIfUnique(header)
	rctx.addLinkHeaderIfUnique(header)

	assert.Len(t, rctx.collectedLinkHeaders, 1)
}

func TestAddLinkHeaderIfUnique_AllowsDifferentHeaders(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{URL: "/a.js", Rel: "preload", As: "script"})
	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{URL: "/b.js", Rel: "preload", As: "script"})
	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{URL: "/c.css", Rel: "preload", As: "style"})

	assert.Len(t, rctx.collectedLinkHeaders, 3)
}

func TestIsGoogleFontsURL_DetectsGoogleFonts(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{url: "https://fonts.googleapis.com/css2?family=Roboto", expected: true},
		{url: "https://fonts.gstatic.com/s/roboto/v30/roboto.woff2", expected: true},
		{url: "https://FONTS.GOOGLEAPIS.COM/css", expected: true},
		{url: "https://example.com/fonts.css", expected: false},
		{url: "https://example.com", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			assert.Equal(t, tc.expected, isGoogleFontsURL(tc.url))
		})
	}
}

func TestDetermineFontAssetType_ReturnsCorrectTypes(t *testing.T) {
	testCases := []struct {
		url          string
		expectedAs   string
		expectedType string
	}{
		{url: "/fonts/roboto.woff2", expectedAs: "font", expectedType: "font/woff2"},
		{url: "/fonts/roboto.woff", expectedAs: "style", expectedType: ""},
		{url: "https://fonts.googleapis.com/css2", expectedAs: "style", expectedType: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			asType, fontType := determineFontAssetType(tc.url)
			assert.Equal(t, tc.expectedAs, asType)
			assert.Equal(t, tc.expectedType, fontType)
		})
	}
}

func TestBuildPreloadLogic_WithNoPreloadsGlobally(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	preload, script := ro.buildPreloadLogic(
		[]string{"comp-a", "comp-b"},
		true,
		rctx,
	)

	assert.Empty(t, preload)
	assert.Empty(t, script)
}

func TestBuildPreloadLogic_PreloadsAllComponents(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"comp-a": {BaseJSPath: "/js/comp-a.js"},
		"comp-b": {BaseJSPath: "/js/comp-b.js"},
	}

	preload, script := ro.buildPreloadLogic(
		[]string{"comp-a", "comp-b"},
		false,
		rctx,
	)

	assert.Contains(t, preload, `<link rel="modulepreload" href="/js/comp-a.js">`)
	assert.Contains(t, preload, `<link rel="modulepreload" href="/js/comp-b.js">`)
	assert.Contains(t, script, `<script type="module" src="/js/comp-a.js"></script>`)
	assert.Contains(t, script, `<script type="module" src="/js/comp-b.js"></script>`)
}

func TestBuildPreloadLogic_PreloadsAllComponentsWithJS(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"comp-a": {BaseJSPath: "/js/a.js"},
		"comp-b": {BaseJSPath: "/js/b.js"},
	}

	preload, script := ro.buildPreloadLogic(
		[]string{"comp-a", "comp-b"},
		false,
		rctx,
	)

	assert.Contains(t, preload, `<link rel="modulepreload" href="/js/a.js">`)
	assert.Contains(t, preload, `<link rel="modulepreload" href="/js/b.js">`)
	assert.Contains(t, script, `<script type="module" src="/js/a.js"></script>`)
	assert.Contains(t, script, `<script type="module" src="/js/b.js"></script>`)
}

func TestBuildPreloadLogic_SkipsComponentsWithoutJS(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"has-js": {BaseJSPath: "/js/has-js.js"},
		"no-js":  {BaseJSPath: ""},
	}

	preload, script := ro.buildPreloadLogic(
		[]string{"has-js", "no-js"},
		false,
		rctx,
	)

	assert.Contains(t, preload, "has-js.js")
	assert.NotContains(t, preload, "no-js")
	assert.NotContains(t, script, "no-js")
}

func TestBuildPreloadLogic_SortsComponentsDeterministically(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"zebra":  {BaseJSPath: "/js/zebra.js"},
		"alpha":  {BaseJSPath: "/js/alpha.js"},
		"middle": {BaseJSPath: "/js/middle.js"},
	}

	preload, _ := ro.buildPreloadLogic(
		[]string{"zebra", "alpha", "middle"},
		false,
		rctx,
	)

	alphaPos := strings.Index(preload, "alpha.js")
	middlePos := strings.Index(preload, "middle.js")
	zebraPos := strings.Index(preload, "zebra.js")

	assert.True(t, alphaPos < middlePos, "alpha should come before middle")
	assert.True(t, middlePos < zebraPos, "middle should come before zebra")
}

func TestAppendPreloadTags_FormatsCorrectly(t *testing.T) {
	preload := make([]byte, 0, 256)
	script := make([]byte, 0, 256)

	appendPreloadTags(&preload, &script, "/js/component.js", "")

	assert.Equal(t, `<link rel="modulepreload" href="/js/component.js">`, string(preload))
	assert.Equal(t, `<script type="module" src="/js/component.js"></script>`, string(script))
}

func TestAppendPreloadTags_WithSRIHash(t *testing.T) {
	preload := make([]byte, 0, 256)
	script := make([]byte, 0, 256)

	appendPreloadTags(&preload, &script, "/js/component.js", "sha384-abc123")

	assert.Equal(t, `<link rel="modulepreload" href="/js/component.js" integrity="sha384-abc123" crossorigin="anonymous">`, string(preload))
	assert.Equal(t, `<script type="module" src="/js/component.js" integrity="sha384-abc123" crossorigin="anonymous"></script>`, string(script))
}

func TestAppendPreloadTags_EscapesSpecialCharacters(t *testing.T) {
	preload := make([]byte, 0, 256)
	script := make([]byte, 0, 256)

	appendPreloadTags(&preload, &script, `/js/comp<script>.js`, "")

	assert.Contains(t, string(preload), "&lt;script&gt;")
	assert.Contains(t, string(script), "&lt;script&gt;")
}

func TestFormatComponentNotFoundError_ContainsHelpfulInfo(t *testing.T) {
	err := formatComponentNotFoundError("my-widget")

	assert.Contains(t, err, "my-widget")

	assert.Contains(t, err, "components/my-widget.pkc")
	assert.Contains(t, err, "remove the <my-widget>")

	assert.Contains(t, err, "piko.sh")
}

func TestAddStandardLinkHeaders_AddsFrameworkAndTheme(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	addStandardLinkHeaders(rctx)

	require.Len(t, rctx.collectedLinkHeaders, 2)

	foundFramework := false
	foundTheme := false
	for _, h := range rctx.collectedLinkHeaders {
		if strings.Contains(h.URL, "ppframework") {
			foundFramework = true
			assert.Equal(t, "preload", h.Rel)
			assert.Equal(t, "script", h.As)
		}
		if strings.Contains(h.URL, "theme.css") {
			foundTheme = true
			assert.Equal(t, "preload", h.Rel)
			assert.Equal(t, "style", h.As)
		}
	}
	assert.True(t, foundFramework, "should include framework JS header")
	assert.True(t, foundTheme, "should include theme CSS header")
}

func TestAddStandardLinkHeaders_DeduplicatesOnMultipleCalls(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	addStandardLinkHeaders(rctx)
	addStandardLinkHeaders(rctx)
	addStandardLinkHeaders(rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 2)
}

func TestProcessFontConfigurations_AddsGoogleFontsPreconnect(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	fonts := []config.FontDefinition{
		{URL: "https://fonts.googleapis.com/css2?family=Roboto", Instant: true},
	}

	processFontConfigurations(fonts, rctx)

	require.GreaterOrEqual(t, len(rctx.collectedLinkHeaders), 3)

	hasGoogleAPIs := false
	hasGstatic := false
	for _, h := range rctx.collectedLinkHeaders {
		if h.URL == "https://fonts.googleapis.com" && h.Rel == "preconnect" {
			hasGoogleAPIs = true
		}
		if h.URL == "https://fonts.gstatic.com" && h.Rel == "preconnect" {
			hasGstatic = true
			assert.Equal(t, "anonymous", h.CrossOrigin)
		}
	}
	assert.True(t, hasGoogleAPIs, "should include Google APIs preconnect")
	assert.True(t, hasGstatic, "should include Gstatic preconnect with crossorigin")
}

func TestProcessFontConfigurations_AddsLocalFontWithoutPreconnect(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	fonts := []config.FontDefinition{
		{URL: "/fonts/local.css", Instant: true},
	}

	processFontConfigurations(fonts, rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 1)
	assert.Equal(t, "/fonts/local.css", rctx.collectedLinkHeaders[0].URL)
	assert.Equal(t, "preload", rctx.collectedLinkHeaders[0].Rel)
}

func TestProcessFontConfigurations_HandlesMixedFonts(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	fonts := []config.FontDefinition{
		{URL: "https://fonts.googleapis.com/css2?family=Inter", Instant: true},
		{URL: "/fonts/custom.woff2", Instant: true},
	}

	processFontConfigurations(fonts, rctx)

	hasPreconnect := false
	hasGoogleFont := false
	hasLocalFont := false
	for _, h := range rctx.collectedLinkHeaders {
		if h.Rel == "preconnect" {
			hasPreconnect = true
		}
		if strings.Contains(h.URL, "fonts.googleapis.com/css2") {
			hasGoogleFont = true
		}
		if h.URL == "/fonts/custom.woff2" {
			hasLocalFont = true
			assert.Equal(t, "font", h.As)
		}
	}
	assert.True(t, hasPreconnect, "should have preconnect headers")
	assert.True(t, hasGoogleFont, "should include Google font")
	assert.True(t, hasLocalFont, "should include local font")
}

func TestProcessFontConfigurations_EmptyFontsDoesNothing(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	processFontConfigurations([]config.FontDefinition{}, rctx)

	assert.Empty(t, rctx.collectedLinkHeaders)
}

func TestBuildSvgSpriteSheet_UsesPreFetchedData(t *testing.T) {
	ClearSpriteSheetCacheForTesting()

	rb := newTestRegistryBuilder()
	entries := make([]svgSymbolEntry, 15)
	for i := range 15 {
		id := "icon-" + string(rune('a'+i))
		rb.withSVG(id, `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"})
		entry := &ParsedSvgData{
			InnerHTML:  `<path d="M0 0"/>`,
			Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
		}
		entry.CachedSymbol, entry.CachedDefs, _ = ComputeSymbolAndDefs(id, entry)
		entries[i] = svgSymbolEntry{id: id, data: entry}
	}
	mockReg := rb.build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()
	rctx.requiredSvgSymbols = entries

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	for i := range 15 {
		id := "icon-" + string(rune('a'+i))
		assert.Contains(t, spriteSheet, `<symbol id="`+id+`"`)
	}

	assert.Empty(t, rctx.diagnostics.Warnings)
}

func TestBuildPreloadLogic_NilMetadataReturnsEmpty(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	preload, script := ro.buildPreloadLogic(
		[]string{"comp-a"},
		false,
		rctx,
	)

	assert.Empty(t, preload)
	assert.Empty(t, script)
}

func TestBuildPreloadLogic_EmptyComponentsReturnsEmpty(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{}

	preload, script := ro.buildPreloadLogic(
		[]string{},
		false,
		rctx,
	)

	assert.Empty(t, preload)
	assert.Empty(t, script)
}

func TestBuildPreloadLogic_NilMetadataFromErrorReturnsEmpty(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	preload, script := ro.buildPreloadLogic(
		[]string{"comp-a"},
		false,
		rctx,
	)

	assert.Empty(t, preload)
	assert.Empty(t, script)
}

func TestBuildSvgSpriteSheet_HandlesNilData(t *testing.T) {
	ClearSpriteSheetCacheForTesting()

	rctx := NewTestRenderContextBuilder().Build()

	for i := range 5 {
		rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
			svgSymbolEntry{id: "icon-" + string(rune('a'+i)), data: nil})
	}

	ro := NewTestOrchestratorBuilder().Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	assert.Contains(t, spriteSheet, `<svg xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, spriteSheet, `</svg>`)
	assert.NotContains(t, spriteSheet, `<symbol`)
}

func TestBuildSvgSpriteSheet_HandlesMixedNilData(t *testing.T) {
	ClearSpriteSheetCacheForTesting()

	svgA := &ParsedSvgData{
		InnerHTML:  `<path d="M0 0"/>`,
		Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
	}
	svgA.CachedSymbol, svgA.CachedDefs, _ = ComputeSymbolAndDefs("icon-a", svgA)

	mockReg := newTestRegistryBuilder().
		withSVG("icon-a", `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	rctx.requiredSvgSymbols = append(rctx.requiredSvgSymbols,
		svgSymbolEntry{id: "icon-a", data: svgA},
		svgSymbolEntry{id: "icon-missing", data: nil},
	)

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	require.NoError(t, err)

	assert.Contains(t, spriteSheet, `<symbol id="icon-a"`)
	assert.NotContains(t, spriteSheet, `icon-missing`)
}

func TestBuildPreloadLogic_PreloadsRequiredModules(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"comp-a": {
			BaseJSPath:      "/js/comp-a.js",
			RequiredModules: []string{"/_piko/assets/mod/lib/shared.js", "/_piko/assets/mod/lib/only-a.js"},
		},
		"comp-b": {
			BaseJSPath:      "/js/comp-b.js",
			RequiredModules: []string{"/_piko/assets/mod/lib/shared.js"},
		},
	}

	preload, script := ro.buildPreloadLogic([]string{"comp-a", "comp-b"}, false, rctx)

	assert.Equal(t, 1, strings.Count(preload, `href="/_piko/assets/mod/lib/shared.js"`),
		"a module two components share must be preloaded once")
	assert.Contains(t, preload, `<link rel="modulepreload" href="/_piko/assets/mod/lib/only-a.js">`)

	assert.NotContains(t, script, "/_piko/assets/mod/lib/",
		"a library module is not an entry point and must not get a script tag")

	lastComponent := strings.LastIndex(preload, `href="/js/comp-b.js"`)
	firstModule := strings.Index(preload, `href="/_piko/assets/mod/lib/`)
	assert.Less(t, lastComponent, firstModule,
		"component entry points must precede library modules in the preload scanner's path")
}

func TestBuildPreloadLogic_SortsRequiredModulesDeterministically(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	renderOnce := func(componentTypes []string) string {
		rctx := NewTestRenderContextBuilder().Build()
		rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
			"comp-a": {BaseJSPath: "/js/comp-a.js", RequiredModules: []string{"/lib/z.js", "/lib/m.js"}},
			"comp-b": {BaseJSPath: "/js/comp-b.js", RequiredModules: []string{"/lib/a.js"}},
		}
		preload, _ := ro.buildPreloadLogic(componentTypes, false, rctx)
		return preload
	}

	forward := renderOnce([]string{"comp-a", "comp-b"})
	reversed := renderOnce([]string{"comp-b", "comp-a"})

	assert.Equal(t, forward, reversed, "output must not depend on the order components were discovered")
	assert.Less(t, strings.Index(forward, "/lib/a.js"), strings.Index(forward, "/lib/m.js"))
	assert.Less(t, strings.Index(forward, "/lib/m.js"), strings.Index(forward, "/lib/z.js"))
}

func TestBuildPreloadLogic_DoesNotPreloadAComponentTwiceAsAModule(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"comp-a": {BaseJSPath: "/js/comp-b.js", RequiredModules: []string{"/js/comp-b.js"}},
		"comp-b": {BaseJSPath: "/js/comp-b.js"},
	}

	preload, _ := ro.buildPreloadLogic([]string{"comp-a", "comp-b"}, false, rctx)

	assert.Equal(t, 2, strings.Count(preload, `href="/js/comp-b.js"`),
		"the two entry points still preload, but the module must not add a third")
}

func TestBuildPreloadLogic_LeavesTheCachedModuleListAlone(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	rctx := NewTestRenderContextBuilder().Build()

	cached := []string{"/lib/z.js", "/lib/a.js"}
	rctx.componentMetadata = map[string]*render_dto.ComponentMetadata{
		"comp-a": {BaseJSPath: "/js/comp-a.js", RequiredModules: cached},
	}

	_, _ = ro.buildPreloadLogic([]string{"comp-a"}, false, rctx)

	assert.Equal(t, []string{"/lib/z.js", "/lib/a.js"}, cached,
		"the module list belongs to a shared cache entry and must not be sorted in place")
}
