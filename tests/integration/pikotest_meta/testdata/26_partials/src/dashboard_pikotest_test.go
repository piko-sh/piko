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

	dashboard "testcase_26_partials/pages/pages_dashboard_0fcaf832"
)

func TestPartials_MainPageElements(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST,
		piko.WithPageID("pages/dashboard"),
	)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard-page").Exists()
	view.QueryAST(".dashboard-main").Exists()
	view.QueryAST(".cards-grid").Exists()

	view.QueryAST(".section-title").Exists()
	view.QueryAST(".section-title").HasText("Overview")

	view.QueryAST(".recent-activity").Exists()
	view.QueryAST(".activity-title").HasText("Recent Activity")
	view.QueryAST(".activity-list").Exists()
	view.QueryAST(".activity-item").Count(3)
}

func TestPartials_HeaderElements(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".site-header").Exists()
	view.QueryAST(".header-content").Exists()
	view.QueryAST(".header-title").Exists()
	view.QueryAST(".header-title").HasText("Admin Dashboard")

	view.QueryAST(".header-subtitle").Exists()
	view.QueryAST(".header-subtitle").HasText("Welcome back, Admin")

	view.QueryAST(".header-nav").Exists()
	view.QueryAST(".nav-link").Count(3)
}

func TestPartials_CardElements(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".card").Count(3)

	view.QueryAST(".card-header").Count(3)
	view.QueryAST(".card-title").Count(3)
	view.QueryAST(".card-body").Count(3)
	view.QueryAST(".card-content").Count(3)
	view.QueryAST(".card-footer").Count(3)
}

func TestPartials_CardProps(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	cardTitles := view.QueryAST(".card .card-title")
	cardTitles.Index(0).HasText("Total Users")
	cardTitles.Index(1).HasText("Total Orders")
	cardTitles.Index(2).HasText("Revenue")

	cardContents := view.QueryAST(".card .card-content")
	cardContents.Index(0).HasText("1,250")
	cardContents.Index(1).HasText("847")
	cardContents.Index(2).HasText("$45,230")
}

func TestPartials_FooterElements(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".site-footer").Exists()
	view.QueryAST(".footer-content").Exists()
	view.QueryAST(".copyright").Exists()

	view.QueryAST(".footer-links").Exists()
	view.QueryAST(".footer-link").Count(3)
}

func TestPartials_FooterProps(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".copyright").HasText("Copyright 2024 Acme Corp")
}

func TestPartials_SlotContent(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".btn-view").Count(3)

	buttons := view.QueryAST(".btn-view")
	buttons.Index(0).HasText("View All Users")
	buttons.Index(1).HasText("View All Orders")
	buttons.Index(2).HasText("View Report")
}

func TestPartials_NestedQueries(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".cards-grid .card").Count(3)

	view.QueryAST(".site-header .nav-link").Count(3)

	view.QueryAST(".site-footer .footer-link").Count(3)
}

func TestPartials_FullASTFlattening(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".dashboard-page").Exists()

	view.QueryAST(".site-header").Exists()

	view.QueryAST(".card").Exists()

	view.QueryAST(".site-footer").Exists()

	view.QueryAST(".dashboard-page .site-header").Exists()
	view.QueryAST(".dashboard-page .dashboard-main").Exists()
	view.QueryAST(".dashboard-page .site-footer").Exists()
}

func TestPartials_ActivityItems(t *testing.T) {
	tester := piko.NewComponentTester(t, dashboard.BuildAST)

	request := piko.NewTestRequest("GET", "/dashboard").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	items := view.QueryAST(".activity-item")
	items.Count(3)
	items.Index(0).HasText("New user registered")
	items.Index(1).HasText("Order #1234 completed")
	items.Index(2).HasText("Payment received")
}
