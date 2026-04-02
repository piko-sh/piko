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

package browser

import (
	"fmt"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// Screenshot captures a PNG screenshot of an element.
//
// Takes selector (string) which identifies the element to capture.
//
// Returns []byte which contains the raw PNG image data.
func (p *Page) Screenshot(selector string) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotElement(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("Screenshot(%q) failed: %v", selector, err)
	}
	return buffer
}

// ScreenshotViewport captures a PNG screenshot of what is currently visible
// in the browser window.
//
// Returns []byte which contains the raw PNG image data.
func (p *Page) ScreenshotViewport() []byte {
	buffer, err := browser_provider_chromedp.ScreenshotViewport(p.actionCtx())
	if err != nil {
		p.t.Fatalf("ScreenshotViewport() failed: %v", err)
	}
	return buffer
}

// ScreenshotFull captures a PNG screenshot of the full page.
//
// Returns []byte which contains the raw PNG image data.
func (p *Page) ScreenshotFull() []byte {
	buffer, err := browser_provider_chromedp.ScreenshotFull(p.actionCtx())
	if err != nil {
		p.t.Fatalf("ScreenshotFull() failed: %v", err)
	}
	return buffer
}

// SaveScreenshot captures a screenshot of an element and saves it to a file.
// The path is relative to the output directory configured via WithOutputDir.
//
// Takes selector (string) which identifies the element to capture.
// Takes path (string) which specifies the relative file path for the
// screenshot.
//
// Returns *Page which allows method chaining.
func (p *Page) SaveScreenshot(selector, path string) *Page {
	buffer, err := browser_provider_chromedp.ScreenshotElement(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("SaveScreenshot(%q, %q) failed: %v", selector, path, err)
	}
	if err := p.outputSandbox.WriteFileAtomic(path, buffer, outputFilePermissions); err != nil {
		p.t.Fatalf("SaveScreenshot(%q, %q) failed to write file: %v", selector, path, err)
	}
	return p
}

// SaveScreenshotViewport captures a viewport screenshot and saves it to a file.
// The path is relative to the output directory configured via WithOutputDir.
//
// Takes path (string) which specifies the relative file path to save the
// screenshot.
//
// Returns *Page which allows method chaining.
func (p *Page) SaveScreenshotViewport(path string) *Page {
	buffer, err := browser_provider_chromedp.ScreenshotViewport(p.actionCtx())
	if err != nil {
		p.t.Fatalf("SaveScreenshotViewport(%q) failed: %v", path, err)
	}
	if err := p.outputSandbox.WriteFileAtomic(path, buffer, outputFilePermissions); err != nil {
		p.t.Fatalf("SaveScreenshotViewport(%q) failed to write file: %v", path, err)
	}
	return p
}

// ScreenshotJPEG captures a JPEG screenshot of the viewport.
//
// Takes quality (int) which specifies the image quality from 0 to 100, where
// 100 is best quality but largest file size.
//
// Returns []byte which contains the JPEG image data.
func (p *Page) ScreenshotJPEG(quality int) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotJPEG(p.actionCtx(), quality)
	if err != nil {
		p.t.Fatalf("ScreenshotJPEG(%d) failed: %v", quality, err)
	}
	return buffer
}

// ScreenshotWebP captures a WebP screenshot of the viewport with the
// specified quality.
//
// Takes quality (int) which specifies the image quality from 0 to 100,
// where 100 is best quality.
//
// Returns []byte which contains the encoded WebP image data.
func (p *Page) ScreenshotWebP(quality int) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotWebP(p.actionCtx(), quality)
	if err != nil {
		p.t.Fatalf("ScreenshotWebP(%d) failed: %v", quality, err)
	}
	return buffer
}

// ScreenshotRegion captures a screenshot of a specific area of the viewport.
// The coordinates are relative to the viewport, not the document.
//
// Takes x (float64) which is the distance from the left edge in pixels.
// Takes y (float64) which is the distance from the top edge in pixels.
// Takes width (float64) which is the width of the area in pixels.
// Takes height (float64) which is the height of the area in pixels.
//
// Returns []byte which contains the screenshot image data.
func (p *Page) ScreenshotRegion(x, y, width, height float64) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotRegion(p.actionCtx(), x, y, width, height)
	if err != nil {
		p.t.Fatalf("ScreenshotRegion(%.0f, %.0f, %.0f, %.0f) failed: %v", x, y, width, height, err)
	}
	return buffer
}

// ScreenshotElementWithPadding captures a screenshot of an element with extra
// padding around it.
//
// Takes selector (string) which identifies the element to capture.
// Takes padding (float64) which specifies the extra space around the element.
//
// Returns []byte which contains the screenshot image data.
func (p *Page) ScreenshotElementWithPadding(selector string, padding float64) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotElementWithPadding(p.actionCtx(), selector, padding)
	if err != nil {
		p.t.Fatalf("ScreenshotElementWithPadding(%q, %.0f) failed: %v", selector, padding, err)
	}
	return buffer
}

// ScreenshotWithOptions captures a screenshot with custom format and quality
// options.
//
// Takes opts (ScreenshotOptions) which specifies the image format and quality
// settings.
//
// Returns []byte which contains the screenshot image data.
func (p *Page) ScreenshotWithOptions(opts browser_provider_chromedp.ScreenshotOptions) []byte {
	buffer, err := browser_provider_chromedp.ScreenshotWithFormat(p.actionCtx(), opts)
	if err != nil {
		p.t.Fatalf("ScreenshotWithOptions() failed: %v", err)
	}
	return buffer
}

// CompareScreenshots compares two screenshots and returns the percentage of
// bytes that differ between them.
//
// Takes a ([]byte) which is the first screenshot to compare.
// Takes b ([]byte) which is the second screenshot to compare.
//
// Returns float64 which is 0.0 if the screenshots are the same, or 1.0 if
// they are completely different.
func (p *Page) CompareScreenshots(a, b []byte) float64 {
	diff, err := browser_provider_chromedp.CompareScreenshots(a, b)
	if err != nil {
		p.t.Fatalf("CompareScreenshots() failed: %v", err)
	}
	return diff
}

// PrintToPDF creates a PDF of the current page using default options.
//
// Returns []byte which contains the PDF data.
func (p *Page) PrintToPDF() []byte {
	buffer, err := browser_provider_chromedp.PrintToPDF(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PrintToPDF() failed: %v", err)
	}
	return buffer
}

// PrintToPDFWithOptions creates a PDF of the page with the given settings.
//
// Takes opts (browser_provider_chromedp.PDFOptions) which specifies the PDF
// output settings.
//
// Returns []byte which contains the PDF file data.
func (p *Page) PrintToPDFWithOptions(opts browser_provider_chromedp.PDFOptions) []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFWithOptions(p.actionCtx(), opts)
	if err != nil {
		p.t.Fatalf("PrintToPDFWithOptions() failed: %v", err)
	}
	return buffer
}

// PrintToPDFLandscape generates a landscape PDF of the current page.
//
// Returns []byte which contains the PDF document data.
func (p *Page) PrintToPDFLandscape() []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFLandscape(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PrintToPDFLandscape() failed: %v", err)
	}
	return buffer
}

// PrintToPDFA4 creates a PDF of the current page in A4 size.
//
// Returns []byte which contains the PDF document data.
func (p *Page) PrintToPDFA4() []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFA4(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PrintToPDFA4() failed: %v", err)
	}
	return buffer
}

// PrintToPDFNoBackground creates a PDF of the page without background images
// or colours.
//
// Returns []byte which contains the PDF document data.
func (p *Page) PrintToPDFNoBackground() []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFNoBackground(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PrintToPDFNoBackground() failed: %v", err)
	}
	return buffer
}

// PrintToPDFWithHeaderFooter creates a PDF with custom header and footer.
//
// Takes header (string) which sets the HTML content for the page header.
// Takes footer (string) which sets the HTML content for the page footer.
//
// Returns []byte which holds the PDF data.
func (p *Page) PrintToPDFWithHeaderFooter(header, footer string) []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFWithHeaderFooter(p.actionCtx(), header, footer)
	if err != nil {
		p.t.Fatalf("PrintToPDFWithHeaderFooter() failed: %v", err)
	}
	return buffer
}

// PrintToPDFPageRange generates a PDF of specific pages.
// Use pageRanges like "1-5, 8, 11-13" for specific pages.
//
// Takes pageRanges (string) which specifies pages to include in the PDF.
//
// Returns []byte which contains the generated PDF data.
func (p *Page) PrintToPDFPageRange(pageRanges string) []byte {
	buffer, err := browser_provider_chromedp.PrintToPDFPageRange(p.actionCtx(), pageRanges)
	if err != nil {
		p.t.Fatalf("PrintToPDFPageRange(%q) failed: %v", pageRanges, err)
	}
	return buffer
}

// SavePDF generates a PDF of the current page and saves it to a file. The path
// is relative to the output directory configured via WithOutputDir.
//
// Takes path (string) which specifies the relative file path where the PDF is
// saved.
//
// Returns *Page which allows method chaining.
func (p *Page) SavePDF(path string) *Page {
	buffer, err := browser_provider_chromedp.PrintToPDF(p.actionCtx())
	if err != nil {
		p.t.Fatalf("SavePDF(%q) failed: %v", path, err)
	}
	if err := p.outputSandbox.WriteFileAtomic(path, buffer, outputFilePermissions); err != nil {
		p.t.Fatalf("SavePDF(%q) failed to write file: %v", path, err)
	}
	return p
}

// Device represents a predefined device for emulation.
type Device struct {
	// Name is the display name of the device.
	Name string

	// Width is the screen width in pixels.
	Width int64

	// Height is the viewport height in pixels.
	Height int64

	// Scale is the device pixel ratio for high-density displays; 1.0 is standard.
	Scale float64

	// Mobile indicates whether to emulate a mobile device viewport.
	Mobile bool
}

var (
	// DeviceIPhone12 represents the iPhone 12 mobile device configuration.
	DeviceIPhone12 = Device{Name: "iPhone 12", Width: 390, Height: 844, Scale: 3, Mobile: true}

	// DeviceIPhone13 is a device profile for the iPhone 13 with a 390x844 display.
	DeviceIPhone13 = Device{Name: "iPhone 13", Width: 390, Height: 844, Scale: 3, Mobile: true}

	// DeviceIPhone14 represents an iPhone 14 device with standard dimensions.
	DeviceIPhone14 = Device{Name: "iPhone 14", Width: 390, Height: 844, Scale: 3, Mobile: true}

	// DevicePixel5 represents a Google Pixel 5 device with its screen dimensions.
	DevicePixel5 = Device{Name: "Pixel 5", Width: 393, Height: 851, Scale: 2.75, Mobile: true}

	// DevicePixel7 defines the screen dimensions for a Google Pixel 7 device.
	DevicePixel7 = Device{Name: "Pixel 7", Width: 412, Height: 915, Scale: 2.625, Mobile: true}

	// DeviceIPadPro11 defines the screen specifications for an iPad Pro 11 tablet.
	DeviceIPadPro11 = Device{Name: "iPad Pro 11", Width: 834, Height: 1194, Scale: 2, Mobile: true}

	// DeviceIPadPro129 defines the iPad Pro 12.9 inch device configuration.
	DeviceIPadPro129 = Device{Name: "iPad Pro 12.9", Width: 1024, Height: 1366, Scale: 2, Mobile: true}

	// DeviceIPadMini is a device preset for the iPad Mini with Retina display.
	DeviceIPadMini = Device{Name: "iPad Mini", Width: 768, Height: 1024, Scale: 2, Mobile: true}

	// DeviceDesktop1080p is a desktop viewport preset for 1080p resolution.
	DeviceDesktop1080p = Device{Name: "Desktop 1080p", Width: 1920, Height: 1080, Scale: 1, Mobile: false}

	// DeviceDesktop720p is a desktop device preset with 720p resolution.
	DeviceDesktop720p = Device{Name: "Desktop 720p", Width: 1280, Height: 720, Scale: 1, Mobile: false}

	// DeviceLaptop represents a standard laptop display with 1366x768 resolution.
	DeviceLaptop = Device{Name: "Laptop", Width: 1366, Height: 768, Scale: 1, Mobile: false}
)

// SetViewport sets the viewport width and height.
//
// Takes width (int64) which is the viewport width in pixels.
// Takes height (int64) which is the viewport height in pixels.
//
// Returns *Page which allows method chaining.
func (p *Page) SetViewport(width, height int64) *Page {
	detail := fmt.Sprintf("%dx%d", width, height)
	p.beforeAction("SetViewport", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetViewport(p.actionCtx(), width, height)
	p.afterAction("SetViewport", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetViewport(%d, %d) failed: %v", width, height, err)
	}
	return p
}

// EmulateDevice emulates a predefined device.
//
// Takes device (Device) which specifies the device to emulate.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateDevice(device Device) *Page {
	p.beforeAction("EmulateDevice", device.Name)
	start := time.Now()
	err := browser_provider_chromedp.EmulateDevice(p.actionCtx(), browser_provider_chromedp.DeviceParams{
		UserAgent: "",
		Width:     device.Width,
		Height:    device.Height,
		Scale:     device.Scale,
		Mobile:    device.Mobile,
	})
	p.afterAction("EmulateDevice", device.Name, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateDevice(%q) failed: %v", device.Name, err)
	}
	return p
}

// ResetEmulation resets device emulation to defaults.
//
// Returns *Page which allows method chaining.
func (p *Page) ResetEmulation() *Page {
	p.beforeAction("ResetEmulation", "")
	start := time.Now()
	err := browser_provider_chromedp.ResetEmulation(p.actionCtx())
	p.afterAction("ResetEmulation", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ResetEmulation() failed: %v", err)
	}
	return p
}

// EmulateIPhone14 sets the browser to act like an iPhone 14.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateIPhone14() *Page {
	p.beforeAction("EmulateIPhone14", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateIPhone14(p.actionCtx())
	p.afterAction("EmulateIPhone14", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateIPhone14() failed: %v", err)
	}
	return p
}

// EmulateIPhone14Pro sets up the page to behave like an iPhone 14 Pro.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateIPhone14Pro() *Page {
	p.beforeAction("EmulateIPhone14Pro", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateIPhone14Pro(p.actionCtx())
	p.afterAction("EmulateIPhone14Pro", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateIPhone14Pro() failed: %v", err)
	}
	return p
}

// EmulateIPhone13 sets the browser to behave like an iPhone 13.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateIPhone13() *Page {
	p.beforeAction("EmulateIPhone13", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateIPhone13(p.actionCtx())
	p.afterAction("EmulateIPhone13", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateIPhone13() failed: %v", err)
	}
	return p
}

// EmulateIPad sets the page to behave like an iPad device.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateIPad() *Page {
	p.beforeAction("EmulateIPad", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateIPad(p.actionCtx())
	p.afterAction("EmulateIPad", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateIPad() failed: %v", err)
	}
	return p
}

// EmulateIPadPro sets the page to behave like an iPad Pro 11-inch device.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateIPadPro() *Page {
	p.beforeAction("EmulateIPadPro", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateIPadPro(p.actionCtx())
	p.afterAction("EmulateIPadPro", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateIPadPro() failed: %v", err)
	}
	return p
}

// EmulateGalaxyS9 sets the page to behave like a Samsung Galaxy S9 device.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateGalaxyS9() *Page {
	p.beforeAction("EmulateGalaxyS9", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateGalaxyS9(p.actionCtx())
	p.afterAction("EmulateGalaxyS9", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateGalaxyS9() failed: %v", err)
	}
	return p
}

// EmulateMobile sets the page to behave like a mobile device (iPhone 13).
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateMobile() *Page {
	p.beforeAction("EmulateMobile", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateMobile(p.actionCtx())
	p.afterAction("EmulateMobile", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateMobile() failed: %v", err)
	}
	return p
}

// EmulateTablet sets the browser to behave like a tablet device (iPad).
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateTablet() *Page {
	p.beforeAction("EmulateTablet", "")
	start := time.Now()
	err := browser_provider_chromedp.EmulateTablet(p.actionCtx())
	p.afterAction("EmulateTablet", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateTablet() failed: %v", err)
	}
	return p
}

// EmulateDesktopHD sets the viewport to 1920x1080 (Full HD desktop size).
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateDesktopHD() *Page {
	p.beforeAction("EmulateDesktopHD", "1920x1080")
	start := time.Now()
	err := browser_provider_chromedp.EmulateDesktopHD(p.actionCtx())
	p.afterAction("EmulateDesktopHD", "1920x1080", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateDesktopHD() failed: %v", err)
	}
	return p
}

// EmulateDesktop4K sets a 3840x2160 desktop viewport (4K UHD).
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateDesktop4K() *Page {
	p.beforeAction("EmulateDesktop4K", "3840x2160")
	start := time.Now()
	err := browser_provider_chromedp.EmulateDesktop4K(p.actionCtx())
	p.afterAction("EmulateDesktop4K", "3840x2160", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateDesktop4K() failed: %v", err)
	}
	return p
}

// EmulateViewportByName sets the viewport to a common size by name.
//
// Valid names: "mobile-s", "mobile-m", "mobile-l", "tablet", "laptop",
// "laptop-l", "desktop", "desktop-4k".
//
// Takes sizeName (string) which specifies the viewport preset name.
//
// Returns *Page which allows method chaining.
func (p *Page) EmulateViewportByName(sizeName string) *Page {
	p.beforeAction("EmulateViewportByName", sizeName)
	start := time.Now()
	err := browser_provider_chromedp.EmulateViewportByName(p.actionCtx(), sizeName)
	p.afterAction("EmulateViewportByName", sizeName, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EmulateViewportByName(%q) failed: %v", sizeName, err)
	}
	return p
}
