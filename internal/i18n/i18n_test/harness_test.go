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

package i18n_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/i18n/i18n_adapters"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/safedisk"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type TestSpec struct {
	Description   string `json:"description"`
	ErrorContains string `json:"errorContains,omitempty"`
	ShouldError   bool   `json:"shouldError,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	spec := loadTestSpec(t, tc)

	i18nDir := filepath.Join(tc.Path, "i18n")
	absI18nDir, err := filepath.Abs(i18nDir)
	require.NoError(t, err)

	baseDir := filepath.Dir(absI18nDir)

	testSandbox, _ := safedisk.NewNoOpSandbox(baseDir, safedisk.ModeReadOnly)
	defer func() { _ = testSandbox.Close() }()

	service, err := i18n_adapters.NewFSService(context.Background(), testSandbox, "", "i18n")

	if spec.ShouldError {
		require.Error(t, err, "Expected i18n_adapters.NewFSService to fail, but it succeeded for: %s", tc.Name)
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains, "The error message did not contain the expected text")
		}
		return
	}
	require.NoError(t, err, "i18n_adapters.NewFSService failed unexpectedly for test case: %s", tc.Name)
	require.NotNil(t, service, "Service should not be nil on success")

	store := service.GetStore()
	require.NotNil(t, store, "Store should not be nil on success")

	translations := make(i18n_domain.Translations)
	for _, locale := range store.Locales() {
		entries := store.GetEntriesForLocale(locale)
		if entries == nil {
			continue
		}
		localeMap := make(map[string]string, len(entries))
		for key, entry := range entries {
			localeMap[key] = entry.Template
		}
		translations[locale] = localeMap
	}

	goldenPath := filepath.Join(tc.Path, "golden", "golden-translations.json")

	translationsBytes, jsonErr := json.ConfigStd.MarshalIndent(translations, "", "  ")
	require.NoError(t, jsonErr, "Failed to marshal translations to JSON")

	assertGoldenFileJSON(t, goldenPath, translationsBytes, "Translations for %s", tc.Name)
}

func loadTestSpec(t *testing.T, tc testCase) TestSpec {
	t.Helper()
	var spec TestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	if os.IsNotExist(err) {
		return TestSpec{
			Description: "Default test case",
		}
	}
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

func assertGoldenFileJSON(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()
	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, actualBytes, 0644))
	}
	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	assert.JSONEq(t, string(expectedBytes), string(actualBytes), msgAndArgs...)
}
