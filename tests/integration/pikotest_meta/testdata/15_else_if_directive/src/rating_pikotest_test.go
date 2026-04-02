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

	rating "testcase_15_else_if_directive/pages/pages_rating_84a9502b"
)

func TestElseIf_GradeA(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST,
		piko.WithPageID("pages/rating"),
	)

	request := piko.NewTestRequest("GET", "/rating?score=95").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-a").Exists()
	view.QueryAST(".grade-a").HasText("A - Excellent")

	view.QueryAST(".grade-b").NotExists()
	view.QueryAST(".grade-c").NotExists()
	view.QueryAST(".grade-d").NotExists()
	view.QueryAST(".grade-f").NotExists()

	view.QueryAST(".excellent").Exists()
	view.QueryAST(".excellent").HasText("Outstanding performance!")
}

func TestElseIf_GradeB(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating?score=85").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-b").Exists()
	view.QueryAST(".grade-b").HasText("B - Good")

	view.QueryAST(".grade-a").NotExists()
	view.QueryAST(".grade-c").NotExists()
	view.QueryAST(".grade-d").NotExists()
	view.QueryAST(".grade-f").NotExists()

	view.QueryAST(".passing").Exists()
	view.QueryAST(".passing").HasText("Keep up the good work!")
}

func TestElseIf_GradeC(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating?score=75").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-c").Exists()
	view.QueryAST(".grade-c").HasText("C - Satisfactory")

	view.QueryAST(".grade-a").NotExists()
	view.QueryAST(".grade-b").NotExists()
	view.QueryAST(".grade-d").NotExists()
	view.QueryAST(".grade-f").NotExists()

	view.QueryAST(".passing").Exists()
}

func TestElseIf_GradeD(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating?score=65").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-d").Exists()
	view.QueryAST(".grade-d").HasText("D - Needs Improvement")

	view.QueryAST(".grade-a").NotExists()
	view.QueryAST(".grade-b").NotExists()
	view.QueryAST(".grade-c").NotExists()
	view.QueryAST(".grade-f").NotExists()

	view.QueryAST(".failing").Exists()
	view.QueryAST(".failing").HasText("More study needed.")
}

func TestElseIf_GradeF(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating?score=45").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-f").Exists()
	view.QueryAST(".grade-f").HasText("F - Failing")

	view.QueryAST(".grade-a").NotExists()
	view.QueryAST(".grade-b").NotExists()
	view.QueryAST(".grade-c").NotExists()
	view.QueryAST(".grade-d").NotExists()

	view.QueryAST(".failing").Exists()
}

func TestElseIf_ZeroScore(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade-f").Exists()
	view.QueryAST(".failing").Exists()
}

func TestElseIf_BoundaryScores(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	tests := []struct {
		name       string
		score      string
		gradeClass string
	}{
		{name: "score 90 boundary", score: "90", gradeClass: ".grade-a"},
		{name: "score 89 boundary", score: "89", gradeClass: ".grade-b"},
		{name: "score 80 boundary", score: "80", gradeClass: ".grade-b"},
		{name: "score 79 boundary", score: "79", gradeClass: ".grade-c"},
		{name: "score 70 boundary", score: "70", gradeClass: ".grade-c"},
		{name: "score 69 boundary", score: "69", gradeClass: ".grade-d"},
		{name: "score 60 boundary", score: "60", gradeClass: ".grade-d"},
		{name: "score 59 boundary", score: "59", gradeClass: ".grade-f"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := piko.NewTestRequest("GET", "/rating?score="+tc.score).Build(context.Background())
			view := tester.Render(request, piko.NoProps{})

			view.QueryAST(tc.gradeClass).Exists()
		})
	}
}

func TestElseIf_OnlyOneGradeVisible(t *testing.T) {
	tester := piko.NewComponentTester(t, rating.BuildAST)

	request := piko.NewTestRequest("GET", "/rating?score=75").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".grade").Count(1)
}
