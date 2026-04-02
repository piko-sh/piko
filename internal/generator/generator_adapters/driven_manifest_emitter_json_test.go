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
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewJSONManifestEmitter(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()
	emitter := NewJSONManifestEmitter(sandbox)

	require.NotNil(t, emitter)
}

func TestJSONManifestEmitter_EmitCode(t *testing.T) {
	t.Parallel()

	t.Run("writes empty manifest successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		emitter := NewJSONManifestEmitter(sandbox)

		manifest := &generator_dto.Manifest{
			Pages:    map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}

		err := emitter.EmitCode(context.Background(), manifest, "manifest.json")

		require.NoError(t, err)

		data, readErr := sandbox.ReadFile("manifest.json")
		require.NoError(t, readErr)
		assert.NotEmpty(t, data)
	})

	t.Run("writes manifest with pages", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		emitter := NewJSONManifestEmitter(sandbox)

		manifest := &generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{
				"pages/home.pk": {
					PackagePath:        "test.com/pages/home",
					OriginalSourcePath: "pages/home.pk",
					RoutePatterns:      map[string]string{"en": "/home"},
					I18nStrategy:       "prefix",
					StyleBlock:         ".home { color: red; }",
					AssetRefs:          []templater_dto.AssetRef{{Kind: "image", Path: "/img/logo.svg"}},
					CustomTags:         []string{"custom-tag"},
					LocalTranslations: i18n_domain.Translations{
						"en": {"greeting": "Hello"},
					},
				},
			},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}

		err := emitter.EmitCode(context.Background(), manifest, "manifest.json")

		require.NoError(t, err)

		data, readErr := sandbox.ReadFile("manifest.json")
		require.NoError(t, readErr)

		var unmarshalled generator_dto.Manifest
		unmarshalErr := json.Unmarshal(data, &unmarshalled)
		require.NoError(t, unmarshalErr)
		assert.Len(t, unmarshalled.Pages, 1)
	})

	t.Run("writes manifest with partials", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		emitter := NewJSONManifestEmitter(sandbox)

		manifest := &generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{
				"partials/card.pk": {
					PackagePath:        "test.com/partials/card",
					OriginalSourcePath: "partials/card.pk",
					PartialName:        "partials-card",
					PartialSrc:         "/_piko/partial/partials-card",
					RoutePattern:       "/_piko/partial/partials-card",
					StyleBlock:         ".card { padding: 1rem; }",
				},
			},
			Emails: map[string]generator_dto.ManifestEmailEntry{},
		}

		err := emitter.EmitCode(context.Background(), manifest, "manifest.json")

		require.NoError(t, err)

		data, _ := sandbox.ReadFile("manifest.json")
		var unmarshalled generator_dto.Manifest
		_ = json.Unmarshal(data, &unmarshalled)
		assert.Len(t, unmarshalled.Partials, 1)
	})

	t.Run("writes manifest with emails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		emitter := NewJSONManifestEmitter(sandbox)

		manifest := &generator_dto.Manifest{
			Pages:    map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails: map[string]generator_dto.ManifestEmailEntry{
				"emails/welcome.pk": {
					PackagePath:         "test.com/emails/welcome",
					OriginalSourcePath:  "emails/welcome.pk",
					StyleBlock:          "table { border-collapse: collapse; }",
					HasSupportedLocales: true,
					LocalTranslations: i18n_domain.Translations{
						"en": {"subject": "Welcome"},
					},
				},
			},
		}

		err := emitter.EmitCode(context.Background(), manifest, "manifest.json")

		require.NoError(t, err)

		data, _ := sandbox.ReadFile("manifest.json")
		var unmarshalled generator_dto.Manifest
		_ = json.Unmarshal(data, &unmarshalled)
		assert.Len(t, unmarshalled.Emails, 1)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		emitter := NewJSONManifestEmitter(sandbox)

		manifest := &generator_dto.Manifest{
			Pages:    map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}

		err := emitter.EmitCode(context.Background(), manifest, "output/directory/manifest.json")

		require.NoError(t, err)
		assert.Equal(t, 1, sandbox.CallCounts["MkdirAll"])
	})
}

func TestJSONManifestEmitter_EmitCode_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupMock func(*safedisk.MockSandbox)
		name      string
	}{
		{
			name: "MkdirAll error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.MkdirAllErr = errors.New("cannot create directory")
			},
		},
		{
			name: "CreateTemp error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.CreateTempErr = errors.New("disk full")
			},
		},
		{
			name: "Write error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileWriteErr = errors.New("write failed")
			},
		},
		{
			name: "Sync error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileSyncErr = errors.New("sync failed")
			},
		},
		{
			name: "Close error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileCloseErr = errors.New("close failed")
			},
		},
		{
			name: "Chmod error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.ChmodErr = errors.New("permission denied")
			},
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("rename failed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
			defer sandbox.Close()
			tc.setupMock(sandbox)

			emitter := NewJSONManifestEmitter(sandbox)

			manifest := &generator_dto.Manifest{
				Pages:    map[string]generator_dto.ManifestPageEntry{},
				Partials: map[string]generator_dto.ManifestPartialEntry{},
				Emails:   map[string]generator_dto.ManifestEmailEntry{},
			}

			err := emitter.EmitCode(context.Background(), manifest, "manifest.json")

			require.Error(t, err)
		})
	}
}
