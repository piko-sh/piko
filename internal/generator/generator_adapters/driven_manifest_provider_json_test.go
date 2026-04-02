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

package generator_adapters

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewJSONManifestProvider(t *testing.T) {
	t.Parallel()

	t.Run("creates provider with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/manifest", safedisk.ModeReadOnly)
		defer sandbox.Close()

		provider := NewJSONManifestProvider(
			"/manifest/manifest.json",
			WithJSONManifestSandbox(sandbox),
		)

		require.NotNil(t, provider)
		assert.Equal(t, sandbox, provider.sandbox)
		assert.Equal(t, "manifest.json", provider.manifestFileName)
	})

	t.Run("creates provider with default sandbox when none injected", func(t *testing.T) {
		t.Parallel()

		provider := NewJSONManifestProvider("/tmp/test-json-manifest/manifest.json")

		require.NotNil(t, provider)
		assert.Equal(t, "manifest.json", provider.manifestFileName)
	})
}

func TestJSONManifestProvider_Load(t *testing.T) {
	t.Parallel()

	t.Run("returns error when sandbox is nil", func(t *testing.T) {
		t.Parallel()

		provider := &JSONManifestProvider{
			sandbox:          nil,
			manifestFileName: "manifest.json",
		}

		_, err := provider.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sandbox not available")
	})

	t.Run("returns error when manifest file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/manifest", safedisk.ModeReadOnly)
		defer sandbox.Close()

		provider := NewJSONManifestProvider(
			"/manifest/manifest.json",
			WithJSONManifestSandbox(sandbox),
		)

		_, err := provider.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("loads valid manifest file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/manifest", safedisk.ModeReadWrite)
		defer sandbox.Close()

		manifest := generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{
				"pages/index.pk": {
					PackagePath:        "test.com/pages/index",
					OriginalSourcePath: "pages/index.pk",
				},
			},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}
		data, _ := json.Marshal(manifest)
		require.NoError(t, sandbox.WriteFile("manifest.json", data, 0600))

		provider := NewJSONManifestProvider(
			"/manifest/manifest.json",
			WithJSONManifestSandbox(sandbox),
		)

		loaded, err := provider.Load(context.Background())

		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Len(t, loaded.Pages, 1)
		assert.Equal(t, "test.com/pages/index", loaded.Pages["pages/index.pk"].PackagePath)
	})

	t.Run("returns error on corrupted JSON", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/manifest", safedisk.ModeReadWrite)
		defer sandbox.Close()
		require.NoError(t, sandbox.WriteFile("manifest.json", []byte("not valid json{{{"), 0600))

		provider := NewJSONManifestProvider(
			"/manifest/manifest.json",
			WithJSONManifestSandbox(sandbox),
		)

		_, err := provider.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "corrupt manifest")
	})
}

func TestJSONManifestProvider_Load_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when ReadFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/manifest", safedisk.ModeReadWrite)
		defer sandbox.Close()
		require.NoError(t, sandbox.WriteFile("manifest.json", []byte("{}"), 0600))
		sandbox.ReadFileErr = errors.New("disk read error")

		provider := NewJSONManifestProvider(
			"/manifest/manifest.json",
			WithJSONManifestSandbox(sandbox),
		)

		_, err := provider.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read manifest file")
	})
}
