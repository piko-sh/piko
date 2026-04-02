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

	themes "testcase_24_has_class/pages/pages_themes_d1dab74d"
)

func TestHasClass_SingleClass(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST,
		piko.WithPageID("pages/themes"),
	)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("button.btn").Index(0).HasClass("btn")
	view.QueryAST(".card").Index(0).HasClass("card")
	view.QueryAST(".alert.success").HasClass("alert")
	view.QueryAST(".alert.success").HasClass("success")
}

func TestHasClass_MultipleClasses(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	primaryBtn := view.QueryAST("button.btn.primary").Index(0)
	primaryBtn.HasClass("btn")
	primaryBtn.HasClass("primary")

	shadowCard := view.QueryAST(".card.shadow.rounded")
	shadowCard.HasClass("card")
	shadowCard.HasClass("shadow")
	shadowCard.HasClass("rounded")

	compactCard := view.QueryAST(".card.compact.bordered.highlighted")
	compactCard.HasClass("card")
	compactCard.HasClass("compact")
	compactCard.HasClass("bordered")
	compactCard.HasClass("highlighted")
}

func TestHasClass_AllButtons(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	buttons := view.QueryAST(".buttons-section button")
	buttons.Count(6)

	view.QueryAST("button.primary.large").HasClass("large")
	view.QueryAST("button.small.outline").HasClass("small")
	view.QueryAST("button.small.outline").HasClass("outline")
	view.QueryAST("button.danger.disabled").HasClass("danger")
	view.QueryAST("button.danger.disabled").HasClass("disabled")
}

func TestHasClass_AlertVariants(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".alert.success").HasClass("success")
	view.QueryAST(".alert.warning").HasClass("warning")
	view.QueryAST(".alert.error.critical").HasClass("error")
	view.QueryAST(".alert.error.critical").HasClass("critical")
	view.QueryAST(".alert.info.dismissible").HasClass("info")
	view.QueryAST(".alert.info.dismissible").HasClass("dismissible")
}

func TestHasClass_BadgeAndTag(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".badge.new").HasClass("badge")
	view.QueryAST(".badge.new").HasClass("new")
	view.QueryAST(".badge.hot.featured").HasClass("hot")
	view.QueryAST(".badge.hot.featured").HasClass("featured")

	view.QueryAST(".tag.category.primary").HasClass("tag")
	view.QueryAST(".tag.category.primary").HasClass("category")
	view.QueryAST(".tag.category.primary").HasClass("primary")
}

func TestHasClass_SectionClasses(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".buttons-section").HasClass("buttons-section")
	view.QueryAST(".cards-section").HasClass("cards-section")
	view.QueryAST(".alerts-section").HasClass("alerts-section")
	view.QueryAST(".mixed-section").HasClass("mixed-section")
}

func TestHasClass_WithIndex(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	cards := view.QueryAST(".card")
	cards.Count(4)

	cards.Index(0).HasClass("card")
	cards.Index(1).HasClass("featured")
	cards.Index(2).HasClass("shadow")
	cards.Index(2).HasClass("rounded")
	cards.Index(3).HasClass("compact")
	cards.Index(3).HasClass("bordered")
	cards.Index(3).HasClass("highlighted")
}

func TestHasClass_CombinedWithOtherAssertions(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	successAlert := view.QueryAST(".alert.success")
	successAlert.Exists()
	successAlert.HasClass("alert")
	successAlert.HasClass("success")
	successAlert.HasText("Success Alert")

	largeBtn := view.QueryAST("button.large")
	largeBtn.Exists()
	largeBtn.HasClass("btn")
	largeBtn.HasClass("primary")
	largeBtn.HasClass("large")
	largeBtn.HasText("Large Primary")
}

func TestHasClass_ElementStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, themes.BuildAST)

	request := piko.NewTestRequest("GET", "/themes").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".theme-demo").Exists()
	view.QueryAST(".theme-demo").HasClass("theme-demo")

	view.QueryAST(".btn").Count(6)
	view.QueryAST(".card").Count(4)
	view.QueryAST(".alert").Count(4)
	view.QueryAST(".badge").Count(2)
	view.QueryAST(".tag").Count(1)
}
