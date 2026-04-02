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

// This file re-exports types and functions from the internal browser provider
// so that external consumers (e.g. integration tests) can access them through
// the public wdk/browser API without importing the internal package directly.

import bpc "piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"

// ActionContext holds the context needed for browser action execution.
type ActionContext = bpc.ActionContext

// Browser wraps a chromedp browser instance and manages its lifecycle.
type Browser = bpc.Browser

// BrowserOptions holds settings for creating a browser instance.
type BrowserOptions = bpc.BrowserOptions

// BrowserPool manages multiple Browser instances and distributes page creation
// across them via round-robin for multi-process parallelism.
type BrowserPool = bpc.BrowserPool

// BrowserPoolConfig configures optional behaviour of a BrowserPool.
type BrowserPoolConfig = bpc.BrowserPoolConfig

// ExclusiveBrowserPool manages a pool of Browser instances where each browser
// is checked out to at most one consumer at a time, guaranteeing single-tab
// exclusivity for focus, blur, and mouse-capture sensitive tests.
type ExclusiveBrowserPool = bpc.ExclusiveBrowserPool

// IncognitoPage wraps a page with its incognito browser context for proper
// cleanup.
type IncognitoPage = bpc.IncognitoPage

// NetworkMock defines a mock network response for testing.
type NetworkMock = bpc.NetworkMock

// NormaliseOptions controls how DOM normalisation is performed.
type NormaliseOptions = bpc.NormaliseOptions

// PageHelper wraps a chromedp context with additional helper methods for E2E
// testing.
type PageHelper = bpc.PageHelper

// PartialConfig defines configuration for a partial with staged variants.
type PartialConfig = bpc.PartialConfig

// PartialStage defines a single stage variant for a partial.
type PartialStage = bpc.PartialStage

// PKCComponentConfig defines a PKC component to verify on the page.
type PKCComponentConfig = bpc.PKCComponentConfig

// LoadTestSpecOption configures the behaviour of LoadTestSpec.
type LoadTestSpecOption = bpc.LoadTestSpecOption

// DownloadTrackerOption configures a DownloadTracker.
type DownloadTrackerOption = bpc.DownloadTrackerOption

var (
	// NewBrowser is an alias for bpc.NewBrowser that creates a new browser
	// instance.
	NewBrowser = bpc.NewBrowser

	// NewBrowserPool is an alias for bpc.NewBrowserPool that creates a pool of
	// browsers.
	NewBrowserPool = bpc.NewBrowserPool

	// NewExclusiveBrowserPool creates a pool where each browser is checked out
	// exclusively.
	NewExclusiveBrowserPool = bpc.NewExclusiveBrowserPool

	// DefaultPoolSize returns a sensible default pool size based on CPU count.
	DefaultPoolSize = bpc.DefaultPoolSize

	// DefaultBrowserOptions is the default browser configuration from bpc.
	DefaultBrowserOptions = bpc.DefaultBrowserOptions

	// NewPageHelper is an alias for bpc.NewPageHelper.
	NewPageHelper = bpc.NewPageHelper

	// Click is an alias for bpc.Click.
	Click = bpc.Click

	// Fill is an alias for bpc.Fill.
	Fill = bpc.Fill

	// Eval is a shorthand alias for bpc.Eval.
	Eval = bpc.Eval

	// FindElements is an alias for bpc.FindElements.
	FindElements = bpc.FindElements

	// GetElementText retrieves the text content from an XML element.
	GetElementText = bpc.GetElementText

	// GetElementHTML is a helper that extracts HTML content from a DOM element.
	GetElementHTML = bpc.GetElementHTML

	// GetElementAttribute is an alias for bpc.GetElementAttribute.
	GetElementAttribute = bpc.GetElementAttribute

	// GetElementValue is an alias to bpc.GetElementValue.
	GetElementValue = bpc.GetElementValue

	// EvalOnElement is a re-export of bpc.EvalOnElement.
	EvalOnElement = bpc.EvalOnElement

	// SetViewport is an alias for the browser protocol command to set the
	// viewport.
	SetViewport = bpc.SetViewport

	// WaitForSelector is a browser page control for waiting until a selector
	// matches.
	WaitForSelector = bpc.WaitForSelector

	// WaitForServerReady is an alias for bpc.WaitForServerReady.
	WaitForServerReady = bpc.WaitForServerReady

	// FindAvailablePort is an alias for bpc.FindAvailablePort.
	FindAvailablePort = bpc.FindAvailablePort

	// ExecuteStep is an alias for bpc.ExecuteStep.
	ExecuteStep = bpc.ExecuteStep

	// ExecuteAssertion is an alias for bpc.ExecuteAssertion.
	ExecuteAssertion = bpc.ExecuteAssertion

	// IsAssertionAction is a predicate that checks if a string is an assertion
	// action.
	IsAssertionAction = bpc.IsAssertionAction

	// LoadTestSpec is an alias for bpc.LoadTestSpec.
	LoadTestSpec = bpc.LoadTestSpec

	// WithTestSpecSandboxFactory sets a factory for creating sandboxes when
	// loading a test spec file.
	WithTestSpecSandboxFactory = bpc.WithTestSpecSandboxFactory

	// WithDownloadSandboxFactory sets a factory for creating sandboxes in the
	// download tracker.
	WithDownloadSandboxFactory = bpc.WithDownloadSandboxFactory

	// CaptureDOM is an alias for bpc.CaptureDOM.
	CaptureDOM = bpc.CaptureDOM

	// NormaliseDOM is a function that normalises HTML DOM structures.
	NormaliseDOM = bpc.NormaliseDOM

	// DefaultNormaliseOptions provides the standard normalisation settings from
	// bpc.
	DefaultNormaliseOptions = bpc.DefaultNormaliseOptions

	// GetFormData returns form data from a form element as a map.
	GetFormData = bpc.GetFormData

	// ListenForEvent attaches an event listener to capture e.detail.
	ListenForEvent = bpc.ListenForEvent

	// GetEventDetail returns the event detail captured by a prior ListenForEvent
	// call.
	GetEventDetail = bpc.GetEventDetail
)
