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

//go:build integration

package layouter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"
	browserpkg "piko.sh/piko/wdk/browser"
)

const layouterCSSReset = `*, *::before, *::after {
  box-sizing: border-box;
}
html, body, div, span, applet, object, iframe,
h1, h2, h3, h4, h5, h6, p, blockquote, pre,
a, abbr, acronym, address, big, cite, code,
del, dfn, em, img, ins, kbd, q, s, samp,
small, strike, strong, sub, sup, tt, var,
b, u, i, center,
dl, dt, dd, ol, ul, li,
fieldset, form, label, legend,
table, caption, tbody, tfoot, thead, tr, th, td,
article, aside, canvas, details, embed,
figure, figcaption, footer, header, hgroup,
menu, nav, output, ruby, section, summary,
time, mark, audio, video {
  margin: 0;
  padding: 0;
  border: 0;
  vertical-align: baseline;
}
body {
  line-height: 1.4;
}
img {
  max-width: 100%;
  max-height: 100%;
}
a {
  text-decoration: none;
}
`

const extractPositionsJS = `(() => {
	const result = {};
	document.querySelectorAll('[data-layout-id]').forEach(element => {
		const rect = element.getBoundingClientRect();
		const identifier = element.getAttribute('data-layout-id');
		const entry = {
			x: rect.x,
			y: rect.y,
			width: rect.width,
			height: rect.height
		};
		const textRects = [];
		const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT, {
			acceptNode: function(node) {
				let parent = node.parentElement;
				while (parent && parent !== element) {
					if (parent.hasAttribute('data-layout-id')) return NodeFilter.FILTER_REJECT;
					parent = parent.parentElement;
				}
				return NodeFilter.FILTER_ACCEPT;
			}
		});
		let textNode;
		while ((textNode = walker.nextNode()) !== null) {
			if (textNode.textContent.trim() === '') continue;
			const rng = document.createRange();
			rng.selectNodeContents(textNode);
			const clientRects = rng.getClientRects();
			for (let i = 0; i < clientRects.length; i++) {
				const r = clientRects[i];
				if (r.width > 0 && r.height > 0) {
					textRects.push({x: r.x, y: r.y, width: r.width, height: r.height});
				}
			}
		}
		if (textRects.length > 0) {
			entry.textRects = textRects;
		}
		result[identifier] = entry;
	});
	return JSON.stringify(result);
})()`

func runLayouterTestCase(t *testing.T, tc testCase) {
	t.Helper()

	fmt.Printf("\n--- [LAYOUTER] START: %s ---\n", tc.Name)

	harness := newLayouterHarness(t, tc)
	defer harness.cleanup()

	fmt.Println("[LAYOUTER] Phase 1: Setting up test environment...")
	require.NoError(t, harness.setup(), "failed to setup harness")

	if harness.spec.Skip != "" {
		t.Skip(harness.spec.Skip)
	}

	fmt.Println("[LAYOUTER] Phase 2: Building server...")
	require.NoError(t, harness.buildServer(), "failed to build server")

	if harness.spec.ExpectedPositions != nil {

		fmt.Println("[LAYOUTER] Phase 3: Running layouter pipeline...")
		layouterPositions := extractLayouterPositions(t, harness.tempDirectory, harness.spec)

		t.Log("expected positions:")
		for identifier, rect := range harness.spec.ExpectedPositions {
			t.Logf("  %s: page=%d x=%.1f y=%.1f w=%.1f h=%.1f",
				identifier, rect.PageIndex, rect.X, rect.Y, rect.Width, rect.Height)
		}

		fmt.Println("[LAYOUTER] Phase 4: Comparing positions...")
		comparePositions(t, harness.spec.ExpectedPositions, layouterPositions, harness.spec.Tolerance)
	} else {

		fmt.Println("[LAYOUTER] Phase 3: Starting server...")
		require.NoError(t, harness.startServer(), "failed to start server")

		fmt.Println("[LAYOUTER] Phase 4: Setting up browser...")
		require.NoError(t, harness.setupBrowser(), "failed to setup browser")

		fmt.Println("[LAYOUTER] Phase 5: Setting viewport...")
		setViewportError := browserpkg.SetViewport(
			harness.actionContext,
			int64(harness.spec.ViewportWidth),
			int64(harness.spec.ViewportHeight),
		)
		require.NoError(t, setViewportError, "failed to set viewport")

		fmt.Println("[LAYOUTER] Phase 6: Navigating to page...")
		require.NoError(t, harness.navigateToPage(), "failed to navigate to page")

		time.Sleep(500 * time.Millisecond)

		fmt.Println("[LAYOUTER] Phase 6b: Injecting test font...")
		require.NoError(t, injectTestFont(harness.incognitoPage.Ctx), "failed to inject test font")

		fmt.Println("[LAYOUTER] Phase 7: Extracting browser positions...")
		browserPositions := extractBrowserPositions(t, harness.incognitoPage.Ctx)

		fmt.Println("[LAYOUTER] Phase 8: Running layouter pipeline...")
		layouterPositions := extractLayouterPositions(t, harness.tempDirectory, harness.spec)

		fmt.Println("[LAYOUTER] Phase 9: Comparing positions...")
		comparePositions(t, browserPositions, layouterPositions, harness.spec.Tolerance)
	}

	fmt.Printf("--- [LAYOUTER] COMPLETE: %s ---\n", tc.Name)
}

func extractBrowserPositions(t *testing.T, browserContext context.Context) map[string]elementRect {
	t.Helper()

	var resultJSON string
	err := chromedp.Run(browserContext, chromedp.Evaluate(extractPositionsJS, &resultJSON))
	require.NoError(t, err, "failed to extract browser positions via JS")

	var positions map[string]elementRect
	err = json.Unmarshal([]byte(resultJSON), &positions)
	require.NoError(t, err, "failed to parse browser positions JSON")

	t.Log("browser positions:")
	for identifier, rect := range positions {
		t.Logf("  %s: x=%.1f y=%.1f w=%.1f h=%.1f", identifier, rect.X, rect.Y, rect.Width, rect.Height)
		for i, tr := range rect.TextRects {
			t.Logf("    text[%d]: x=%.1f y=%.1f w=%.1f h=%.1f", i, tr.X, tr.Y, tr.Width, tr.Height)
		}
	}

	return positions
}

func extractLayouterPositions(t *testing.T, tempDirectory string, spec *layouterTestSpec) map[string]elementRect {
	t.Helper()

	pikoRoot, err := findPikoProjectRoot()
	require.NoError(t, err, "failed to find piko project root")
	fontPath := filepath.Join(pikoRoot, "internal", "fonts", "NotoSans-Regular.ttf")

	manifestPath := filepath.Join(tempDirectory, "dist", "manifest.bin")

	command := exec.Command(filepath.Join(tempDirectory, "server"))
	command.Dir = tempDirectory
	envVars := []string{
		"PIKO_EXTRACT_POSITIONS=1",
		"PIKO_MANIFEST_PATH=" + manifestPath,
		"PIKO_REQUEST_PATH=" + spec.RequestURL,
		fmt.Sprintf("PIKO_VIEWPORT_WIDTH=%d", spec.ViewportWidth),
		fmt.Sprintf("PIKO_VIEWPORT_HEIGHT=%d", spec.ViewportHeight),
		"PIKO_FONT_PATH=" + fontPath,
		"PIKO_EXTRA_CSS=" + layouterCSSReset,
		"GOWORK=off",
	}
	if spec.PageHeightPx > 0 {
		envVars = append(envVars,
			"PIKO_PAGINATE=1",
			fmt.Sprintf("PIKO_PAGE_WIDTH_PX=%g", spec.PageWidthPx),
			fmt.Sprintf("PIKO_PAGE_HEIGHT_PX=%g", spec.PageHeightPx),
			fmt.Sprintf("PIKO_PAGE_MARGIN_PX=%g", spec.PageMarginPx),
		)
	}
	command.Env = append(os.Environ(), envVars...)

	output, err := command.Output()
	if err != nil {
		if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
			t.Fatalf("layout extraction failed: %v\nStderr: %s", err, string(exitError.Stderr))
		}
		t.Fatalf("layout extraction failed: %v", err)
	}

	var positions map[string]elementRect
	err = json.Unmarshal(output, &positions)
	require.NoError(t, err, "failed to parse layout extraction output")

	t.Log("layouter positions:")
	for identifier, rect := range positions {
		if spec.PageHeightPx > 0 {
			t.Logf("  %s: page=%d x=%.1f y=%.1f w=%.1f h=%.1f",
				identifier, rect.PageIndex, rect.X, rect.Y, rect.Width, rect.Height)
		} else {
			t.Logf("  %s: x=%.1f y=%.1f w=%.1f h=%.1f",
				identifier, rect.X, rect.Y, rect.Width, rect.Height)
		}
		for i, tr := range rect.TextRects {
			t.Logf("    text[%d]: x=%.1f y=%.1f w=%.1f h=%.1f", i, tr.X, tr.Y, tr.Width, tr.Height)
		}
	}

	return positions
}

func comparePositions(
	t *testing.T,
	expected map[string]elementRect,
	actual map[string]elementRect,
	tolerance float64,
) {
	t.Helper()

	require.NotEmpty(t, expected, "no expected positions defined")

	for identifier, expectedRect := range expected {
		actualRect, exists := actual[identifier]
		if !exists {
			t.Errorf("data-layout-id=%q expected but not found in layouter output", identifier)
			continue
		}

		if expectedRect.PageIndex != actualRect.PageIndex {
			t.Errorf("[%s] pageIndex: expected=%d actual=%d",
				identifier, expectedRect.PageIndex, actualRect.PageIndex)
		}

		assertWithinTolerance(t, identifier, "x", expectedRect.X, actualRect.X, tolerance)
		assertWithinTolerance(t, identifier, "y", expectedRect.Y, actualRect.Y, tolerance)
		assertWithinTolerance(t, identifier, "width", expectedRect.Width, actualRect.Width, tolerance)
		assertWithinTolerance(t, identifier, "height", expectedRect.Height, actualRect.Height, tolerance)

		compareTextRects(t, identifier, expectedRect.TextRects, actualRect.TextRects, tolerance)
	}

	for identifier := range actual {
		if _, exists := expected[identifier]; !exists {
			t.Errorf("data-layout-id=%q found in layouter output but not expected", identifier)
		}
	}
}

func compareTextRects(
	t *testing.T,
	identifier string,
	expectedRects []elementRect,
	actualRects []elementRect,
	tolerance float64,
) {
	t.Helper()

	if len(expectedRects) == 0 && len(actualRects) == 0 {
		return
	}

	if len(actualRects) <= 1 {
		return
	}

	if len(expectedRects) != len(actualRects) {
		t.Errorf("[%s] text rect count: expected=%d actual=%d",
			identifier, len(expectedRects), len(actualRects))
		return
	}

	for i := range expectedRects {
		label := fmt.Sprintf("%s/text[%d]", identifier, i)
		assertWithinTolerance(t, label, "x", expectedRects[i].X, actualRects[i].X, tolerance)
		assertWithinTolerance(t, label, "y", expectedRects[i].Y, actualRects[i].Y, tolerance)
		assertWithinTolerance(t, label, "width", expectedRects[i].Width, actualRects[i].Width, tolerance)
		assertWithinTolerance(t, label, "height", expectedRects[i].Height, actualRects[i].Height, tolerance)
	}
}

func assertWithinTolerance(
	t *testing.T,
	identifier string,
	dimension string,
	expected float64,
	actual float64,
	tolerance float64,
) {
	t.Helper()

	delta := math.Abs(expected - actual)
	if delta > tolerance {
		t.Errorf("[%s] %s: expected=%.2f actual=%.2f delta=%.2f (tolerance=%.2f)",
			identifier, dimension, expected, actual, delta, tolerance)
	}
}

func injectTestFont(browserContext context.Context) error {
	dataURI := base64FontDataURI()
	boldDataURI := base64BoldFontDataURI()

	loadScript := fmt.Sprintf(
		`(async () => {
			const s = document.createElement('style');
			s.textContent = "@font-face { font-family: 'NotoSans'; src: url('%s') format('truetype'); font-weight: 400; font-style: normal; font-display: block; } @font-face { font-family: 'NotoSans'; src: url('%s') format('truetype'); font-weight: 700; font-style: normal; font-display: block; } * { font-family: 'NotoSans' !important; }";
			document.head.appendChild(s);
			await document.fonts.load("12px NotoSans");
			await document.fonts.load("bold 12px NotoSans");
			await document.fonts.ready;
			document.body.offsetHeight;
			return true;
		})()`, dataURI, boldDataURI)

	var loaded bool
	err := chromedp.Run(browserContext,
		chromedp.Evaluate(loadScript, &loaded, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	)
	if err != nil {
		return fmt.Errorf("injecting and loading test font: %w", err)
	}

	return nil
}
