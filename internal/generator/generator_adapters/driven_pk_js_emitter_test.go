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
	"strings"
	"sync/atomic"
	"testing"

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
			checkContains:      []string{"const x", "42"},
			checkExcludes:      []string{": number"},
		},
		{
			name:               "pk extension is stripped from artefact ID",
			source:             `const x = 1;`,
			pagePath:           "pages/cart.pk",
			expectedArtefactID: "pk-js/pages/cart.js",
			checkContains:      []string{"const x", "1"},
		},
		{
			name:               "nested path is preserved in artefact ID",
			source:             `const y = "test";`,
			pagePath:           "pages/admin/dashboard",
			expectedArtefactID: "pk-js/pages/admin/dashboard.js",
		},
		{
			name: "typescript with interface is transpiled",
			source: `
interface User { name: string; }
const user: User = { name: "Alice" };
`,
			pagePath:           "pages/users",
			expectedArtefactID: "pk-js/pages/users.js",
			checkContains:      []string{"const user", "Alice"},
			checkExcludes:      []string{"interface User", ": User"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock, lastArtefactID, _, lastContent, _, lastDesiredProfiles := newCapturingRegistryMock(false)
			emitter := generator_adapters.NewPKJSEmitter(mock)
			artefactID, err := emitter.EmitJS(ctx, tc.source, tc.pagePath, "", false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if artefactID != tc.expectedArtefactID {
				t.Errorf("expected artefact ID %q, got %q", tc.expectedArtefactID, artefactID)
			}
			if tc.expectEmpty {
				if atomic.LoadInt64(&mock.UpsertArtefactCallCount) != 0 {
					t.Error("expected no registry call for empty source")
				}
				return
			}
			if atomic.LoadInt64(&mock.UpsertArtefactCallCount) != 1 {
				t.Errorf("expected 1 registry call, got %d", atomic.LoadInt64(&mock.UpsertArtefactCallCount))
			}
			if *lastArtefactID != tc.expectedArtefactID {
				t.Errorf("expected stored artefact ID %q, got %q", tc.expectedArtefactID, *lastArtefactID)
			}
			for _, s := range tc.checkContains {
				if !strings.Contains(*lastContent, s) {
					t.Errorf("expected content to contain %q, got:\n%s", s, *lastContent)
				}
			}
			for _, s := range tc.checkExcludes {
				if strings.Contains(*lastContent, s) {
					t.Errorf("expected content NOT to contain %q, got:\n%s", s, *lastContent)
				}
			}
			if *lastDesiredProfiles == nil {
				t.Error("expected desired profiles to be set")
			} else {
				if !hasProfileNamed(*lastDesiredProfiles, "minified") {
					t.Error("expected 'minified' profile")
				}
				if !hasProfileNamed(*lastDesiredProfiles, "gzip") {
					t.Error("expected 'gzip' profile")
				}
				if !hasProfileNamed(*lastDesiredProfiles, "br") {
					t.Error("expected 'br' profile")
				}
			}
		})
	}
}
func TestPKJSEmitter_EmitJS_SyntaxError(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, _ := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = {{{", "pages/broken", "", false)
	if err == nil {
		t.Error("expected error for syntax error, got nil")
	}
	if atomic.LoadInt64(&mock.UpsertArtefactCallCount) != 0 {
		t.Error("registry should not be called on transpile error")
	}
}
func TestPKJSEmitter_EmitJS_RegistryError(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, _ := newCapturingRegistryMock(true)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = 1;", "pages/test", "", false)
	if err == nil {
		t.Error("expected error when registry fails, got nil")
	}
}
func TestPKJSEmitter_EmitJS_NilRegistry(t *testing.T) {
	t.Parallel()

	emitter := generator_adapters.NewPKJSEmitter(nil)
	ctx := context.Background()
	artefactID, err := emitter.EmitJS(ctx, "const x = 1;", "pages/test", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artefactID != "" {
		t.Errorf("expected empty artefact ID with nil registry, got %q", artefactID)
	}
}
func TestPKJSEmitter_ProfilesPriorityNeed(t *testing.T) {
	t.Parallel()

	mock, _, _, _, _, lastDesiredProfiles := newCapturingRegistryMock(false)
	emitter := generator_adapters.NewPKJSEmitter(mock)
	ctx := context.Background()
	_, err := emitter.EmitJS(ctx, "const x = 1;", "pages/checkout", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	minifiedProfile, ok := getProfileByName(*lastDesiredProfiles, "minified")
	if !ok {
		t.Fatal("expected 'minified' profile")
	}
	if minifiedProfile.Priority != registry_dto.PriorityNeed {
		t.Errorf("expected minified profile to have PriorityNeed, got %v", minifiedProfile.Priority)
	}
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
