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

package markdown_collection_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_adapters/driver_markdown"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/tests/integration/testutil"
	"piko.sh/piko/wdk/safedisk"
)

func createProvider(t *testing.T, testdataDir string) (*driver_markdown.MarkdownProvider, collection_dto.ContentSource) {
	t.Helper()

	parser := markdown_provider_goldmark.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)

	sandbox, err := safedisk.NewNoOpSandbox(testdataDir, safedisk.ModeReadOnly)
	require.NoError(t, err, "Failed to create sandbox for %s", testdataDir)
	t.Cleanup(func() { _ = sandbox.Close() })

	source := collection_dto.ContentSource{
		Sandbox:  sandbox,
		BasePath: testdataDir,
	}

	return driver_markdown.NewMarkdownProvider("markdown", sandbox, service, nil, nil), source
}

func TestMarkdownCollection(t *testing.T) {
	cases, err := testutil.DiscoverTestCases("testdata")
	require.NoError(t, err)
	require.NotEmpty(t, cases, "No test cases found in testdata/")

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			provider, _ := createProvider(t, tc.Path)

			config := collection_dto.ProviderConfig{
				BasePath:      tc.Path,
				DefaultLocale: "en",
				Locales:       []string{"en", "fr", "de", "es"},
			}

			collections, err := provider.DiscoverCollections(ctx, config)
			require.NoError(t, err, "DiscoverCollections failed")

			var allItems []collection_dto.ContentItem
			for _, col := range collections {
				items, fetchErr := provider.FetchStaticContent(ctx, col.Name, collection_dto.ContentSource{})
				require.NoError(t, fetchErr, "FetchStaticContent failed for collection %q", col.Name)
				allItems = append(allItems, items...)
			}

			snapshots := snapshotItems(allItems)
			jsonBytes := marshalJSON(t, snapshots)

			goldenDir := filepath.Join(tc.Path, "golden")
			itemsGoldenPath := filepath.Join(goldenDir, "items.golden.json")
			testutil.AssertGoldenJSON(t, itemsGoldenPath, jsonBytes)

			navGoldenPath := filepath.Join(goldenDir, "navigation.golden.json")
			_, statErr := os.Stat(navGoldenPath)
			shouldTestNav := statErr == nil || (testutil.UpdateGolden != nil && *testutil.UpdateGolden)

			if shouldTestNav && len(allItems) > 0 {
				builder := collection_domain.NewNavigationBuilder()
				navConfig := collection_dto.DefaultNavigationConfig()
				navConfig.Locale = "en"
				groups := builder.BuildNavigationGroups(context.Background(), allItems, navConfig)

				navSnapshot := snapshotNavigation(groups)
				navJSON := marshalJSON(t, navSnapshot)
				testutil.AssertGoldenJSON(t, navGoldenPath, navJSON)
			}
		})
	}
}
