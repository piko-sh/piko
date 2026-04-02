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

	api "testcase_16_http_methods/pages/pages_api_60975cfd"
)

func TestHTTPMethods_GET(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST,
		piko.WithPageID("pages/api"),
	)

	request := piko.NewTestRequest("GET", "/api").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".method-value").HasText("GET")
	view.QueryAST(".description").HasText("Fetching data")

	view.QueryAST(".get-indicator").Exists()
	view.QueryAST(".get-indicator").HasText("GET Request")

	view.QueryAST(".post-indicator").NotExists()
	view.QueryAST(".put-indicator").NotExists()
	view.QueryAST(".delete-indicator").NotExists()
	view.QueryAST(".patch-indicator").NotExists()

	view.QueryAST(".read-action").Exists()
	view.QueryAST(".create-action").NotExists()
	view.QueryAST(".update-action").NotExists()
	view.QueryAST(".delete-action").NotExists()
}

func TestHTTPMethods_POST(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST)

	request := piko.NewTestRequest("POST", "/api").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".method-value").HasText("POST")
	view.QueryAST(".description").HasText("Creating resource")

	view.QueryAST(".post-indicator").Exists()
	view.QueryAST(".post-indicator").HasText("POST Request")

	view.QueryAST(".get-indicator").NotExists()
	view.QueryAST(".put-indicator").NotExists()
	view.QueryAST(".delete-indicator").NotExists()

	view.QueryAST(".create-action").Exists()
	view.QueryAST(".read-action").NotExists()
}

func TestHTTPMethods_PUT(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST)

	request := piko.NewTestRequest("PUT", "/api").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".method-value").HasText("PUT")
	view.QueryAST(".description").HasText("Updating resource")

	view.QueryAST(".put-indicator").Exists()
	view.QueryAST(".put-indicator").HasText("PUT Request")

	view.QueryAST(".update-action").Exists()
	view.QueryAST(".create-action").NotExists()
	view.QueryAST(".delete-action").NotExists()
}

func TestHTTPMethods_DELETE(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST)

	request := piko.NewTestRequest("DELETE", "/api").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".method-value").HasText("DELETE")
	view.QueryAST(".description").HasText("Removing resource")

	view.QueryAST(".delete-indicator").Exists()
	view.QueryAST(".delete-indicator").HasText("DELETE Request")

	view.QueryAST(".delete-action").Exists()
	view.QueryAST(".read-action").NotExists()
	view.QueryAST(".create-action").NotExists()
	view.QueryAST(".update-action").NotExists()
}

func TestHTTPMethods_PATCH(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST)

	request := piko.NewTestRequest("PATCH", "/api").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".method-value").HasText("PATCH")
	view.QueryAST(".description").HasText("Partial update")

	view.QueryAST(".patch-indicator").Exists()
	view.QueryAST(".patch-indicator").HasText("PATCH Request")

	view.QueryAST(".update-action").Exists()
}

func TestHTTPMethods_OnlyOneIndicator(t *testing.T) {
	tester := piko.NewComponentTester(t, api.BuildAST)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			request := piko.NewTestRequest(method, "/api").Build(context.Background())
			view := tester.Render(request, piko.NoProps{})

			view.QueryAST(".indicator").Count(1)
		})
	}
}
