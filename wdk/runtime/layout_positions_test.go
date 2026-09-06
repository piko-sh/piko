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

package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestExtractLayoutPositions_ReportsManifestProblems(t *testing.T) {
	t.Parallel()

	emptyManifestDirectory := t.TempDir()
	writeEmptyManifest(t, emptyManifestDirectory)

	testCases := []struct {
		name         string
		manifestPath string
		requestPath  string
		wantErr      string
	}{
		{
			name:         "a missing manifest directory",
			manifestPath: filepath.Join(t.TempDir(), "absent", "manifest.bin"),
			requestPath:  "/",
			wantErr:      "opening manifest directory",
		},
		{
			name:         "a directory without a manifest",
			manifestPath: filepath.Join(t.TempDir(), "manifest.bin"),
			requestPath:  "/",
			wantErr:      "loading manifest",
		},
		{
			name:         "a page the manifest does not contain",
			manifestPath: filepath.Join(emptyManifestDirectory, "manifest.bin"),
			requestPath:  "/missing",
			wantErr:      "page entry not found",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := LayoutPositionConfig{}
			config.ManifestPath = testCase.manifestPath
			config.RequestPath = testCase.requestPath

			positions, err := ExtractLayoutPositions(context.Background(), config)

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErr)
			assert.Nil(t, positions)
		})
	}
}

func writeEmptyManifest(t *testing.T, directory string) {
	t.Helper()

	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })
	manifest := &generator_dto.Manifest{}
	manifest.Pages = map[string]generator_dto.ManifestPageEntry{}
	manifest.Partials = map[string]generator_dto.ManifestPartialEntry{}
	manifest.Emails = map[string]generator_dto.ManifestEmailEntry{}
	require.NoError(t, generator_adapters.NewFlatBufferManifestEmitter(sandbox).EmitCode(context.Background(), manifest, "manifest.bin"))
}
