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

	auth "testcase_13_else_directive/pages/pages_auth_eadff339"
)

func TestElseDirective_LoggedOut(t *testing.T) {
	tester := piko.NewComponentTester(t, auth.BuildAST,
		piko.WithPageID("pages/auth"),
	)

	request := piko.NewTestRequest("GET", "/auth").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".logged-in").NotExists()
	view.QueryAST(".logged-out").Exists()

	view.QueryAST(".login-prompt").Exists()
	view.QueryAST(".login-prompt").HasText("Please log in to continue")
	view.QueryAST(".login-btn").Exists()
	view.QueryAST(".login-btn").HasText("Sign In")

	view.QueryAST(".admin-badge").NotExists()
	view.QueryAST(".user-badge").NotExists()
	view.QueryAST(".welcome-message").NotExists()
}

func TestElseDirective_LoggedInUser(t *testing.T) {
	tester := piko.NewComponentTester(t, auth.BuildAST)

	request := piko.NewTestRequest("GET", "/auth?auth=user").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".logged-in").Exists()
	view.QueryAST(".logged-out").NotExists()

	view.QueryAST(".welcome-message").Exists()
	view.QueryAST(".welcome-message").ContainsText("JohnDoe")

	view.QueryAST(".admin-badge").NotExists()
	view.QueryAST(".user-badge").Exists()
	view.QueryAST(".badge.user").HasText("Regular User")

	view.QueryAST(".access-admin").NotExists()
	view.QueryAST(".access-default").Exists()
	view.QueryAST(".access-default").HasText("Standard access only")
}

func TestElseDirective_LoggedInAdmin(t *testing.T) {
	tester := piko.NewComponentTester(t, auth.BuildAST)

	request := piko.NewTestRequest("GET", "/auth?auth=admin").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".logged-in").Exists()
	view.QueryAST(".logged-out").NotExists()

	view.QueryAST(".welcome-message").Exists()
	view.QueryAST(".welcome-message").ContainsText("AdminUser")

	view.QueryAST(".admin-badge").Exists()
	view.QueryAST(".badge.admin").HasText("Administrator")
	view.QueryAST(".user-badge").NotExists()

	view.QueryAST(".access-admin").Exists()
	view.QueryAST(".access-admin").HasText("Full administrative access")
	view.QueryAST(".access-default").NotExists()
}

func TestElseDirective_NestedIfElse(t *testing.T) {
	tester := piko.NewComponentTester(t, auth.BuildAST)

	request := piko.NewTestRequest("GET", "/auth?auth=user").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".logged-in").Exists()

	view.QueryAST(".user-badge").Exists()
	view.QueryAST(".admin-badge").NotExists()

	reqAdmin := piko.NewTestRequest("GET", "/auth?auth=admin").Build(context.Background())
	viewAdmin := tester.Render(reqAdmin, piko.NoProps{})

	viewAdmin.QueryAST(".admin-badge").Exists()
	viewAdmin.QueryAST(".user-badge").NotExists()
}

func TestElseDirective_Permissions(t *testing.T) {
	tester := piko.NewComponentTester(t, auth.BuildAST)

	tests := []struct {
		name          string
		authParam     string
		expectAdmin   bool
		expectDefault bool
	}{
		{name: "no auth", authParam: "", expectAdmin: false, expectDefault: true},
		{name: "user auth", authParam: "user", expectAdmin: false, expectDefault: true},
		{name: "admin auth", authParam: "admin", expectAdmin: true, expectDefault: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "/auth"
			if tc.authParam != "" {
				url = "/auth?auth=" + tc.authParam
			}

			request := piko.NewTestRequest("GET", url).Build(context.Background())
			view := tester.Render(request, piko.NoProps{})

			if tc.expectAdmin {
				view.QueryAST(".access-admin").Exists()
				view.QueryAST(".access-default").NotExists()
			} else {
				view.QueryAST(".access-admin").NotExists()
				view.QueryAST(".access-default").Exists()
			}
		})
	}
}
