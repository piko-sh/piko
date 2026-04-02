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

package pages

import (
	"context"
	"testing"

	"piko.sh/piko"

	home "pikotest_03_expected_fail_wrong_title/dist/pages/pages_home_dab9c3dd"
)

func TestWrongTitle_ShouldFail(t *testing.T) {
	tester := piko.NewComponentTester(t, home.BuildAST)

	request := piko.NewTestRequest("GET", "/").Build(context.Background())
	view := tester.Render(request, piko.NoProps{})

	view.AssertTitle("Wrong Title That Does Not Match")
}
