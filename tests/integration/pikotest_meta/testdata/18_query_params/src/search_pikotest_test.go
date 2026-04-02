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

	search "testcase_18_query_params/pages/pages_search_dd0a1a3f"
)

func TestQueryParams_NoParams(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST,
		piko.WithPageID("pages/search"),
	)

	request := piko.NewTestRequest("GET", "/search").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".no-query").Exists()
	view.QueryAST(".has-query").NotExists()
	view.QueryAST(".no-query-message").HasText("Enter a search term")

	view.QueryAST(".no-filters").Exists()
	view.QueryAST(".active-filters").NotExists()

	view.QueryAST(".page-value").HasText("1")
}

func TestQueryParams_WithQuery(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=golang").
		WithQueryParam("q", "golang").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".has-query").Exists()
	view.QueryAST(".no-query").NotExists()
	view.QueryAST(".query-value").HasText("golang")
}

func TestQueryParams_WithCategory(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?category=books").
		WithQueryParam("category", "books").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".active-filters").Exists()
	view.QueryAST(".no-filters").NotExists()
	view.QueryAST(".category-value").HasText("books")
}

func TestQueryParams_WithSort(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?sort=price").
		WithQueryParam("sort", "price").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".active-filters").Exists()
	view.QueryAST(".sort-value").HasText("price")
}

func TestQueryParams_DefaultSort(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".no-filters").Exists()
}

func TestQueryParams_WithPage(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?page=5").
		WithQueryParam("page", "5").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-value").HasText("5")
}

func TestQueryParams_InvalidPage(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?page=invalid").
		WithQueryParam("page", "invalid").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-value").HasText("1")
}

func TestQueryParams_MultipleParams(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=test&category=electronics&sort=date&page=3").
		WithQueryParam("q", "test").
		WithQueryParam("category", "electronics").
		WithQueryParam("sort", "date").
		WithQueryParam("page", "3").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".query-value").HasText("test")
	view.QueryAST(".category-value").HasText("electronics")
	view.QueryAST(".sort-value").HasText("date")
	view.QueryAST(".page-value").HasText("3")

	view.QueryAST(".has-query").Exists()
	view.QueryAST(".active-filters").Exists()
}

func TestQueryParams_WithQueryParams(t *testing.T) {
	tester := piko.NewComponentTester(t, search.BuildAST)

	request := piko.NewTestRequest("GET", "/search?q=widgets&category=tools").
		WithQueryParam("q", "widgets").
		WithQueryParam("category", "tools").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".query-value").HasText("widgets")
	view.QueryAST(".category-value").HasText("tools")
}
