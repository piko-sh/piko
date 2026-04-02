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

	"github.com/stretchr/testify/assert"
	"piko.sh/piko"

	empty "testcase_12_edge_cases/pages/pages_empty_54bb9383"
)

func TestEdgeCases_EmptyList(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST,
		piko.WithPageID("pages/empty"),
	)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".item").NotExists()
	view.QueryAST(".item").Count(0)

	view.QueryAST(".item-list").Exists()
	view.QueryAST(".item-list").Count(1)
}

func TestEdgeCases_FalseConditional(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".banner").NotExists()
	view.QueryAST(".banner-text").NotExists()

	view.QueryAST(".banner-section").Exists()
}

func TestEdgeCases_EmptyMessageConditional(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".message").NotExists()

	view.QueryAST(".no-message").Exists()
	view.QueryAST(".no-message").HasText("No message provided")
}

func TestEdgeCases_EmptyListMessage(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".empty-message").Exists()
	view.QueryAST(".empty-message").HasText("No items to display")
}

func TestEdgeCases_NotExistsAssertions(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".non-existent-class").NotExists()
	view.QueryAST("#non-existent-id").NotExists()
	view.QueryAST("table").NotExists()
	view.QueryAST("video").NotExists()

	view.QueryAST(".banner").NotExists()
	view.QueryAST(".item").NotExists()
}

func TestEdgeCases_ZeroCount(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	assert.Equal(t, 0, view.QueryAST(".non-existent").Len())
	assert.Equal(t, 0, view.QueryAST(".banner").Len())
	assert.Equal(t, 0, view.QueryAST(".item").Len())
}

func TestEdgeCases_SingleElement(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".single-element").Count(1)
	view.QueryAST("#edge-case-page").Count(1)
	view.QueryAST("h1").Count(1)
	assert.Equal(t, 1, view.QueryAST(".page-header").Len())
}

func TestEdgeCases_AssertMinCountWithZero(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page").MinCount(0)
	view.QueryAST(".non-existent").MinCount(0)
}

func TestEdgeCases_PageStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".page").Exists()
	view.QueryAST(".page-header").Exists()
	view.QueryAST(".page-content").Exists()
	view.QueryAST(".page-footer").Exists()

	view.QueryAST(".banner-section").Exists()
	view.QueryAST(".list-section").Exists()
	view.QueryAST(".message-section").Exists()
}

func TestEdgeCases_IdSelector(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("#edge-case-page").Exists()
	view.QueryAST("#edge-case-page").Count(1)
	view.QueryAST("#edge-case-page").HasClass("page")
}

func TestEdgeCases_ChainedAssertionsOnMissingElement(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".non-existent").
		NotExists().
		AssertCount(0)
}

func TestEdgeCases_Metadata(t *testing.T) {
	tester := piko.NewComponentTester(t, empty.BuildAST)

	request := piko.NewTestRequest("GET", "/empty").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Edge Cases Test")

	view.AssertDescription("")
}
