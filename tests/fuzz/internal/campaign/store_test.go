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

package campaign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestMatchesKeepsTheFailingSourceDistinct(t *testing.T) {
	original := Result{Kind: "target", Target: "formatter", Mode: "package", Phase: "formatter/idempotence", Subject: "pages/main.pk"}
	candidate := original
	candidate.Subject = "components/fuzz-counter.pkc"
	assert.False(t, original.matches(candidate), "an unrelated PKC problem shared a bucket with the failing page")
	assert.NotEqual(t, original.Fingerprint(), candidate.Fingerprint())
	legacy := original
	legacy.Subject = ""
	assert.True(t, legacy.matches(original), "a receipt without a subject must match its source")
	assert.False(t, original.matches(legacy), "a receipt with an identified subject must not match an unidentified one")
	assert.True(t, original.matches(original))
}

func TestMatchesIgnoresMessagesForSubjectlessReceipts(t *testing.T) {
	recorded := Result{Target: "catalog-rebuild", Mode: "interp", Kind: "worker", Phase: "catalog-watch/edit-card", Message: "catalog-watch/edit-card: first message"}
	named := recorded
	named.Subject = "semantic/buy/name"
	named.Message = "catalog-watch/edit-card: second message"
	assert.True(t, recorded.matches(named))
}

func TestCheckpointKeepsRuntimeCoverageOptIn(t *testing.T) {
	dir := t.TempDir()
	old := cleanReport()
	require.NoError(t, checkpoint(dir, &old))
	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "RuntimeCoverage", "empty measurement changed pre-existing report shape")
	assert.Equal(t, uint64(len(data)), old.Retained.CheckpointBytes)
	decoded, err := Load(dir)
	require.NoError(t, err)
	assert.False(t, decoded.Options.RuntimeCoverage)
	assert.Empty(t, decoded.RuntimeCoverage)
}

func TestCheckpointRefusesWhatLoadWouldRefuse(t *testing.T) {
	dir := t.TempDir()
	report := cleanReport()
	report.Active = strings.Repeat("a", maxReportBytes)

	err := checkpoint(dir, &report)

	require.ErrorIs(t, err, errCheckpointOversize)
	assert.NoFileExists(t, filepath.Join(dir, "report.json"))
}

func TestLoadRejectsInvalidReports(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		message string
	}{
		{name: "malformed", content: "{", message: "decoding campaign report"},
		{name: "version", content: `{"Version":9}`, message: "unsupported campaign report version"},
		{name: "options", content: `{"Version":2,"Options":{"Mode":"both","Browser":"off"}}`, message: "invalid campaign report"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "report.json"), []byte(testCase.content), 0o600))
			_, err := Load(dir)
			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestSaveJSONReplacesAtomically(t *testing.T) {
	name := filepath.Join(t.TempDir(), "nested", "value.json")
	require.NoError(t, saveJSON(name, map[string]int{"first": 1}))
	require.NoError(t, saveJSON(name, map[string]int{"second": 2}))
	data, err := os.ReadFile(name)
	require.NoError(t, err)
	assert.JSONEq(t, `{"second":2}`, string(data))
	entries, err := os.ReadDir(filepath.Dir(name))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "temporary checkpoint left behind")
}

func TestNewPublishesTheInitialCheckpoint(t *testing.T) {
	repository := nativeFixture(t)
	state := t.TempDir()
	options := cleanReport().Options

	dir, report, err := New(t.Context(), repository, state, options)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(state, "campaigns"), filepath.Dir(dir))
	assert.Equal(t, "created", report.Status)
	assert.NotEmpty(t, report.Tree)
	assert.Equal(t, generate.Version, report.Generator)
	assert.Contains(t, report.Limits, "GOWORK")
	saved, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, options, saved.Options)
	assert.Equal(t, report.Provenance.WorkspaceSHA256, saved.Provenance.WorkspaceSHA256)
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	repository := nativeFixture(t)
	state := t.TempDir()
	options := cleanReport().Options
	options.Mode = "sideways"

	_, _, err := New(t.Context(), repository, state, options)

	require.ErrorContains(t, err, "invalid campaign mode or browser policy")
	entries, readErr := os.ReadDir(filepath.Join(state, "campaigns"))
	require.NoError(t, readErr)
	assert.Empty(t, entries, "an invalid campaign left a directory behind")
}
