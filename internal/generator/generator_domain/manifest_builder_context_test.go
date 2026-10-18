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

package generator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	contextTestCardImport = "example.com/partials/card.pk"
)

func contextTestPage(name string) *generator_dto.GeneratedArtefact {
	return &generator_dto.GeneratedArtefact{
		SuggestedPath: "/output/pages/" + name + ".go",
		Component: &annotator_dto.VirtualComponent{
			IsPage:                 true,
			IsPublic:               true,
			HashedName:             "pages_" + name,
			CanonicalGoPackagePath: "example.com/pages/" + name,
			Source: &annotator_dto.ParsedComponent{
				SourcePath:    "/project/pages/" + name + ".pk",
				ComponentType: "page",
				Script:        &annotator_dto.ParsedScript{},
				PikoImports:   []annotator_dto.PikoImport{{Path: contextTestCardImport}},
			},
		},
		Result: &annotator_dto.AnnotationResult{},
	}
}

func contextTestCard(jsArtefactID string) *generator_dto.GeneratedArtefact {
	return &generator_dto.GeneratedArtefact{
		SuggestedPath: "/output/partials/card.go",
		JSArtefactID:  jsArtefactID,
		Component: &annotator_dto.VirtualComponent{
			IsPublic:               true,
			HashedName:             "partials_card",
			CanonicalGoPackagePath: "example.com/partials/card",
			Source: &annotator_dto.ParsedComponent{
				SourcePath:       "/project/partials/card.pk",
				ComponentType:    "partial",
				ModuleImportPath: contextTestCardImport,
				Script:           &annotator_dto.ParsedScript{},
			},
		},
		Result: &annotator_dto.AnnotationResult{},
	}
}

func TestManifestBuilder_BuildWithContext(t *testing.T) {
	t.Parallel()

	builder := &ManifestBuilder{baseDir: "/project", pagesSourceDir: "pages"}

	t.Run("a page rebuilt alone loses its partials' scripts without context", func(t *testing.T) {
		t.Parallel()

		manifest, err := builder.Build([]*generator_dto.GeneratedArtefact{contextTestPage("home")})

		require.NoError(t, err)
		assert.Empty(t, manifest.Pages["pages/home.pk"].JSArtefactIDs)
	})

	t.Run("a page rebuilt alone keeps its partials' scripts from context", func(t *testing.T) {
		t.Parallel()

		manifest, err := builder.BuildWithContext(
			[]*generator_dto.GeneratedArtefact{contextTestPage("home")},
			[]*generator_dto.GeneratedArtefact{contextTestPage("about"), contextTestCard("pk-js/partials/card.js")},
		)

		require.NoError(t, err)
		require.Contains(t, manifest.Pages, "pages/home.pk")
		assert.Equal(t, []string{"pk-js/partials/card.js"}, manifest.Pages["pages/home.pk"].JSArtefactIDs)
		assert.NotContains(t, manifest.Pages, "pages/about.pk", "context artefacts are lookups only and get no entries")
		assert.NotContains(t, manifest.Partials, "partials/card.pk")
	})

	t.Run("a rebuilt artefact overrides its context version", func(t *testing.T) {
		t.Parallel()

		manifest, err := builder.BuildWithContext(
			[]*generator_dto.GeneratedArtefact{contextTestPage("home"), contextTestCard("")},
			[]*generator_dto.GeneratedArtefact{contextTestCard("pk-js/partials/card.js")},
		)

		require.NoError(t, err)
		assert.Empty(t, manifest.Pages["pages/home.pk"].JSArtefactIDs,
			"the rebuilt partial dropped its script, so the stale context script must not be kept")
		assert.Contains(t, manifest.Partials, "partials/card.pk")
	})
}
