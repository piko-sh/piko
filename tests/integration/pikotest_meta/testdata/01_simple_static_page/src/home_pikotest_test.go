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

	home "testcase_01_simple_static_page/pages/pages_home_16ddfae3"
)

func TestSimpleStaticPage_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST,
		piko.WithPageID("pages/home"),
	)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Home Page")
}

func TestSimpleStaticPage_StatusCode(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertStatusCode(200)
}

func TestSimpleStaticPage_H1Content(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	h1 := view.QueryAST("h1")
	h1.Exists()
	h1.HasText("Welcome Home")
}

func TestSimpleStaticPage_DescriptionParagraph(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	description := view.QueryAST("p.description")
	description.Exists()
	description.HasText("This is a simple static page.")
}

func TestSimpleStaticPage_DOMStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	rootDiv := view.QueryAST("div")
	rootDiv.Exists()

	view.QueryAST("h1").Count(1)
	view.QueryAST("p").Count(1)
}

func TestSimpleStaticPage_NoJSScript(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertNoJSScript()
}
