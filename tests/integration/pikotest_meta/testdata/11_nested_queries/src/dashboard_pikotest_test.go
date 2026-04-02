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

	dashboard "testcase_11_nested_queries/pages/pages_dashboard_0fcaf832"
)

func TestNestedQueries_DescendantSelector(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST,
		piko.WithPageID("pages/dashboard"),
	)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard h1").Count(1)
	view.QueryAST(".dashboard-header h1").Count(1)
	view.QueryAST(".dashboard-header nav").Count(1)
	view.QueryAST(".main-nav ul").Count(1)
	view.QueryAST(".main-nav li").Count(3)
	view.QueryAST(".main-nav a").Count(3)
}

func TestNestedQueries_DeeplyNested(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard .dashboard-content .widget-container .widget").Count(2)
	view.QueryAST(".dashboard main section div.widget").Count(2)

	view.QueryAST(".widget .item-list .item").Count(6)
}

func TestNestedQueries_MultipleClasses(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("a.nav-link.active").Count(1)
	view.QueryAST("a.nav-link.active").HasText("Home")

	view.QueryAST("button.action-btn.primary").Count(1)
	view.QueryAST("button.action-btn").Count(2)
}

func TestNestedQueries_TagWithClass(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("header.dashboard-header").Count(1)
	view.QueryAST("main.dashboard-content").Count(1)
	view.QueryAST("footer.dashboard-footer").Count(1)
	view.QueryAST("aside.sidebar").Count(1)
	view.QueryAST("section.widget-container").Count(1)

	view.QueryAST("div.widget").Count(2)
	view.QueryAST("ul.nav-list").Count(1)
	view.QueryAST("ul.item-list").Count(2)
	view.QueryAST("ul.action-list").Count(1)
}

func TestNestedQueries_StructuralContext(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard-header h1").HasText("Dashboard")
	view.QueryAST(".sidebar h3").HasText("Quick Actions")

	view.QueryAST(".main-nav a").Count(3)
	view.QueryAST(".widget-footer a").Count(2)

	view.QueryAST(".widget-header button").Count(2)
	view.QueryAST(".sidebar button").Count(2)
}

func TestNestedQueries_IndexWithText(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	navLinks := view.QueryAST(".nav-item a")
	navLinks.Count(3)
	navLinks.Index(0).HasText("Home")
	navLinks.Index(1).HasText("Reports")
	navLinks.Index(2).HasText("Settings")
}

func TestNestedQueries_WidgetStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".widget .widget-header").Count(2)
	view.QueryAST(".widget .widget-body").Count(2)
	view.QueryAST(".widget .widget-footer").Count(2)

	view.QueryAST(".widget-header h2").Count(2)
	view.QueryAST(".widget-header button").Count(2)
}

func TestNestedQueries_ListItemContent(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".item").Count(6)
	view.QueryAST(".item .item-icon").Count(6)
	view.QueryAST(".item .item-text").Count(6)
}

func TestNestedQueries_CountAtDifferentLevels(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard").Count(1)
	view.QueryAST(".widget").Count(2)
	view.QueryAST(".item-list").Count(2)
	view.QueryAST(".item").Count(6)
	view.QueryAST(".item-icon").Count(6)
	view.QueryAST(".item-text").Count(6)
}

func TestNestedQueries_HeaderFooterMain(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST("header").Count(1)
	view.QueryAST("main").Count(1)
	view.QueryAST("footer").Count(1)
	view.QueryAST("nav").Count(1)
	view.QueryAST("aside").Count(1)
	view.QueryAST("section").Count(1)
}

func TestNestedQueries_SpecificWidgetContent(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	widgetTitles := view.QueryAST(".widget-title")
	widgetTitles.Count(2)

	widgetLinks := view.QueryAST(".widget-link")
	widgetLinks.Count(2)
	widgetLinks.Index(0).HasText("View All")
	widgetLinks.Index(1).HasText("View All")
}

func TestNestedQueries_SidebarStructure(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".sidebar .sidebar-section").Count(1)
	view.QueryAST(".sidebar .sidebar-title").Count(1)
	view.QueryAST(".sidebar .action-list").Count(1)
	view.QueryAST(".sidebar .action-item").Count(2)

	view.QueryAST(".action-btn.primary").HasText("New Report")
}
