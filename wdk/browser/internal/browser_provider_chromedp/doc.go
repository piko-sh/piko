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

// Package browser_provider_chromedp wraps chromedp to provide Piko-aware
// browser automation for end-to-end testing.
//
// It is shared between the internal integration test suite
// (tests/integration/e2e_browser) and the public test API
// (pkg/e2etest). It handles browser lifecycle management, DOM utilities, action
// and assertion implementations, and JSON-driven test specifications. Shadow
// DOM piercing, partial reload triggers, and event bus interactions are
// handled transparently via the " >>> " selector syntax and embedded
// JavaScript templates.
//
// # Actions and assertions
//
// Browser actions (click, fill, navigate, press, etc.) are dispatched via
// [ExecuteStep], whilst assertions (checkText, checkAttribute, checkVisible,
// etc.) are dispatched via [ExecuteAssertion]. Both use map-based dispatch
// tables keyed by action name strings.
//
// All selectors support shadow DOM piercing with the " >>> " syntax:
//
//	"my-component >>> .inner-element"
//
// # Thread safety
//
// [Browser] and [PageHelper] are safe for concurrent use. Browser guards page
// creation with a mutex, and PageHelper protects console log access with its
// own mutex. Individual action and assertion functions are not inherently
// thread-safe and should be called from a single goroutine per page context.
package browser_provider_chromedp
