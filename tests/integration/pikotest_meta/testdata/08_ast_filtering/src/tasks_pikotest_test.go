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

	tasks "testcase_08_ast_filtering/pages/pages_tasks_37db46f6"
)

func TestASTFiltering_TotalCount(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST,
		piko.WithPageID("pages/tasks"),
	)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("li.task-item").Count(5)
}

func TestASTFiltering_CountMethod(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	items := view.QueryAST("li.task-item")

	assert.Equal(t, 5, items.Len())

	titles := view.QueryAST(".task-title")
	assert.Equal(t, 5, titles.Len())

	statusSpans := view.QueryAST(".task-status")
	assert.Equal(t, 5, statusSpans.Len())
}

func TestASTFiltering_AssertMinMaxCount(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	items := view.QueryAST("li.task-item")

	items.MinCount(3)
	items.MinCount(5)

	items.MaxCount(10)
	items.MaxCount(5)
}

func TestASTFiltering_IndexMethod(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	titles := view.QueryAST(".task-title")

	firstTitle := titles.Index(0)
	firstTitle.Exists()
	firstTitle.Count(1)

	thirdTitle := titles.Index(2)
	thirdTitle.Exists()
	thirdTitle.Count(1)
}

func TestASTFiltering_MultipleSelectors(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("h1").Count(1)
	view.QueryAST("ul").Count(1)
	view.QueryAST("li").Count(5)
	view.QueryAST("span").Count(15)

	view.QueryAST(".task-list").Count(1)
	view.QueryAST(".task-item").Count(5)
	view.QueryAST(".task-title").Count(5)
	view.QueryAST(".task-status").Count(5)
	view.QueryAST(".task-priority").Count(5)
}

func TestASTFiltering_CombinedSelectors(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("li.task-item").Count(5)
	view.QueryAST("span.task-title").Count(5)
	view.QueryAST("div.task-list").Count(1)

	view.QueryAST(".task-list h1").Count(1)
	view.QueryAST(".task-list ul").Count(1)
	view.QueryAST(".task-list li").Count(5)
}

func TestASTFiltering_ExistsNotExists(t *testing.T) {
	tester := piko.NewComponentTester(t, tasks.BuildAST)

	request := piko.NewTestRequest("GET", "/tasks").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".task-list").Exists()
	view.QueryAST(".task-item").Exists()
	view.QueryAST("h1").Exists()

	view.QueryAST(".non-existent").NotExists()
	view.QueryAST("table").NotExists()
	view.QueryAST("#missing-id").NotExists()
}
