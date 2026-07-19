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
package generator_adapters_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func newCapturingRegistryMock(shouldFail bool) (
	mock *registry_domain.MockRegistryService,
	lastArtefactID *string,
	lastSourcePath *string,
	lastContent *string,
	lastStorageBackend *string,
	lastDesiredProfiles *[]registry_dto.NamedProfile,
) {
	var (
		aID      string
		sPath    string
		content  string
		backend  string
		profiles []registry_dto.NamedProfile
	)
	lastArtefactID = &aID
	lastSourcePath = &sPath
	lastContent = &content
	lastStorageBackend = &backend
	lastDesiredProfiles = &profiles

	mock = &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID, sourcePath string,
			sourceData io.Reader,
			storageBackendID string,
			desiredProfiles []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			aID = artefactID
			sPath = sourcePath
			backend = storageBackendID
			profiles = desiredProfiles
			if sourceData != nil {
				buffer := new(bytes.Buffer)
				_, _ = buffer.ReadFrom(sourceData)
				content = buffer.String()
			}
			if shouldFail {
				return nil, io.ErrUnexpectedEOF
			}
			return &registry_dto.ArtefactMeta{
				ID:         artefactID,
				SourcePath: sourcePath,
			}, nil
		},
	}
	return
}

func TestPKJSEmitter_EmitJS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		name               string
		source             string
		pagePath           string
		expectedArtefactID string
		expectedSourcePath string
		checkContains      []string
		checkExcludes      []string
		expectEmpty        bool
	}{
		{
			name:               "empty source returns empty artefact ID",
			source:             "",
			pagePath:           "pages/test",
			expectedArtefactID: "",
			expectEmpty:        true,
		},
		{
			name:               "simple typescript is transpiled and stored",
			source:             `const x: number = 42;`,
			pagePath:           "pages/checkout",
			expectedArtefactID: "pk-js/pages/checkout.js",
			expectedSourcePath: "pk/checkout.js",
			checkContains:      []string{"const x", "42"},
			checkExcludes:      []string{": number"},
		},
		{
			name:               "pk extension is stripped from artefact ID",
			source:             `const x = 1;`,
			pagePath:           "pages/cart.pk",
			expectedArtefactID: "pk-js/pages/cart.js",
			expectedSourcePath: "pk/cart.js",
			checkContains:      []string{"const x", "1"},
		},
		{
			name:               "nested path is preserved in artefact ID",
			source:             `const y = "test";`,
			pagePath:           "pages/admin/dashboard",
			expectedArtefactID: "pk-js/pages/admin/dashboard.js",
			expectedSourcePath: "pk/dashboard.js",
		},
		{
			name: "typescript with interface is transpiled",
			source: `
interface User { name: string; }
const user: User = { name: "Alice" };
`,
			pagePath:           "pages/users",
			expectedArtefactID: "pk-js/pages/users.js",
			expectedSourcePath: "pk/users.js",
			checkContains:      []string{"const user", "Alice"},
			checkExcludes:      []string{"interface User", ": User"},
		},
		{
			name:               "actions.gen keeps a js mime via the source path",
			source:             `const a = 1;`,
			pagePath:           "pk/actions.gen",
			expectedArtefactID: "pk-js/pk/actions.gen.js",
			expectedSourcePath: "pk/actions.gen.js",
			checkContains:      []string{"const a", "1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock, lastArtefactID, lastSourcePath, lastContent, _, lastDesiredProfiles := newCapturingRegistryMock(false)
			emitter := generator_adapters.NewPKJSEmitter(mock)
			artefactID, err := emitter.EmitJS(ctx, tc.source, tc.pagePath, "", "", false)
			require.NoError(t, err, "unexpected error")
			assert.Equal(t, tc.expectedArtefactID, artefactID,
				"expected artefact ID %q, got %q", tc.expectedArtefactID, artefactID)
			if tc.expectEmpty {
				assert.Equal(t, int64(0), mock.UpsertArtefactCallCount.Load(),
					"expected no registry call for empty source")
				return
			}
			assert.Equal(t, int64(1), mock.UpsertArtefactCallCount.Load(),
				"expected 1 registry call, got %d", mock.UpsertArtefactCallCount.Load())
			assert.Equal(t, tc.expectedArtefactID, *lastArtefactID,
				"expected stored artefact ID %q, got %q", tc.expectedArtefactID, *lastArtefactID)
			assert.Equal(t, tc.expectedSourcePath, *lastSourcePath,
				"expected stored source path %q, got %q", tc.expectedSourcePath, *lastSourcePath)
			for _, s := range tc.checkContains {
				assert.Contains(t, *lastContent, s,
					"expected content to contain %q, got:\n%s", s, *lastContent)
			}
			for _, s := range tc.checkExcludes {
				assert.NotContains(t, *lastContent, s,
					"expected content NOT to contain %q, got:\n%s", s, *lastContent)
			}
			if *lastDesiredProfiles == nil {
				assert.Fail(t, "expected desired profiles to be set")
			} else {
				assert.True(t, hasProfileNamed(*lastDesiredProfiles, "minified"), "expected 'minified' profile")
				assert.True(t, hasProfileNamed(*lastDesiredProfiles, "gzip"), "expected 'gzip' profile")
				assert.True(t, hasProfileNamed(*lastDesiredProfiles, "br"), "expected 'br' profile")
			}
		})
	}
}
func TestPKJSEmitter_EmitJS_SyntaxError(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, _ := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = {{{", "pages/broken", "", "", false)
	assert.Error(t, err, "expected error for syntax error, got nil")
	assert.Equal(t, int64(0), mock.UpsertArtefactCallCount.Load(),
		"registry should not be called on transpile error")
}
func TestPKJSEmitter_EmitJS_RegistryError(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, _ := newCapturingRegistryMock(true)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = 1;", "pages/test", "", "", false)
	assert.Error(t, err, "expected error when registry fails, got nil")
}
func TestPKJSEmitter_EmitJS_NilRegistry(t *testing.T) {
	t.Parallel()

	emitter := generator_adapters.NewPKJSEmitter(nil)
	ctx := context.Background()
	artefactID, err := emitter.EmitJS(ctx, "const x = 1;", "pages/test", "", "", false)
	require.NoError(t, err, "unexpected error")
	assert.Empty(t, artefactID, "expected empty artefact ID with nil registry, got %q", artefactID)
}
func TestPKJSEmitter_ProfilesPriorityNeed(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, lastDesiredProfiles := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = 1;", "pages/checkout", "", "", false)
	require.NoError(t, err, "unexpected error")
	minifiedProfile, ok := getProfileByName(*lastDesiredProfiles, "minified")
	require.True(t, ok, "expected 'minified' profile")
	assert.Equal(t, registry_dto.PriorityNeed, minifiedProfile.Priority,
		"expected minified profile to have PriorityNeed, got %v", minifiedProfile.Priority)
}

func TestPKJSEmitter_EmitJS_ModuleAliasRewriting(t *testing.T) {
	t.Parallel()

	mock, _, _, lastContent, _, _ := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()

	source := `import { greet } from '@/lib/greeting';
const el = document.querySelector('.output');
if (el) { el.textContent = greet("World"); }`

	_, err := emitter.EmitJS(ctx, source, "pages/test", "github.com/org/repo", "", false)
	require.NoError(t, err, "unexpected error")

	assert.Contains(t, *lastContent, `"/_piko/assets/github.com/org/repo/lib/greeting.js"`,
		"expected @/ import to be rewritten to served asset path, got:\n%s", *lastContent)
	assert.NotContains(t, *lastContent, `@/lib/greeting`,
		"expected @/ alias to be removed from output, got:\n%s", *lastContent)
}

func TestPKJSEmitter_EmitJS_TSExtensionRewriting(t *testing.T) {
	t.Parallel()

	mock, _, _, lastContent, _, _ := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()

	source := `import { helper } from './utils.ts';
console.log(helper());`

	_, err := emitter.EmitJS(ctx, source, "pages/test", "", "", false)
	require.NoError(t, err, "unexpected error")

	assert.Contains(t, *lastContent, `"./utils.js"`,
		"expected .ts import to be rewritten to .js, got:\n%s", *lastContent)
	assert.NotContains(t, *lastContent, `"./utils.ts"`,
		"expected .ts extension to be removed from output, got:\n%s", *lastContent)
}

func hasProfileNamed(profiles []registry_dto.NamedProfile, name string) bool {
	for i := range profiles {
		if profiles[i].Name == name {
			return true
		}
	}
	return false
}

func getProfileByName(profiles []registry_dto.NamedProfile, name string) (registry_dto.DesiredProfile, bool) {
	for i := range profiles {
		if profiles[i].Name == name {
			return profiles[i].Profile, true
		}
	}
	return registry_dto.DesiredProfile{}, false
}
