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

package lifecycle_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		code        string
		expected    string
		expectError bool
	}{
		{
			name:        "simple package declaration",
			code:        "package main",
			expected:    "main",
			expectError: false,
		},
		{
			name:        "package with trailing comment",
			code:        "package main // this is a comment",
			expected:    "main",
			expectError: false,
		},
		{
			name:        "package with leading whitespace",
			code:        "   package mypackage",
			expected:    "mypackage",
			expectError: false,
		},
		{
			name:        "package after comment line",
			code:        "// copyright header\npackage mypackage",
			expected:    "mypackage",
			expectError: false,
		},
		{
			name:        "package after multiple comment lines",
			code:        "// line 1\n// line 2\n// line 3\npackage mypackage",
			expected:    "mypackage",
			expectError: false,
		},
		{
			name:        "package with extra spaces",
			code:        "package    main   ",
			expected:    "main",
			expectError: false,
		},
		{
			name:        "missing package declaration",
			code:        "func main() {}",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty code",
			code:        "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "package keyword in comment only",
			code:        "// package foo\nfunc bar() {}",
			expected:    "",
			expectError: true,
		},
		{
			name:        "underscore package name",
			code:        "package _",
			expected:    "_",
			expectError: false,
		},
		{
			name:        "package with inline comment containing package keyword",
			code:        "package main // package comment",
			expected:    "main",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := extractPackageName(tc.code)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "missing package declaration")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestCreatePageEntryFromManifest(t *testing.T) {
	t.Parallel()

	t.Run("returns page entry for existing page", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{
				"pages/home.pk": {
					PackagePath:        "example.com/project/pages/home",
					OriginalSourcePath: "pages/home.pk",
					RoutePatterns:      map[string]string{"": "/"},
				},
			},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}

		result := createPageEntryFromManifest(manifest, "pages/home.pk")

		require.NotNil(t, result)
		assert.Equal(t, "example.com/project/pages/home", result.PackagePath)
		assert.Equal(t, "/", result.RoutePatterns[""])
	})

	t.Run("returns page entry for existing partial", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{
				"partials/header.pk": {
					PackagePath:        "example.com/project/partials/header",
					OriginalSourcePath: "partials/header.pk",
					PartialSrc:         "/_partials/header",
				},
			},
			Emails: map[string]generator_dto.ManifestEmailEntry{},
		}

		result := createPageEntryFromManifest(manifest, "partials/header.pk")

		require.NotNil(t, result)
		assert.Equal(t, "example.com/project/partials/header", result.PackagePath)
		assert.Equal(t, "/_partials/header", result.RoutePatterns[""])
	})

	t.Run("returns page entry for existing email", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Pages:    map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails: map[string]generator_dto.ManifestEmailEntry{
				"emails/welcome.pk": {
					PackagePath:         "example.com/project/emails/welcome",
					OriginalSourcePath:  "emails/welcome.pk",
					HasSupportedLocales: true,
				},
			},
		}

		result := createPageEntryFromManifest(manifest, "emails/welcome.pk")

		require.NotNil(t, result)
		assert.Equal(t, "example.com/project/emails/welcome", result.PackagePath)
		assert.True(t, result.HasSupportedLocales)
	})

	t.Run("returns nil for non-existent path", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Pages:    map[string]generator_dto.ManifestPageEntry{},
			Partials: map[string]generator_dto.ManifestPartialEntry{},
			Emails:   map[string]generator_dto.ManifestEmailEntry{},
		}

		result := createPageEntryFromManifest(manifest, "unknown/path.pk")

		assert.Nil(t, result)
	})

	t.Run("prioritises pages over partials with same path", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Pages: map[string]generator_dto.ManifestPageEntry{
				"shared/component.pk": {
					PackagePath: "example.com/project/pages/component",
				},
			},
			Partials: map[string]generator_dto.ManifestPartialEntry{
				"shared/component.pk": {
					PackagePath: "example.com/project/partials/component",
				},
			},
			Emails: map[string]generator_dto.ManifestEmailEntry{},
		}

		result := createPageEntryFromManifest(manifest, "shared/component.pk")

		require.NotNil(t, result)

		assert.Equal(t, "example.com/project/pages/component", result.PackagePath)
	})
}

func TestNewInterpretedBuildOrchestrator(t *testing.T) {
	t.Parallel()

	t.Run("creates orchestrator with all fields initialised", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				I18nDefaultLocale: "en-GB",
				ModuleName:        "example.com/project",
				ProjectRoot:       "/home/user/project",
			},
		)

		require.NotNil(t, orchestrator)
		assert.Equal(t, "example.com/project", orchestrator.moduleName)
		assert.Equal(t, "/home/user/project", orchestrator.projectRoot)
		assert.NotNil(t, orchestrator.progCache)
		assert.NotNil(t, orchestrator.dirtyCodeCache)
		assert.NotNil(t, orchestrator.reverseDepsMap)
		assert.NotNil(t, orchestrator.artefactByPackagePath)
		assert.NotNil(t, orchestrator.interpSemaphore)
	})

	t.Run("creates semaphore with at least 1 slot", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		select {
		case orchestrator.interpSemaphore <- struct{}{}:

			<-orchestrator.interpSemaphore
		default:
			t.Fatal("Expected semaphore to have at least 1 slot")
		}
	})
}

func TestInterpretedBuildOrchestrator_IsInitialised(t *testing.T) {
	t.Parallel()

	t.Run("returns false when vfsAdapter is nil", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		assert.False(t, orchestrator.IsInitialised())
	})
}

func TestInterpretedBuildOrchestrator_getDefaultLocale(t *testing.T) {
	t.Parallel()

	t.Run("returns configured locale when set", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				I18nDefaultLocale: "de-DE",
				ModuleName:        "module",
				ProjectRoot:       "/root",
			},
		)

		assert.Equal(t, "de-DE", orchestrator.getDefaultLocale())
	})

	t.Run("returns 'en' when locale not configured", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		assert.Equal(t, "en", orchestrator.getDefaultLocale())
	})
}

func TestInterpretedBuildOrchestrator_isEmptyVirtualModule(t *testing.T) {
	t.Parallel()

	orchestrator := NewInterpretedBuildOrchestrator(
		InterpretedBuildOrchestratorDeps{
			ModuleName:  "module",
			ProjectRoot: "/root",
		},
	)

	t.Run("returns true when virtual module is nil", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: nil,
		}

		assert.True(t, orchestrator.isEmptyVirtualModule(result))
	})

	t.Run("returns true when components map is empty", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}

		assert.True(t, orchestrator.isEmptyVirtualModule(result))
	})

	t.Run("returns false when components exist", func(t *testing.T) {
		t.Parallel()

		result := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"abc123": {},
				},
			},
		}

		assert.False(t, orchestrator.isEmptyVirtualModule(result))
	})
}

func TestInterpretedBuildOrchestrator_GetCachedEntry(t *testing.T) {
	t.Parallel()

	t.Run("returns false when cache is empty", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		entry, found := orchestrator.GetCachedEntry("pages/home.pk")

		assert.False(t, found)
		assert.Nil(t, entry)
	})
}

func TestInterpretedBuildOrchestrator_GetAllCachedKeys(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when cache is empty", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		keys := orchestrator.GetAllCachedKeys()

		assert.Empty(t, keys)
	})
}

func TestInterpretedBuildOrchestrator_isComponentDirty(t *testing.T) {
	t.Parallel()

	t.Run("returns false when dirty cache is empty", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		assert.False(t, orchestrator.isComponentDirty("pages/home.pk"))
	})

	t.Run("returns true when component is in dirty cache", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewInterpretedBuildOrchestrator(
			InterpretedBuildOrchestratorDeps{
				ModuleName:  "module",
				ProjectRoot: "/root",
			},
		)

		orchestrator.dirtyCodeCache["pages/home.pk"] = []byte("code")

		assert.True(t, orchestrator.isComponentDirty("pages/home.pk"))
	})
}

func TestParseLocalImportPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		source       string
		modulePrefix string
		expected     []string
	}{
		{
			name:         "standard local import",
			source:       "package main\n\nimport \"mymod/pkg/domain\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/domain"},
		},
		{
			name:         "aliased local import",
			source:       "package main\n\nimport alias \"mymod/pkg/domain\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/domain"},
		},
		{
			name:         "blank import",
			source:       "package main\n\nimport _ \"mymod/pkg/init\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/init"},
		},
		{
			name:         "dot import",
			source:       "package main\n\nimport . \"mymod/pkg/utils\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/utils"},
		},
		{
			name:         "mixed local and external imports",
			source:       "package main\n\nimport (\n\t\"fmt\"\n\t\"mymod/pkg/api\"\n\t\"strings\"\n)\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/api"},
		},
		{
			name:         "multiple local imports",
			source:       "package main\n\nimport (\n\t\"mymod/pkg/a\"\n\t\"mymod/pkg/b\"\n)\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/pkg/a", "mymod/pkg/b"},
		},
		{
			name:         "no local imports",
			source:       "package main\n\nimport \"fmt\"\n",
			modulePrefix: "mymod/",
			expected:     nil,
		},
		{
			name:         "no imports at all",
			source:       "package main\n",
			modulePrefix: "mymod/",
			expected:     nil,
		},
		{
			name:         "malformed source returns nil",
			source:       "not valid go code",
			modulePrefix: "mymod/",
			expected:     nil,
		},
		{
			name:         "module prefix safety prevents partial match",
			source:       "package main\n\nimport \"mymodule/pkg/a\"\n",
			modulePrefix: "mymod/",
			expected:     nil,
		},
		{
			name:         "hashed alias from partial import",
			source:       "package main\n\nimport partials_card_bfc4a3cf \"mymod/dist/partials/partials_card_bfc4a3cf\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/dist/partials/partials_card_bfc4a3cf"},
		},
		{
			name:         "deeply nested import path",
			source:       "package main\n\nimport \"mymod/internal/foo/bar/baz\"\n",
			modulePrefix: "mymod/",
			expected:     []string{"mymod/internal/foo/bar/baz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseLocalImportPaths(tc.source, tc.modulePrefix)

			assert.Equal(t, tc.expected, result)
		})
	}
}
