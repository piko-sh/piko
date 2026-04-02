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

// Package browser provides a fluent API for writing browser-based
// end-to-end tests for Piko applications. It supports both
// programmatic testing with method chaining and declarative
// spec-based testing using JSON files.
//
// # Setup
//
// Create a TestMain to initialise the test harness with the e2e
// build tag:
//
//	//go:build e2e
//
//	var harness *browser.Harness
//
//	func TestMain(m *testing.M) {
//	    harness = browser.NewHarness(browser.WithProjectDir("."))
//	    if err := harness.Setup(); err != nil {
//	        fmt.Fprintf(os.Stderr, "E2E setup failed: %v\n", err)
//	        os.Exit(1)
//	    }
//	    code := m.Run()
//	    harness.Cleanup()
//	    os.Exit(code)
//	}
//
// # Programmatic tests
//
// Write tests using the fluent [Page] API. All action methods return
// *Page for method chaining:
//
//	func TestCalculator(t *testing.T) {
//	    p := browser.New(t)
//	    defer p.Close()
//
//	    p.Navigate("/calc").
//	        Fill("#num1", "5").
//	        Fill("#num2", "3").
//	        Click("#add")
//
//	    p.Assert("#result").HasText("8")
//	}
//
// # Spec-based tests
//
// Use JSON test specifications for declarative testing:
//
//	func TestHomepage(t *testing.T) {
//	    browser.RunSpec(t, "testdata/homepage.json")
//	}
//
// # Page
//
// The [Page] type covers navigation, element interaction, waiting,
// assertions via an [Assertion] builder, screenshots, storage and
// cookie access, network interception, iframe support, and
// Piko-specific operations such as partial reloads and bus events.
//
// # Thread safety
//
// [Harness] and [New] are safe for concurrent use. Each test
// receives its own isolated [Page] backed by an incognito browser
// context, so tests can run in parallel without interference.
package browser
