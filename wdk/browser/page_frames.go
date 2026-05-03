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
	"piko.sh/piko/wdk/safedisk"
)

// GetFrames returns details about all frames in the page.
//
// Returns []browser_provider_chromedp.FrameInfo which holds frame details.
func (p *Page) GetFrames() []browser_provider_chromedp.FrameInfo {
	frames, err := browser_provider_chromedp.GetFrames(p.actionCtx())
	if err != nil {
		p.t.Fatalf("GetFrames() failed: %v", err)
	}
	return frames
}

// CountFrames returns the number of iframes in the page.
//
// Returns int which is the count of iframes found.
func (p *Page) CountFrames() int {
	count, err := browser_provider_chromedp.CountFrames(p.actionCtx())
	if err != nil {
		p.t.Fatalf("CountFrames() failed: %v", err)
	}
	return count
}

// ClickInFrame clicks an element within an iframe.
//
// Takes frameSelector (string) which identifies the iframe to target.
// Takes elementSelector (string) which identifies the element to click within
// the iframe.
//
// Returns *Page which allows method chaining for fluent test composition.
func (p *Page) ClickInFrame(frameSelector, elementSelector string) *Page {
	detail := fmt.Sprintf("%s > %s", frameSelector, elementSelector)
	p.beforeAction("ClickInFrame", detail)
	start := time.Now()
	err := browser_provider_chromedp.ClickInFrame(p.actionCtx(), frameSelector, elementSelector)
	p.afterAction("ClickInFrame", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClickInFrame(%q, %q) failed: %v", frameSelector, elementSelector, err)
	}
	return p
}

// FillInFrame fills an input element within an iframe.
//
// Takes frameSelector (string) which identifies the iframe to target.
// Takes elementSelector (string) which identifies the input element within the
// iframe.
// Takes value (string) which specifies the text to enter into the element.
//
// Returns *Page which allows method chaining for fluent test assertions.
func (p *Page) FillInFrame(frameSelector, elementSelector, value string) *Page {
	detail := fmt.Sprintf("%s > %s = %q", frameSelector, elementSelector, value)
	p.beforeAction("FillInFrame", detail)
	start := time.Now()
	err := browser_provider_chromedp.FillInFrame(p.actionCtx(), frameSelector, elementSelector, value)
	p.afterAction("FillInFrame", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("FillInFrame(%q, %q, %q) failed: %v", frameSelector, elementSelector, value, err)
	}
	return p
}

// GetTextInFrame gets the text content of an element within an iframe.
//
// Takes frameSelector (string) which identifies the iframe element.
// Takes elementSelector (string) which identifies the element within the frame.
//
// Returns string which is the text content of the element.
func (p *Page) GetTextInFrame(frameSelector, elementSelector string) string {
	text, err := browser_provider_chromedp.GetTextInFrame(p.actionCtx(), frameSelector, elementSelector)
	if err != nil {
		p.t.Fatalf("GetTextInFrame(%q, %q) failed: %v", frameSelector, elementSelector, err)
	}
	return text
}

// WaitForElementInFrame waits for an element to appear within an iframe.
//
// Takes frameSelector (string) which identifies the iframe to search within.
// Takes elementSelector (string) which identifies the element to wait for.
//
// Returns *Page which allows method chaining for fluent test syntax.
func (p *Page) WaitForElementInFrame(frameSelector, elementSelector string) *Page {
	detail := fmt.Sprintf("%s > %s", frameSelector, elementSelector)
	p.beforeAction("WaitForElementInFrame", detail)
	start := time.Now()
	err := browser_provider_chromedp.WaitForElementInFrame(p.actionCtx(), frameSelector, elementSelector)
	p.afterAction("WaitForElementInFrame", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForElementInFrame(%q, %q) failed: %v", frameSelector, elementSelector, err)
	}
	return p
}

// GetFrameDocument returns the HTML content of an iframe's document.
//
// Takes frameSelector (string) which identifies the iframe element to query.
//
// Returns string which contains the HTML content of the iframe's document.
func (p *Page) GetFrameDocument(frameSelector string) string {
	html, err := browser_provider_chromedp.GetFrameDocument(p.actionCtx(), frameSelector)
	if err != nil {
		p.t.Fatalf("GetFrameDocument(%q) failed: %v", frameSelector, err)
	}
	return html
}

// IsFrameLoaded checks if an iframe has finished loading.
//
// Takes frameSelector (string) which specifies the CSS selector for the iframe.
//
// Returns bool which is true if the iframe has finished loading.
func (p *Page) IsFrameLoaded(frameSelector string) bool {
	loaded, err := browser_provider_chromedp.IsFrameLoaded(p.actionCtx(), frameSelector)
	if err != nil {
		p.t.Fatalf("IsFrameLoaded(%q) failed: %v", frameSelector, err)
	}
	return loaded
}

// EvalInFrame evaluates JavaScript within a specific iframe.
//
// Takes frameSelector (string) which identifies the iframe to target.
// Takes script (string) which contains the JavaScript code to execute.
//
// Returns any which is the result of the JavaScript evaluation.
func (p *Page) EvalInFrame(frameSelector, script string) any {
	displayScript := truncateRunes(script, displayTextMaxLen)
	detail := fmt.Sprintf("%s: %s", frameSelector, displayScript)
	p.beforeAction("EvalInFrame", detail)
	start := time.Now()
	result, err := browser_provider_chromedp.EvalInFrame(p.actionCtx(), frameSelector, script)
	p.afterAction("EvalInFrame", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EvalInFrame(%q) failed: %v", frameSelector, err)
	}
	return result
}

// SetDownloadPath sets the download directory for future downloads.
//
// Takes path (string) which specifies the directory path for saved files.
//
// Returns *Page which allows method chaining.
func (p *Page) SetDownloadPath(path string) *Page {
	p.beforeAction("SetDownloadPath", path)
	start := time.Now()
	err := browser_provider_chromedp.SetDownloadPath(p.actionCtx(), path)
	p.afterAction("SetDownloadPath", path, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetDownloadPath(%q) failed: %v", path, err)
	}
	return p
}

// WaitForDownload waits for a download after triggering an action.
//
// Takes downloadDir (string) which specifies where the download is saved.
// Takes trigger (func(...)) which initiates the download (e.g. clicking a
// download link).
//
// Returns *browser_provider_chromedp.DownloadInfo which contains details about
// the downloaded
// file.
func (p *Page) WaitForDownload(downloadDir string, trigger func()) *browser_provider_chromedp.DownloadInfo {
	p.beforeAction("WaitForDownload", downloadDir)
	start := time.Now()
	info, err := browser_provider_chromedp.WaitForDownload(p.actionCtx(), downloadDir, 30*time.Second, func() error {
		trigger()
		return nil
	})
	p.afterAction("WaitForDownload", downloadDir, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForDownload(%q) failed: %v", downloadDir, err)
	}
	return info
}

// GetDownloadedFile reads the contents of a downloaded file using a sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides access to the download
// directory.
// Takes filename (string) which is the name of the downloaded file, not a full
// path.
//
// Returns []byte which contains the file contents.
func (p *Page) GetDownloadedFile(sandbox safedisk.Sandbox, filename string) []byte {
	data, err := browser_provider_chromedp.GetDownloadedFile(sandbox, filename)
	if err != nil {
		p.t.Fatalf("GetDownloadedFile(%q) failed: %v", filename, err)
	}
	return data
}

// TriggerDownload triggers a file download via JavaScript.
//
// Takes url (string) which specifies the URL of the file to download.
// Takes filename (string) which specifies the name to save the file as.
//
// Returns *Page which allows method chaining.
func (p *Page) TriggerDownload(url, filename string) *Page {
	detail := fmt.Sprintf("%s as %s", url, filename)
	p.beforeAction("TriggerDownload", detail)
	start := time.Now()
	err := browser_provider_chromedp.TriggerDownload(p.actionCtx(), url, filename)
	p.afterAction("TriggerDownload", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("TriggerDownload(%q, %q) failed: %v", url, filename, err)
	}
	return p
}

// CreateBlobDownload creates and triggers a download from blob data.
//
// Takes content (string) which provides the data to download.
// Takes mimeType (string) which specifies the MIME type of the content.
// Takes filename (string) which sets the suggested download filename.
//
// Returns *Page which allows method chaining for fluent test assertions.
func (p *Page) CreateBlobDownload(content, mimeType, filename string) *Page {
	detail := fmt.Sprintf("%s (%s)", filename, mimeType)
	p.beforeAction("CreateBlobDownload", detail)
	start := time.Now()
	err := browser_provider_chromedp.CreateBlobDownload(p.actionCtx(), content, mimeType, filename)
	p.afterAction("CreateBlobDownload", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("CreateBlobDownload(%q, %q) failed: %v", filename, mimeType, err)
	}
	return p
}
