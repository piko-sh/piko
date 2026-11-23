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

//go:build integration

package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertionsPass(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	t.Chdir(t.TempDir())
	page.Navigate("/").Fill("#name", "value").Check("#check").Focus("#name")

	page.Assert("#title").Exists().Count(1).HasText("Hello").ContainsText("ell").HasHTML(`<h1 id="title" class="big" data-role="heading" style="color: rgb(255, 0, 0)">Hello</h1>`).
		HasAttribute("data-role", "heading").HasClass("big").HasStyle("color", "rgb(255, 0, 0)").IsVisible().IsEnabled()
	page.Assert("#absent").NotExists()
	page.Assert("#name").HasValue("value").IsFocused()
	page.Assert("#check").IsChecked()
	page.Assert("#hidden").IsHidden()
	page.Assert("#disabled").IsDisabled()
	page.Uncheck("#check").Assert("#check").IsUnchecked()

	t.Setenv("PIKO_UPDATE_GOLDEN", "1")
	page.Assert("#title").MatchesGolden("title")
	t.Setenv("PIKO_UPDATE_GOLDEN", "")
	page.Assert("#title").MatchesGolden("title")
	require.Empty(t, recorder.take())
}

func TestAssertionsReportFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	t.Chdir(t.TempDir())
	page.Navigate("/")

	page.Assert("#title").NotExists().MatchesGolden("missing")

	failures := recorder.take()
	require.Len(t, failures, 2)
	assert.Contains(t, failures[0], `Assert("#title").NotExists() failed`)
	assert.Contains(t, failures[1], "PIKO_UPDATE_GOLDEN=1")
}
