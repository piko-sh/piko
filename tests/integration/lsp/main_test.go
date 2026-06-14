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
	t.Run("08_rapid_edits_high_volume", testScenarioRapidEditsHighVolume)
	t.Run("09_concurrent_multifile_storm", testScenarioMultiFileStorm)
}

func TestGoplsBridge_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gopls bridge integration tests in short mode")
	}

	t.Run("phase1_real_gopls", func(t *testing.T) {
		t.Run("01_go_block_hover", testBridgeGoBlockHover)
		t.Run("02_go_block_completion", testBridgeGoBlockCompletion)
		t.Run("03_go_block_definition", testBridgeGoBlockDefinition)
		t.Run("04_cross_component_resolution", testBridgeCrossComponentResolution)
		t.Run("05_diagnostics_mapped_to_line", testBridgeDiagnosticsMappedToLine)
		t.Run("06_rewrite_noise_suppressed", testBridgeRewriteNoiseSuppressed)
		t.Run("07_space_in_workspace_path", testBridgeSpaceInWorkspacePath)
		t.Run("08_enabled_via_init_option", testBridgeEnabledViaInitOption)
		t.Run("09_toggle_off", testBridgeToggleOff)
		t.Run("10_gopls_absent", testBridgeGoplsAbsent)
		t.Run("11_diagnostics_on_edit_and_clear", testBridgeDiagnosticsOnEditAndClear)
		t.Run("12_multi_connection_shared_child", testBridgeMultiConnectionSharedChild)
		t.Run("13_rename_across_block", testBridgeRenameAcrossBlock)
		t.Run("14_completion_auto_import", testBridgeCompletionAutoImport)
		t.Run("15_unterminated_literal_clamp", testBridgeUnterminatedLiteralClampsAndClears)
	})

	t.Run("phase2_fake_gopls", func(t *testing.T) {
		t.Run("13_wedged_gopls", testBridgeWedgedGopls)
		t.Run("14_crashed_gopls", testBridgeCrashedGopls)
		t.Run("15_too_old_gopls", testBridgeTooOldGopls)
		t.Run("16_null_init_gopls", testBridgeNullInitGopls)
		t.Run("17_crash_recovery", testBridgeCrashRecovery)
	})
}
