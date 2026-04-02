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

	messages "testcase_09_text_assertions/pages/pages_messages_2724f70a"
)

func TestTextAssertions_HasText(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST,
		piko.WithPageID("pages/messages"),
	)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".header").HasText("Welcome to the Message Board")

	view.QueryAST(".footer").HasText("End of messages - Copyright 2024")
}

func TestTextAssertions_ContainsText(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".header").ContainsText("Message Board")
	view.QueryAST(".header").ContainsText("Welcome")

	view.QueryAST(".footer").ContainsText("Copyright")
	view.QueryAST(".footer").ContainsText("2024")
}

func TestTextAssertions_MessageContent(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".author").Count(5)

	view.QueryAST(".author").Index(0).HasText("Alice")

	view.QueryAST(".author").Index(1).HasText("Bob")

	view.QueryAST(".content").Index(0).HasText("Hello, World!")
	view.QueryAST(".content").Index(1).ContainsText("quick brown fox")
}

func TestTextAssertions_Pangram(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	pangram := view.QueryAST(".content").Index(1)
	pangram.HasText("The quick brown fox jumps over the lazy dog.")
	pangram.ContainsText("fox")
	pangram.ContainsText("dog")
}

func TestTextAssertions_SpecialCharacters(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	specialContent := view.QueryAST(".content").Index(3)
	specialContent.ContainsText("Special chars")
}

func TestTextAssertions_IndexWithText(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	ids := view.QueryAST(".id")
	ids.Count(5)

	ids.Index(0).ContainsText("ID:")
	ids.Index(0).ContainsText("1")

	ids.Index(4).ContainsText("ID:")
	ids.Index(4).ContainsText("5")
}

func TestTextAssertions_ChainedTextChecks(t *testing.T) {
	tester := piko.NewComponentTester(t, messages.BuildAST)

	request := piko.NewTestRequest("GET", "/messages").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".header").
		Exists().
		AssertCount(1).
		ContainsText("Welcome").
		ContainsText("Message Board")
}
