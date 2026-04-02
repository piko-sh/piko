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

	search "testcase_06_query_params/pages/pages_search_dd0a1a3f"
)

func TestQueryParams_WithQuery(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST,
		piko.WithPageID("pages/search"),
	)

	request := piko.NewTestRequest("GET", "/search?q=golang").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Search Results")

	view.QueryAST(".results-info").Exists()
	view.QueryAST(".query-text").Exists().ContainsText("golang")
}

func TestQueryParams_NoQuery(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".no-query").Exists()
	view.QueryAST(".results-info").NotExists()
}

func TestQueryParams_WithPageParam(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=test&page=5").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-info").Exists().ContainsText("5")
}

func TestQueryParams_WithSortParam(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=test&sort=date").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".sort-info").Exists().ContainsText("date")
}

func TestQueryParams_Defaults(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=test").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-info").ContainsText("1")

	view.QueryAST(".sort-info").ContainsText("relevance")
}

func TestQueryParams_AllParams(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=programming&page=3&sort=votes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".query-text").ContainsText("programming")
	view.QueryAST(".page-info").ContainsText("3")
	view.QueryAST(".sort-info").ContainsText("votes")
}
