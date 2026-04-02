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

	profile "testcase_17_request_headers/pages/pages_profile_4331497b"
)

func TestPathParams_NoParams(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST,
		piko.WithPageID("pages/profile"),
	)

	request := piko.NewTestRequest("GET", "/profile").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".user-details").NotExists()
	view.QueryAST(".no-user").Exists()
	view.QueryAST(".no-user-message").HasText("No user specified")

	view.QueryAST(".section-value").HasText("overview")
}

func TestPathParams_WithUserID(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/123").
		WithPathParam("id", "123").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".user-details").Exists()
	view.QueryAST(".no-user").NotExists()
	view.QueryAST(".id-value").HasText("123")
}

func TestPathParams_WithUsername(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/456/johndoe").
		WithPathParam("id", "456").
		WithPathParam("username", "johndoe").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".id-value").HasText("456")
	view.QueryAST(".username").Exists()
	view.QueryAST(".username-value").HasText("johndoe")
}

func TestPathParams_WithPathParams(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/789/alice").
		WithPathParams(map[string]string{
			"id":       "789",
			"username": "alice",
		}).
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".id-value").HasText("789")
	view.QueryAST(".username-value").HasText("alice")
}

func TestPathParams_SectionOverview(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/123/overview").
		WithPathParam("id", "123").
		WithPathParam("section", "overview").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".section-value").HasText("overview")

	activeItems := view.QueryAST(".nav-item.active")
	activeItems.MinCount(1)
}

func TestPathParams_SectionPosts(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/123/posts").
		WithPathParam("id", "123").
		WithPathParam("section", "posts").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".section-value").HasText("posts")
}

func TestPathParams_SectionSettings(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/123/settings").
		WithPathParam("id", "123").
		WithPathParam("section", "settings").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".section-value").HasText("settings")
}

func TestPathParams_EmptyUsername(t *testing.T) {
	tester := piko.NewComponentTester(t, profile.BuildAST)

	request := piko.NewTestRequest("GET", "/profile/123").
		WithPathParam("id", "123").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".user-details").Exists()
	view.QueryAST(".id-value").HasText("123")
	view.QueryAST(".username").NotExists()
}
