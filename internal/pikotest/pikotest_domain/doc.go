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

// Package pikotest_domain supplies testing utilities for validating Piko
// components and server actions without requiring a running server.
//
// It enables fast, isolated unit tests through AST-based assertions and
// fluent request building.
//
// # Usage
//
// Test a component by creating a ComponentTester with its BuildAST
// function:
//
//	func TestCustomersPage(t *testing.T) {
//	    tester := pikotest_domain.NewComponentTester(t, customers.BuildAST)
//
//	    request := pikotest_domain.NewRequest("GET", "/customers").
//	        WithQueryParam("sort", "desc").
//	        Build(ctx)
//
//	    view := tester.Render(request, customers.NoProps{})
//
//	    view.QueryAST("h1").HasText("Customers")
//	    view.QueryAST(".customer-row").Count(10)
//	}
//
// # Thread safety
//
// Testers are not safe for concurrent use. Create a new tester for
// each test function or subtest.
package pikotest_domain
