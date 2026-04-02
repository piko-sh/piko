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

package browser_provider_chromedp

import "time"

const (
	// ShadowDOMSeparator is the separator used for shadow DOM piercing selectors.
	ShadowDOMSeparator = " >>> "

	// DefaultExpectedStatus is the default HTTP status code expected for page
	// loads.
	DefaultExpectedStatus = 200

	// NullAttributeValue is a marker string for a missing attribute in assertions.
	NullAttributeValue = "null"

	// DefaultActionTimeout is the time limit for user-interaction actions
	// (Click, Focus, Fill, etc.) that poll for an element to appear.
	DefaultActionTimeout = 5 * time.Second

	// DragActionTimeout is the time limit for drag-and-drop operations which
	// involve multiple sequential CDP round-trips.
	DragActionTimeout = 30 * time.Second

	// DefaultAssertionTimeout is the default time limit for assertion retries.
	DefaultAssertionTimeout = 5 * time.Second

	// DefaultPollingInterval is the default polling interval for retry loops.
	DefaultPollingInterval = 100 * time.Millisecond

	// ErrFmtFindingElement is the format string for errors when an element
	// cannot be found.
	ErrFmtFindingElement = "finding element %s: %w"

	// ErrFmtElementNotFound is the error format for element not found errors.
	ErrFmtElementNotFound = "element not found: %s"

	// ErrFmtElementNotFoundShadow is the error format for shadow DOM element not
	// found errors.
	ErrFmtElementNotFoundShadow = "element not found in shadow DOM: %s"

	// ScreenshotQualityFull is the quality setting for full page screenshots.
	ScreenshotQualityFull = 100

	// DebugHTMLPreviewLength is the maximum length for HTML preview in debug
	// output.
	DebugHTMLPreviewLength = 200

	// MinBodyLength is the smallest body length needed to consider a page loaded.
	MinBodyLength = 10

	// jsShadowDOMScrollIntoView is the JS template for scrolling a shadow DOM
	// element into view.
	// Parameters: hostSelector, shadowSelector.
	jsShadowDOMScrollIntoView = `
		(() => {
			const host = document.querySelector(%q);
			if (!host || !host.shadowRoot) return false;
			const el = host.shadowRoot.querySelector(%q);
			if (!el) return false;
			el.scrollIntoView({ block: 'center', inline: 'center' });
			return true;
		})()`

	// ErrFmtScrollingShadowDOMElement is the error format for shadow DOM scroll
	// failures.
	ErrFmtScrollingShadowDOMElement = "scrolling shadow DOM element into view %s: %w"
)
