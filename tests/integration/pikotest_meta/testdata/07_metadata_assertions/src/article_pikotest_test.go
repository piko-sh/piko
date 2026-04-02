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

	article "testcase_07_metadata_assertions/pages/pages_article_1466cebc"
)

func TestMetadata_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST,
		piko.WithPageID("pages/article"),
	)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Understanding Go Interfaces | Tech Blog")
}

func TestMetadata_Description(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertDescription("Learn about Go interfaces and how to use them effectively in your code.")
}

func TestMetadata_Language(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertLanguage("en")
}

func TestMetadata_CanonicalURL(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertCanonicalURL("https://example.com/articles/go-interfaces")
}

func TestMetadata_StatusCode(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertStatusCode(200)
}

func TestMetadata_ContentRendered(t *testing.T) {
	tester := piko.NewComponentTester(t, article.BuildAST)

	request := piko.NewTestRequest("GET", "/article").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("article.blog-post").Exists()
	view.QueryAST("h1").HasText("Understanding Go Interfaces")
	view.QueryAST(".author").ContainsText("Jane Doe")
	view.QueryAST(".content p").ContainsText("Go interfaces are a powerful feature")
}
