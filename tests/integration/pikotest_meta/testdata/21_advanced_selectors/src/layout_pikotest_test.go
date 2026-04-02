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

	layout "testcase_21_advanced_selectors/pages/pages_layout_242e7772"
)

func TestSelector_TagOnly(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST,
		piko.WithPageID("pages/layout"),
	)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("header").Exists()
	view.QueryAST("main").Exists()
	view.QueryAST("footer").Exists()
	view.QueryAST("nav").Count(2)
	view.QueryAST("article").Exists()
	view.QueryAST("section").Exists()
}

func TestSelector_ClassOnly(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-wrapper").Exists()
	view.QueryAST(".site-header").Exists()
	view.QueryAST(".content-area").Exists()
	view.QueryAST(".site-footer").Exists()
	view.QueryAST(".nav-link").Count(3)
	view.QueryAST(".widget").Count(2)
}

func TestSelector_TagAndClass(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("header.site-header").Exists()
	view.QueryAST("main.content-area").Exists()
	view.QueryAST("footer.site-footer").Exists()
	view.QueryAST("nav.main-nav").Exists()
	view.QueryAST("nav.footer-nav").Exists()
	view.QueryAST("article.featured").Exists()
	view.QueryAST("div.widget").Count(2)
}

func TestSelector_MultipleClasses(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".featured.article").Exists()
	view.QueryAST(".btn.btn-small").Exists()
	view.QueryAST(".widget.recent-posts").Exists()
	view.QueryAST(".widget.tags").Exists()
	view.QueryAST(".post-item.featured").Exists()
	view.QueryAST(".tag.featured").Exists()
}

func TestSelector_DescendantCombinator(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("header .nav-link").Count(3)
	view.QueryAST("main .article-title").Exists()
	view.QueryAST(".sidebar .widget").Count(2)
	view.QueryAST(".post-list .post-item").Count(3)
	view.QueryAST(".tag-cloud .tag").Count(3)
	view.QueryAST("footer .footer-link").Count(2)
}

func TestSelector_DeepDescendant(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page-wrapper header nav a").Count(3)
	view.QueryAST(".content-area .sidebar .post-list li").Count(3)
	view.QueryAST("main section div h3").Count(2)
}

func TestSelector_AttributeSelector(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("[data-action]").Exists()
	view.QueryAST("[data-action=logout]").Exists()

	view.QueryAST("a[href]").MinCount(5)
	view.QueryAST("a[href='/']").Exists()
	view.QueryAST("a[href='/about']").Exists()
}

func TestSelector_ClassWithDescendant(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".main-nav .nav-link").Count(3)
	view.QueryAST(".footer-nav .footer-link").Count(2)
	view.QueryAST(".article .article-meta .author").Exists()
	view.QueryAST(".article .article-meta .date").Exists()
}

func TestSelector_SpecificLink(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("a.nav-link.home-link").Exists()
	view.QueryAST("a.nav-link.primary").Exists()
	view.QueryAST(".post-item a.post-link").Count(3)
}

func TestSelector_TextContent(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("a.home-link").HasText("Home")
	view.QueryAST(".user-name").HasText("John Doe")
	view.QueryAST(".article-title").HasText("Featured Article")
	view.QueryAST(".copyright").ContainsText("2024")
}

func TestSelector_NoMatchReturnsEmpty(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".nonexistent").NotExists()
	view.QueryAST("table").NotExists()
	view.QueryAST("#id-selector").NotExists()
	view.QueryAST(".widget.nonexistent").NotExists()
}

func TestSelector_CountAssertions(t *testing.T) {
	tester := piko.NewComponentTester(t, layout.BuildAST)

	request := piko.NewTestRequest("GET", "/layout").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("a").MinCount(5)
	view.QueryAST(".widget-title").Count(2)
	view.QueryAST(".tag").Count(3)
	view.QueryAST(".post-item").MaxCount(5)

	links := view.QueryAST("a")
	if links.Len() < 5 {
		t.Errorf("Expected at least 5 links, got %d", links.Len())
	}
}
