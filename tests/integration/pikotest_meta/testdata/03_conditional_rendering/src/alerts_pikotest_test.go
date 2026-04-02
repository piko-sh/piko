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

	alerts "testcase_03_conditional_rendering/pages/pages_alerts_1b8e330d"
)

func TestConditionalRendering_Title(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST,
		piko.WithPageID("pages/alerts"),
	)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("System Alerts")
}

func TestConditionalRendering_ErrorAlert_Exists(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	errorAlert := view.QueryAST(".alert-error")
	errorAlert.Exists()
}

func TestConditionalRendering_ErrorAlert_Content(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".alert-error .count").Exists().HasText("3")

	view.QueryAST(".alert-error .message").Exists().HasText("Critical system error")
}

func TestConditionalRendering_WarningAlert_Exists(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	warningAlert := view.QueryAST(".alert-warning")
	warningAlert.Exists()
}

func TestConditionalRendering_WarningAlert_Content(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".alert-warning .count").Exists().HasText("2")

	view.QueryAST(".alert-warning .message").Exists().HasText("Disk space running low")
}

func TestConditionalRendering_SuccessAlert_NotRendered(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	successAlert := view.QueryAST(".alert-success")
	successAlert.NotExists()
}

func TestConditionalRendering_AlertCount(t *testing.T) {
	tester := piko.NewComponentTester(t, alerts.BuildAST)

	request := piko.NewTestRequest("GET", "/alerts").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".alert").Count(2)
}
