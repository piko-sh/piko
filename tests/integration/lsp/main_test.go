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

package lsp_stress_test

import (
	"testing"
)

func TestLSPStress_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LSP stress tests in short mode")
	}

	t.Run("01_rapid_sequential_edits", testScenarioRapidEdits)
	t.Run("02_concurrent_completion_during_edit", testScenarioConcurrentCompletion)
	t.Run("03_multi_file_rapid_edits", testScenarioMultiFile)
	t.Run("04_shared_dependency_modification", testScenarioSharedDependency)
	t.Run("05_go_type_change", testScenarioTypeChange)
	t.Run("06_sustained_random_load", testScenarioSustainedLoad)
	t.Run("07_post_stress_smoke", testScenarioPostStressSmoke)
}
