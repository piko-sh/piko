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

	greeting "testcase_22_component_props/pages/pages_greeting_f99c4fc9"
)

func TestProps_DefaultProps(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST,
		piko.WithPageID("pages/greeting"),
	)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{})

	view.QueryAST(".name-value").HasText("Guest")
	view.QueryAST(".message").HasText("Hello, Guest!")

	view.QueryAST(".age").NotExists()

	view.QueryAST(".vip-badge").NotExists()

	view.QueryAST(".no-tags").Exists()
	view.QueryAST(".tags-section").NotExists()
}

func TestProps_WithName(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Alice",
	})

	view.QueryAST(".name-value").HasText("Alice")
	view.QueryAST(".message").HasText("Hello, Alice!")
}

func TestProps_WithGreeting(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name:     "Bob",
		Greeting: "Welcome",
	})

	view.QueryAST(".message").HasText("Welcome, Bob!")
}

func TestProps_WithAge(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Charlie",
		Age:  30,
	})

	view.QueryAST(".age").Exists()
	view.QueryAST(".age-value").HasText("30")
}

func TestProps_WithVIP(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name:  "Diana",
		IsVIP: true,
	})

	view.QueryAST(".vip-badge").Exists()
	view.QueryAST(".vip-badge").HasText("VIP Member")
}

func TestProps_WithTags(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Eve",
		Tags: []string{"developer", "golang", "web"},
	})

	view.QueryAST(".tags-section").Exists()
	view.QueryAST(".no-tags").NotExists()

	view.QueryAST(".tag-item").Count(3)
}

func TestProps_AllProps(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name:     "Frank",
		Age:      25,
		IsVIP:    true,
		Tags:     []string{"admin", "premium"},
		Greeting: "Greetings",
	})

	view.QueryAST(".message").HasText("Greetings, Frank!")
	view.QueryAST(".name-value").HasText("Frank")
	view.QueryAST(".age-value").HasText("25")
	view.QueryAST(".vip-badge").Exists()
	view.QueryAST(".tag-item").Count(2)
}

func TestProps_EmptyTags(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Grace",
		Tags: []string{},
	})

	view.QueryAST(".no-tags").Exists()
	view.QueryAST(".tags-section").NotExists()
}

func TestProps_ZeroAge(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Henry",
		Age:  0,
	})

	view.QueryAST(".age").NotExists()
}

func TestProps_StructureIntact(t *testing.T) {
	tester := piko.NewComponentTester(t, greeting.BuildAST)

	request := piko.NewTestRequest("GET", "/greeting").Build(context.Background())
	view := tester.Render(request, greeting.Props{
		Name: "Test",
	})

	view.QueryAST(".greeting-card").Exists()
	view.QueryAST(".greeting-card h1.message").Exists()
	view.QueryAST(".greeting-card .user-info").Exists()
	view.QueryAST(".user-info .name").Exists()
}
