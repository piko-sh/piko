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

	notifications "testcase_19_p_show/pages/pages_notifications_c7e91f02"
)

func TestPShow_AllHidden(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST,
		piko.WithPageID("pages/notifications"),
	)

	request := piko.NewTestRequest("GET", "/notifications").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification.success").Exists()
	view.QueryAST(".notification.warning").Exists()
	view.QueryAST(".notification.error").Exists()
	view.QueryAST(".notification.info").Exists()

	view.QueryAST(".notification.success").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.warning").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.error").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.info").HasAttributeContaining("style", "display:none")

	view.QueryAST(".no-messages").Exists()

	view.QueryAST(".count-value").HasText("0")
}

func TestPShow_SuccessShown(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications?success=true").
		WithQueryParam("success", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	success := view.QueryAST(".notification.success")
	success.Exists()

	view.QueryAST(".notification.warning").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.error").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.info").HasAttributeContaining("style", "display:none")

	view.QueryAST(".count-value").HasText("1")

	view.QueryAST(".no-messages").HasAttributeContaining("style", "display:none")
}

func TestPShow_WarningShown(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications?warning=true").
		WithQueryParam("warning", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification.warning").Exists()

	view.QueryAST(".notification.success").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.error").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.info").HasAttributeContaining("style", "display:none")

	view.QueryAST(".count-value").HasText("1")
}

func TestPShow_ErrorShown(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications?error=true").
		WithQueryParam("error", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification.error").Exists()

	view.QueryAST(".notification.success").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.warning").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.info").HasAttributeContaining("style", "display:none")

	view.QueryAST(".count-value").HasText("1")
}

func TestPShow_MultipleShown(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications?success=true&error=true").
		WithQueryParam("success", "true").
		WithQueryParam("error", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification.success").Exists()
	view.QueryAST(".notification.error").Exists()

	view.QueryAST(".notification.warning").HasAttributeContaining("style", "display:none")
	view.QueryAST(".notification.info").HasAttributeContaining("style", "display:none")

	view.QueryAST(".count-value").HasText("2")
}

func TestPShow_AllShown(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications?success=true&warning=true&error=true&info=true").
		WithQueryParam("success", "true").
		WithQueryParam("warning", "true").
		WithQueryParam("error", "true").
		WithQueryParam("info", "true").
		Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification.success").Exists()
	view.QueryAST(".notification.warning").Exists()
	view.QueryAST(".notification.error").Exists()
	view.QueryAST(".notification.info").Exists()

	view.QueryAST(".count-value").HasText("4")

	view.QueryAST(".no-messages").HasAttributeContaining("style", "display:none")
}

func TestPShow_ElementsAlwaysInDOM(t *testing.T) {
	tester := piko.NewComponentTester(t, notifications.BuildAST)

	request := piko.NewTestRequest("GET", "/notifications").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.QueryAST(".notification").Count(4)

	view.QueryAST(".notification.success .message").HasText("Operation completed successfully")
	view.QueryAST(".notification.warning .message").HasText("Please review your input")
	view.QueryAST(".notification.error .message").HasText("An error occurred")
	view.QueryAST(".notification.info .message").HasText("Here is some information")
}
