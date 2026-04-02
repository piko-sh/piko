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

	leaderboard "testcase_23_index_assertions/pages/pages_leaderboard_6f85bbc3"
)

func TestIndex_FirstElement(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST,
		piko.WithPageID("pages/leaderboard"),
	)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	first := view.QueryAST(".player-row").Index(0)
	first.Exists()

	view.QueryAST(".player-row").Index(0).HasAttributePresent("data-rank")
	view.QueryAST(".player-row .name").Index(0).HasText("Alice")
	view.QueryAST(".player-row .score").Index(0).HasText("1500")
}

func TestIndex_SecondElement(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	second := view.QueryAST(".player-row").Index(1)
	second.Exists()
	second.HasAttributePresent("data-rank")

	view.QueryAST(".player-row .name").Index(1).HasText("Bob")
	view.QueryAST(".player-row .score").Index(1).HasText("1350")
}

func TestIndex_ThirdElement(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".player-row").Index(2).HasAttributePresent("data-rank")
	view.QueryAST(".player-row .name").Index(2).HasText("Charlie")
	view.QueryAST(".player-row .score").Index(2).HasText("1200")
}

func TestIndex_LastElements(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".player-row .name").Index(3).HasText("Diana")

	view.QueryAST(".player-row .name").Index(4).HasText("Eve")
	view.QueryAST(".player-row .score").Index(4).HasText("1000")
}

func TestIndex_WithMedals(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".medal.gold").Exists()
	view.QueryAST(".medal.silver").Exists()
	view.QueryAST(".medal.bronze").Exists()

	medals := view.QueryAST(".medal")
	if medals.Len() != 3 {
		t.Errorf("Expected 3 medals, got %d", medals.Len())
	}
}

func TestIndex_NoMedalPlayers(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	noMedals := view.QueryAST(".no-medal")
	noMedals.Count(2)
}

func TestIndex_BoundaryConditions(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	last := view.QueryAST(".player-row").Index(4)
	last.Exists()

	view.QueryAST(".player-row").Count(5)
}

func TestIndex_ChainedSelectors(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("tr.player-row").Index(0).Exists()

	headers := view.QueryAST(".header-row th")
	headers.Count(4)
	headers.Index(0).HasClass("col-rank")
	headers.Index(1).HasClass("col-name")
	headers.Index(2).HasClass("col-score")
	headers.Index(3).HasClass("col-badge")
}

func TestIndex_AllRowsIteration(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	expectedNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}

	rows := view.QueryAST(".player-row")
	rows.Count(5)

	for i, expectedName := range expectedNames {
		nameCell := view.QueryAST(".player-row .name").Index(i)
		nameCell.HasText(expectedName)
	}
}

func TestIndex_TableStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, leaderboard.BuildAST)

	request := piko.NewTestRequest("GET", "/leaderboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("table.rankings").Exists()

	view.QueryAST("thead tr").Count(1)

	view.QueryAST("tbody tr").Count(5)

	for i := 0; i < 5; i++ {
		row := view.QueryAST("tbody tr").Index(i)
		row.Exists()
	}
}
