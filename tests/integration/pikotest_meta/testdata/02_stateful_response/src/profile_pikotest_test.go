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

	profile "testcase_02_stateful_response/pages/pages_profile_4331497b"
)

func TestStatefulResponse_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST,
		piko.WithPageID("pages/profile"),
	)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("User Profile")
}

func TestStatefulResponse_Description(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertDescription("View user profile details")
}

func TestStatefulResponse_Username(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	username := view.QueryAST("h1.username")
	username.Exists()
	username.HasText("johndoe")
}

func TestStatefulResponse_Email(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	email := view.QueryAST("span.email")
	email.Exists()
	email.HasText("john@example.com")
}

func TestStatefulResponse_MemberSince(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	stats := view.QueryAST(".stat-value")
	stats.Count(2)

	stats.Index(0).HasText("2020")
}

func TestStatefulResponse_PostCount(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	stats := view.QueryAST(".stat-value")
	stats.Count(2)

	stats.Index(1).HasText("42")
}

func TestStatefulResponse_DOMStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("article.profile-card").Exists()

	view.QueryAST("header").Exists()
	view.QueryAST("div.stats").Exists()

	view.QueryAST("div.stat").Count(2)
}
