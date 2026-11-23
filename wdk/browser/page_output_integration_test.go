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

package browser

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
	"piko.sh/safedisk"
)

func imageWidth(t *testing.T, data []byte) int {
	t.Helper()
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return config.Width
}

func requirePDF(t *testing.T, data []byte) {
	t.Helper()
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")), "not a PDF document")
}

func TestPageScreenshotsAndPDF(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.SetViewport(640, 480).Navigate("/")

	assert.Equal(t, 640, imageWidth(t, page.ScreenshotViewport()))
	assert.Equal(t, 640, imageWidth(t, page.ScreenshotFull()))
	assert.Positive(t, imageWidth(t, page.Screenshot("#drop")))
	assert.Equal(t, 640, imageWidth(t, page.ScreenshotJPEG(80)))
	assert.NotEmpty(t, page.ScreenshotWebP(80))
	assert.Equal(t, 20, imageWidth(t, page.ScreenshotRegion(0, 0, 20, 10, WithScreenshotTimeout(0))))
	assert.Positive(t, imageWidth(t, page.ScreenshotElementWithPadding("#drop", 4)))
	options := browser_provider_chromedp.DefaultScreenshotOptions()
	assert.Equal(t, 640, imageWidth(t, page.ScreenshotWithOptions(options)))
	first := page.ScreenshotViewport()
	assert.Zero(t, page.CompareScreenshots(first, first))

	requirePDF(t, page.PrintToPDF())
	requirePDF(t, page.PrintToPDFWithOptions(browser_provider_chromedp.PDFOptions{PageRanges: "1", Scale: 1}))
	requirePDF(t, page.PrintToPDFLandscape())
	requirePDF(t, page.PrintToPDFA4())
	requirePDF(t, page.PrintToPDFNoBackground())
	requirePDF(t, page.PrintToPDFWithHeaderFooter("<span>head</span>", "<span>foot</span>"))
	requirePDF(t, page.PrintToPDFPageRange("1"))

	directory := t.TempDir()
	require.NoError(t, page.outputSandbox.Close())
	output, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	page.outputSandbox = output
	page.SaveScreenshot("#drop", "element.png").SaveScreenshotViewport("viewport.png").SavePDF("page.pdf")
	for _, name := range []string{"element.png", "viewport.png", "page.pdf"} {
		info, statErr := os.Stat(filepath.Join(directory, name))
		require.NoError(t, statErr)
		assert.Positive(t, info.Size())
	}
	require.Empty(t, recorder.take())

	assert.InDelta(t, 1.0, page.CompareScreenshots(first, first[:len(first)-1]), 0)

	require.NoError(t, page.outputSandbox.Close())
	readOnly, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	page.outputSandbox = readOnly
	requireEachStepFails(t, recorder, []pageStep{
		{name: "SaveScreenshot", run: func() { page.SaveScreenshot("#drop", "denied.png") }},
		{name: "SaveScreenshotViewport", run: func() { page.SaveScreenshotViewport("denied.png") }},
		{name: "SavePDF", run: func() { page.SavePDF("denied.pdf") }},
	})
}

func TestPageEmulation(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")
	width := func() any { return page.EvalReturn("innerWidth") }

	page.EmulateDevice(Device{Name: "Narrow", Width: 300, Height: 500, Scale: 1, Mobile: true})
	assert.InDelta(t, 300, width(), 0)
	page.EmulateIPhone14().EmulateIPhone14Pro().EmulateIPhone13().EmulateIPad().EmulateIPadPro().EmulateGalaxyS9()
	page.EmulateMobile().EmulateTablet().EmulateDesktop4K().EmulateDesktopHD().EmulateViewportByName("mobile-s")
	page.ResetEmulation()
	require.Empty(t, recorder.take())

	failures := recorder.attempt(func() { page.EmulateViewportByName("no-such-size") })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "EmulateViewportByName")
}

func TestPageOutputReportsFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "ScreenshotViewport", run: func() { page.ScreenshotViewport() }},
		{name: "ScreenshotFull", run: func() { page.ScreenshotFull() }},
		{name: "SaveScreenshotViewport", run: func() { page.SaveScreenshotViewport("a.png") }},
		{name: "ScreenshotJPEG", run: func() { page.ScreenshotJPEG(80) }},
		{name: "ScreenshotWebP", run: func() { page.ScreenshotWebP(80) }},
		{name: "ScreenshotRegion", run: func() { page.ScreenshotRegion(0, 0, 1, 1) }},
		{name: "ScreenshotElementWithPadding", run: func() { page.ScreenshotElementWithPadding("#drop", 1) }},
		{name: "ScreenshotWithOptions", run: func() { page.ScreenshotWithOptions(browser_provider_chromedp.DefaultScreenshotOptions()) }},
		{name: "PrintToPDFWithOptions", run: func() { page.PrintToPDFWithOptions(browser_provider_chromedp.PDFOptions{}) }},
		{name: "PrintToPDFLandscape", run: func() { page.PrintToPDFLandscape() }},
		{name: "PrintToPDFA4", run: func() { page.PrintToPDFA4() }},
		{name: "PrintToPDFNoBackground", run: func() { page.PrintToPDFNoBackground() }},
		{name: "PrintToPDFWithHeaderFooter", run: func() { page.PrintToPDFWithHeaderFooter("", "") }},
		{name: "PrintToPDFPageRange", run: func() { page.PrintToPDFPageRange("1") }},
		{name: "PrintToPDF", run: func() { page.PrintToPDF() }},
		{name: "SavePDF", run: func() { page.SavePDF("a.pdf") }},
		{name: "SetViewport", run: func() { page.SetViewport(10, 10) }},
		{name: "EmulateDevice", run: func() { page.EmulateDevice(Device{Name: "x", Width: 1, Height: 1, Scale: 1}) }},
		{name: "ResetEmulation", run: func() { page.ResetEmulation() }},
		{name: "EmulateIPhone14Pro", run: func() { page.EmulateIPhone14Pro() }},
		{name: "EmulateIPhone14", run: func() { page.EmulateIPhone14() }},
		{name: "EmulateIPhone13", run: func() { page.EmulateIPhone13() }},
		{name: "EmulateIPadPro", run: func() { page.EmulateIPadPro() }},
		{name: "EmulateIPad", run: func() { page.EmulateIPad() }},
		{name: "EmulateGalaxyS9", run: func() { page.EmulateGalaxyS9() }},
		{name: "EmulateMobile", run: func() { page.EmulateMobile() }},
		{name: "EmulateTablet", run: func() { page.EmulateTablet() }},
		{name: "EmulateDesktopHD", run: func() { page.EmulateDesktopHD() }},
		{name: "EmulateDesktop4K", run: func() { page.EmulateDesktop4K() }},
		{name: "EmulateViewportByName", run: func() { page.EmulateViewportByName("mobile-s") }},
	})
}
