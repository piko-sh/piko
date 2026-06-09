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

package coordinator_domain

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestRememberStyleDeps(t *testing.T) {
	t.Parallel()

	t.Run("nil result is a no-op", func(t *testing.T) {
		t.Parallel()

		s := &coordinatorService{}
		s.rememberStyleDeps(nil)
		assert.Nil(t, s.getKnownStyleDeps())
	})

	t.Run("stores imported paths per component and skips nil results", func(t *testing.T) {
		t.Parallel()

		s := &coordinatorService{}
		s.rememberStyleDeps(&annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hashA": {ImportedStylePaths: []string{"/proj/a.css"}},
				"hashB": nil,
			},
		})
		assert.Equal(t, []string{"/proj/a.css"}, s.getKnownStyleDeps())
	})

	t.Run("drops a component whose imports become empty", func(t *testing.T) {
		t.Parallel()

		s := &coordinatorService{}
		s.rememberStyleDeps(&annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hashA": {ImportedStylePaths: []string{"/proj/a.css"}},
			},
		})
		s.rememberStyleDeps(&annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hashA": {ImportedStylePaths: nil},
			},
		})
		assert.Nil(t, s.getKnownStyleDeps())
	})
}

func TestGetKnownStyleDeps(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when empty", func(t *testing.T) {
		t.Parallel()

		s := &coordinatorService{}
		assert.Nil(t, s.getKnownStyleDeps())
	})

	t.Run("returns the de-duplicated union across components", func(t *testing.T) {
		t.Parallel()

		s := &coordinatorService{
			knownStyleDeps: map[string][]string{
				"hashA": {"/proj/shared.css", "/proj/a.css"},
				"hashB": {"/proj/shared.css", "/proj/b.css"},
			},
		}
		got := s.getKnownStyleDeps()
		slices.Sort(got)
		assert.Equal(t, []string{"/proj/a.css", "/proj/b.css", "/proj/shared.css"}, got)
	})
}

func TestIncludeKnownStyleDeps(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	existing := filepath.Join(tempDir, "theme.css")
	require.NoError(t, os.WriteFile(existing, []byte(".a{}"), 0o600))
	missing := filepath.Join(tempDir, "gone.css")
	alreadyTracked := filepath.Join(tempDir, "tracked.css")

	s := &coordinatorService{
		resolver:       &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return tempDir }},
		knownStyleDeps: map[string][]string{"hashA": {existing, missing, alreadyTracked}},
	}

	allSourceFiles := map[string]struct{}{alreadyTracked: {}}
	s.includeKnownStyleDeps(t.Context(), allSourceFiles)

	_, hasExisting := allSourceFiles[existing]
	_, hasMissing := allSourceFiles[missing]
	assert.True(t, hasExisting, "an existing stylesheet should be added to the hashable set")
	assert.False(t, hasMissing, "a deleted stylesheet should be skipped")
	assert.Len(t, allSourceFiles, 2)
}
