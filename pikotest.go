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

package piko

import (
	"testing"

	"piko.sh/piko/internal/pikotest/pikotest_domain"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
)

// ActionTester provides a test harness for server actions.
// Create one using NewActionTester with an ActionHandlerEntry.
type ActionTester = pikotest_domain.ActionTester

// ActionResultView wraps the result of an action invocation for assertions.
type ActionResultView = pikotest_domain.ActionResultView

// ActionResult holds the raw result of a server action invocation.
type ActionResult = pikotest_dto.ActionResult

// ComponentTester manages the testing of a compiled Piko component.
// Create one using NewComponentTester with your component's BuildAST function.
type ComponentTester = pikotest_domain.ComponentTester

// TestView wraps the result of a component render and provides methods to
// check state, metadata, and DOM structure.
type TestView = pikotest_domain.TestView

// ASTQueryResult wraps a set of AST nodes from a CSS selector query and
// provides fluent assertion methods.
type ASTQueryResult = pikotest_domain.ASTQueryResult

// BuildASTFunc is the signature of the generated BuildAST function in compiled
// components.
type BuildASTFunc = pikotest_dto.BuildASTFunc

// ComponentOption sets optional behaviour for the ComponentTester.
type ComponentOption = pikotest_dto.ComponentOption

// RequestBuilder builds RequestData for tests.
// Use NewTestRequest to create a builder.
type RequestBuilder = pikotest_domain.RequestBuilder

// NewActionTester creates a new ActionTester for the given action handler
// entry.
//
// Takes tb (testing.TB) which is the test instance for error reporting.
// Takes entry (ActionHandlerEntry) which describes the action to test.
//
// Returns *ActionTester ready for invoking and asserting.
//
// Example:
// tester := piko.NewActionTester(t, CustomerCreate)
// result := tester.Invoke(ctx, map[string]any{"name": "Acme Corp"})
// result.AssertSuccess()
func NewActionTester(tb testing.TB, entry ActionHandlerEntry) *ActionTester {
	return pikotest_domain.NewActionTester(tb, entry)
}

// NewTestRequest creates a new RequestBuilder for the specified HTTP method
// and path. This is the primary entry point for building test requests with
// proper context injection.
//
// Takes method (string) which specifies the HTTP method (GET, POST, etc.).
// Takes path (string) which specifies the request path.
//
// Returns *RequestBuilder which provides a fluent interface for building test
// requests.
//
// Example:
// request := piko.NewTestRequest("GET", "/customers").
//
//	WithQueryParam("sort", "desc").
//	Build(ctx)
func NewTestRequest(method, path string) *RequestBuilder {
	return pikotest_domain.NewRequest(method, path)
}

// NewComponentTester creates a new ComponentTester for the given component's
// BuildAST function.
//
// Example:
// import "myapp/dist/pages/customers"
//
//	func TestCustomersPage(t *testing.T) {
//	    tester := piko.NewComponentTester(t, customers.BuildAST)
//	    // ...
//	}
//
// Takes tb (testing.TB) which provides the test context.
// Takes buildAST (BuildASTFunc) which builds the component's AST.
// Takes opts (...ComponentOption) which configures the tester behaviour.
//
// Returns *ComponentTester which is ready to test the component.
func NewComponentTester(tb testing.TB, buildAST BuildASTFunc, opts ...ComponentOption) *ComponentTester {
	return pikotest_domain.NewComponentTester(tb, buildAST, opts...)
}

// WithRenderer attaches a RenderService to enable full HTML rendering in tests.
// Most tests should use AST queries instead, which do not require a renderer.
//
// Returns ComponentOption which configures the component with the renderer.
//
// Panics if called directly; use pikotest_domain.WithRenderer instead to
// avoid circular dependencies.
func WithRenderer(_ any) ComponentOption {
	panic("WithRenderer must be called via pikotest_domain.WithRenderer directly")
}

// WithPageID sets the page identifier for this component test.
//
// Takes pageID (string) which specifies the unique page identifier.
//
// Returns ComponentOption which configures the test with the given page ID.
func WithPageID(pageID string) ComponentOption {
	return pikotest_domain.WithPageID(pageID)
}
