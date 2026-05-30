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

//go:build crosslang

package harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSpecRejectsMissingName(t *testing.T) {
	tempDirectory := t.TempDir()
	path := filepath.Join(tempDirectory, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"description":"oops"}`), 0o644))
	_, err := LoadSpec(path)
	assert.ErrorContains(t, err, "missing name")
}

func TestLoadSpecRejectsZeroKInner(t *testing.T) {
	tempDirectory := t.TempDir()
	path := filepath.Join(tempDirectory, "bad.json")
	body := `{"name":"x","expected_stdout_sha256":"abc","k_inner":0}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	_, err := LoadSpec(path)
	assert.ErrorContains(t, err, "k_inner")
}

func TestLoadSpecDefaultsTimeoutWhenAbsent(t *testing.T) {
	tempDirectory := t.TempDir()
	path := filepath.Join(tempDirectory, "ok.json")
	body := `{"name":"x","expected_stdout_sha256":"abc","k_inner":1}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	spec, err := LoadSpec(path)
	require.NoError(t, err)
	assert.Equal(t, 120, spec.TimeoutSeconds)
}

func TestMatchesFilterMatchesSubstrings(t *testing.T) {
	config := SuiteConfig{Filter: []string{"fib", "trie"}}
	assert.True(t, config.MatchesFilter("01_fib_iterative"))
	assert.True(t, config.MatchesFilter("11_trie_50k_words"))
	assert.False(t, config.MatchesFilter("09_game_of_life"))
}

func TestMatchesFilterEmptyAllowsEverything(t *testing.T) {
	config := SuiteConfig{}
	assert.True(t, config.MatchesFilter("anything"))
}

func TestRunnerEnabledChecksSet(t *testing.T) {
	config := SuiteConfig{Runners: []RunnerKind{RunnerPiko, RunnerGo}}
	assert.True(t, config.RunnerEnabled(RunnerPiko))
	assert.False(t, config.RunnerEnabled(RunnerCPython))
}
