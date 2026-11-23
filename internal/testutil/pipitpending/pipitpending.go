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

package pipitpending

import (
	"os"
	"testing"
)

const (
	// SkipPrefix starts every skip message, so the pending tests can be found with one
	// search when the pipit pin moves.
	SkipPrefix = "requires a pipit release newer than v0.0.0-alpha.10: "

	// LocalEnv names the variable that runs pending tests against local pipit sources.
	LocalEnv = "PIKO_PIPIT_LOCAL"
)

// Skip skips the test unless it is running against local pipit sources.
//
// Takes t (testing.TB) which is the test to skip.
// Takes missing (string) which describes the pipit fix the test needs.
func Skip(t testing.TB, missing string) {
	t.Helper()
	if Local() {
		return
	}
	t.Skip(SkipPrefix + missing)
}

// Local reports whether tests run against the sibling pipit checkout.
//
// Returns bool which is true when LocalEnv is "1".
func Local() bool {
	return os.Getenv(LocalEnv) == "1"
}
