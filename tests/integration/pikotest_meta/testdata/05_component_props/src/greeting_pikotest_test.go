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

	greeting "testcase_05_component_props/pages/pages_greeting_f99c4fc9"
)

func TestComponentProps_WithName(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST,
		piko.WithPageID("pages/greeting"),
	)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())

	props := greeting.Props{
		Name:  "Alice",
		Age:   25,
		IsVIP: false,
	}

	view := tester.Render(request, props)

	view.AssertTitle("Greeting for Alice")

	view.QueryAST("h1.greeting").Exists().HasText("Hello, Alice!")
}

func TestComponentProps_WithVIP(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())

	props := greeting.Props{
		Name:  "Bob",
		Age:   30,
		IsVIP: true,
	}

	view := tester.Render(request, props)

	view.QueryAST(".vip-badge").Exists().HasText("VIP Member")
}

func TestComponentProps_NonVIP(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())

	props := greeting.Props{
		Name:  "Charlie",
		Age:   20,
		IsVIP: false,
	}

	view := tester.Render(request, props)

	view.QueryAST(".vip-badge").Exists().HasText("Standard")
}

func TestComponentProps_DOMStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())

	props := greeting.Props{
		Name:  "Test",
		Age:   1,
		IsVIP: false,
	}

	view := tester.Render(request, props)

	view.QueryAST("div.greeting-card").Exists()
	view.QueryAST("h1.greeting").Exists()
	view.QueryAST("p.age").Exists()
	view.QueryAST("span.vip-badge").Exists()
}
