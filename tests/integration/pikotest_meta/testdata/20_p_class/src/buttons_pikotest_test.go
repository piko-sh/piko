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

package main_test

import (
	"context"
	"testing"

	"piko.sh/piko"

	buttons "testcase_20_p_class/pages/pages_buttons_5ace0ecd"
)

func TestPClass_DefaultState(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST,
		piko.WithPageID("pages/buttons"),
	)

	request := piko.NewTestRequest("GET", "/buttons").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("button.btn").Count(3)

	view.QueryAST("button.btn").Index(0).HasAttributePresent("p-class:active")
	view.QueryAST("button.btn").Index(0).HasAttributePresent("p-class:disabled")

	view.QueryAST(".status-badge").HasAttributePresent("p-class:active")
	view.QueryAST(".status-badge").HasAttributePresent("p-class:inactive")
}

func TestPClass_ActiveButton(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?active=true").
		WithQueryParam("active", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	firstButton := view.QueryAST("button.btn").Index(0)
	firstButton.HasAttributePresent("p-class:active")
	firstButton.HasAttribute("p-class:active", "state.IsActive")

	view.QueryAST(".status-badge").HasAttribute("p-class:active", "state.IsActive")
}

func TestPClass_DisabledButton(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?disabled=true").
		WithQueryParam("disabled", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("button.btn").Index(0).HasAttribute("p-class:disabled", "state.IsDisabled")
}

func TestPClass_PrimaryButton(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?primary=true").
		WithQueryParam("primary", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("button.btn").Index(1).HasAttribute("p-class:primary", "state.IsPrimary")
}

func TestPClass_LargeButton(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?large=true").
		WithQueryParam("large", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("button.btn").Index(1).HasAttribute("p-class:large", "state.IsLarge")
}

func TestPClass_StatusSuccess(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?status=success").
		WithQueryParam("status", "success").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	thirdButton := view.QueryAST("button.btn").Index(2)
	thirdButton.HasAttributeContaining("p-class:success", "state.Status")
	thirdButton.HasAttributeContaining("p-class:warning", "state.Status")
	thirdButton.HasAttribute("p-class:error", "state.HasError")
}

func TestPClass_MultipleClasses(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons?active=true&disabled=true").
		WithQueryParam("active", "true").
		WithQueryParam("disabled", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	firstButton := view.QueryAST("button.btn").Index(0)
	firstButton.HasAttributePresent("p-class:active")
	firstButton.HasAttributePresent("p-class:disabled")
}

func TestPClass_NegatedCondition(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	request := piko.NewTestRequest("GET", "/buttons").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".status-badge").HasAttribute("p-class:inactive", "!state.IsActive")
}

func TestPClass_BaseClassPreserved(t *testing.T) {
	tester := piko.NewComponentTester(t, buttons.BuildAST)

	testCases := []string{
		"/buttons",
		"/buttons?active=true",
		"/buttons?primary=true&large=true",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			request := piko.NewTestRequest("GET", path).Build(context.Background())
			view := tester.Render(request, piko.NoProps{})

			view.QueryAST("button.btn").Count(3)

			view.QueryAST(".status-badge").Exists()
		})
	}
}
