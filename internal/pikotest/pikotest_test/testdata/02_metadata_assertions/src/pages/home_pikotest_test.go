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

package pages

import (
	"context"
	"testing"

	"piko.sh/piko"

	home "pikotest_02_metadata_assertions/dist/pages/pages_home_dab9c3dd"
)

func TestMetadata_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Metadata Page")
}

func TestMetadata_Description(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertDescription("A page to test metadata assertions")
}

func TestMetadata_Language(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertLanguage("en")
}

func TestMetadata_OGTag(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertHasOGTag("og:title", "Metadata Page")
}

func TestMetadata_MetaTag(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertHasMetaTag("author", "Piko")
}

func TestMetadata_H1Content(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("h1").Exists().HasText("Metadata Test")
}
