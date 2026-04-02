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

	article "testcase_25_metadata/pages/pages_article_1466cebc"
)

func TestMetadata_DefaultArticle(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST,
		piko.WithPageID("pages/article"),
	)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".article-title").HasText("Default Article")
	view.QueryAST(".author-name").HasText("Unknown Author")
	view.QueryAST(".published-date").HasText("2024-01-01")
	view.QueryAST(".content-text").ContainsText("default article content")
}

func TestMetadata_FeaturedArticle(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article?id=featured").
		WithQueryParam("id", "featured").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".article-title").HasText("Featured: Building Web Apps with Piko")
	view.QueryAST(".author-name").HasText("Jane Developer")
	view.QueryAST(".published-date").HasText("2024-06-15")
	view.QueryAST(".content-text").ContainsText("modern web applications")
}

func TestMetadata_TutorialArticle(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article?id=tutorial").
		WithQueryParam("id", "tutorial").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".article-title").HasText("Tutorial: Getting Started")
	view.QueryAST(".author-name").HasText("John Coder")
	view.QueryAST(".published-date").HasText("2024-05-20")
	view.QueryAST(".content-text").ContainsText("comprehensive guide")
}

func TestMetadata_ArticleStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("article.article-page").Exists()

	view.QueryAST(".article-header").Exists()
	view.QueryAST(".article-header h1.article-title").Exists()
	view.QueryAST(".article-header .article-meta").Exists()

	view.QueryAST(".article-content").Exists()
	view.QueryAST(".article-content .content-text").Exists()

	view.QueryAST(".article-footer").Exists()
	view.QueryAST(".footer-note").HasText("Thank you for reading!")
}

func TestMetadata_TimeElement(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	timeEl := view.QueryAST("time.published-date")
	timeEl.Exists()
	timeEl.HasText("2024-01-01")
}

func TestMetadata_MetaSection(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	meta := view.QueryAST(".article-meta")
	meta.Exists()

	view.QueryAST(".article-meta .author").Exists()
	view.QueryAST(".article-meta .date").Exists()
}

func TestMetadata_DifferentArticles(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	testCases := []struct {
		name          string
		id            string
		expectedTitle string
	}{
		{name: "default", id: "", expectedTitle: "Default Article"},
		{name: "featured", id: "featured", expectedTitle: "Featured: Building Web Apps with Piko"},
		{name: "tutorial", id: "tutorial", expectedTitle: "Tutorial: Getting Started"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/article"
			if tc.id != "" {
				path = "/article?id=" + tc.id
			}

			request := piko.NewTestRequest("GET", path).Build(context.Background())
			if tc.id != "" {
				request = piko.NewTestRequest("GET", path).
					WithQueryParam("id", tc.id).
					Build(context.Background())
			}

			view := tester.Render(request, piko.NoProps{})
			view.QueryAST(".article-title").HasText(tc.expectedTitle)
		})
	}
}
