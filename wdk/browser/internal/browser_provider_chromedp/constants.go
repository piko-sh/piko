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

import (
	"time"
)

const (
	// documentReadyStateScript names the embedded script that reads document.readyState.
	documentReadyStateScript = "document_ready_state.js"

	// ShadowDOMSeparator is the separator used for shadow DOM piercing selectors.
	ShadowDOMSeparator = " >>> "

	// DefaultExpectedStatus is the default HTTP status code expected for page loads.
	DefaultExpectedStatus = 200

	// NullAttributeValue is a marker string for a missing attribute in assertions.
	NullAttributeValue = "null"

	// DefaultActionTimeout is the time limit for user-interaction actions (Click, Focus,
	// Fill, etc.) that poll for an element to appear.
	DefaultActionTimeout = 5 * time.Second

	// DragActionTimeout is the time limit for drag-and-drop operations which involve
	// multiple sequential CDP round-trips.
	DragActionTimeout = 30 * time.Second

	// DefaultAssertionTimeout is the default time limit for assertion retries.
	DefaultAssertionTimeout = 5 * time.Second

	// DefaultPollingInterval is the default polling interval for retry loops.
	DefaultPollingInterval = 100 * time.Millisecond

	// ErrFmtFindingElement is the format string for errors when an element cannot be found.
	ErrFmtFindingElement = "finding element %s: %w"

	// ErrFmtElementNotFound is the error format for element not found errors.
	ErrFmtElementNotFound = "element not found: %s"

	// ErrFmtElementNotFoundShadow is the error format for shadow DOM element not found
	// errors.
	ErrFmtElementNotFoundShadow = "element not found in shadow DOM: %s"

	// ScreenshotQualityFull is the quality setting for full page screenshots.
	ScreenshotQualityFull = 100

	// DebugHTMLPreviewLength is the maximum length for HTML preview in debug output.
	DebugHTMLPreviewLength = 200

	// MinBodyLength is the smallest body length needed to consider a page loaded.
	MinBodyLength = 10

	// jsShadowDOMScrollIntoView is the JS template for scrolling a shadow DOM element into
	// view. Parameters: hostSelector, shadowSelector.
	jsShadowDOMScrollIntoView = `
		(() => {
			const host = document.querySelector(%q);
			if (!host || !host.shadowRoot) return false;
			const el = host.shadowRoot.querySelector(%q);
			if (!el) return false;
			el.scrollIntoView({ block: 'center', inline: 'center' });
			return true;
		})()`

	// ErrFmtScrollingShadowDOMElement is the error format for shadow DOM scroll failures.
	ErrFmtScrollingShadowDOMElement = "scrolling shadow DOM element into view %s: %w"
)

const (
	// domQueryTimeout is the short time limit for a single DOM read or write that should
	// finish almost at once.
	domQueryTimeout = 500 * time.Millisecond

	// stableCheckInterval is how often WaitStable compares the page HTML when waiting for
	// the DOM to settle after a navigation.
	stableCheckInterval = 500 * time.Millisecond

	// stableMaxWait is the longest WaitStable waits for the DOM to settle.
	stableMaxWait = 5 * time.Second

	// historyNavigationTimeout is the time limit for going back or forward in the browser
	// history.
	historyNavigationTimeout = 5 * time.Second

	// networkIdleGrace is the extra time WaitForNetworkIdle allows beyond its in-page
	// timeout before giving up on a page that never answers.
	networkIdleGrace = 2 * time.Second

	// navigationTimeout is the time limit for loading a page by URL.
	navigationTimeout = 15 * time.Second

	// visibleImagesTimeout is the time limit for images in the viewport to finish loading
	// while scrolling a page for capture.
	visibleImagesTimeout = 5 * time.Second

	// scrollResetDelay is the pause after scrolling back to the top so the page can repaint.
	scrollResetDelay = 100 * time.Millisecond

	// unwrapSettleDelay is the pause after removing scroll container limits so the page can
	// reflow.
	unwrapSettleDelay = 200 * time.Millisecond

	// downloadStartDelay is the pause before polling for a downloaded file, giving the
	// browser time to begin writing it.
	downloadStartDelay = 500 * time.Millisecond

	// pageCreationBackoffStep is the delay added per attempt when retrying page creation.
	pageCreationBackoffStep = 100 * time.Millisecond

	// pageInitialiseDelay is the pause after creating a page so its targets are ready to
	// use.
	pageInitialiseDelay = 200 * time.Millisecond

	// consoleClearDelay is the pause after clearing the console so in-flight log events
	// arrive before the stored logs are reset.
	consoleClearDelay = 50 * time.Millisecond

	// serverDialTimeout is the time limit for one TCP dial while waiting for the server to
	// accept connections.
	serverDialTimeout = 50 * time.Millisecond

	// formattedHTMLMaxLineLength is the line length used when formatting HTML for output.
	formattedHTMLMaxLineLength = 120

	// tmplKeySelector is the script template key for a CSS selector.
	tmplKeySelector = "Selector"

	// tmplKeyFrameSelector is the script template key for an iframe selector.
	tmplKeyFrameSelector = "FrameSelector"

	// tmplKeyElementSelector is the script template key for a selector inside an iframe.
	tmplKeyElementSelector = "ElementSelector"

	// tmplKeyEventName is the script template key for an event name.
	tmplKeyEventName = "EventName"

	// tmplKeyDetailJSON is the script template key for an event detail as JSON.
	tmplKeyDetailJSON = "DetailJSON"

	// tmplKeyHost is the script template key for a shadow host selector.
	tmplKeyHost = "Host"

	// tmplKeyShadow is the script template key for a selector inside a shadow root.
	tmplKeyShadow = "Shadow"

	// tmplKeyStorageType is the script template key for the web storage type.
	tmplKeyStorageType = "StorageType"

	// tmplKeyKey is the script template key for a storage key.
	tmplKeyKey = "Key"

	// tmplKeyAttrName is the script template key for an attribute name.
	tmplKeyAttrName = "AttrName"
)
