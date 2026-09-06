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

package templater_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
)

func TestNewPartialPageEntry_CopiesRuntimeFields(t *testing.T) {
	t.Parallel()

	entry := NewPartialPageEntry(generator_dto.ManifestPartialEntry{
		PackagePath:        "project/dist/partials/card",
		OriginalSourcePath: "partials/card.pk",
		PartialName:        "partials-card",
		PartialSrc:         "/_piko/partial/partials-card",
		RoutePattern:       "/_piko/partial/partials-card",
		StyleBlock:         ".card{}",
		JSArtefactID:       "pk-js/partials/card.js",
		IsE2EOnly:          true,
		HasPreview:         true,
	})

	assert.Equal(t, "project/dist/partials/card", entry.PackagePath)
	assert.Equal(t, "partials/card.pk", entry.OriginalSourcePath)
	assert.Equal(t, map[string]string{"": "/_piko/partial/partials-card"}, entry.RoutePatterns)
	assert.Equal(t, ".card{}", entry.StyleBlock)
	assert.Equal(t, []string{"pk-js/partials/card.js"}, entry.JSArtefactIDs)
	assert.True(t, entry.GetIsE2EOnly(), "an E2E-only partial must stay behind the E2E guard")
	assert.True(t, entry.HasPreview)
}

func TestNewPartialPageEntry_WithoutJSArtefact(t *testing.T) {
	t.Parallel()

	entry := NewPartialPageEntry(generator_dto.ManifestPartialEntry{
		PackagePath:        "project/dist/partials/static",
		OriginalSourcePath: "partials/static.pk",
		PartialSrc:         "/_piko/partial/partials-static",
	})

	assert.Nil(t, entry.JSArtefactIDs)
	assert.False(t, entry.GetIsE2EOnly())
	assert.False(t, entry.HasPreview)
}

func TestNewEmailPageEntry_CopiesEveryField(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{"en": {"greeting": "Hello"}}
	entry := NewEmailPageEntry(generator_dto.ManifestEmailEntry{
		LocalTranslations:   translations,
		PackagePath:         "project/dist/emails/welcome",
		OriginalSourcePath:  "emails/welcome.pk",
		StyleBlock:          ".email{}",
		HasSupportedLocales: true,
		HasPreview:          true,
	})

	assert.Equal(t, "project/dist/emails/welcome", entry.PackagePath)
	assert.Equal(t, "emails/welcome.pk", entry.OriginalSourcePath)
	assert.Equal(t, ".email{}", entry.StyleBlock)
	assert.Equal(t, translations, entry.LocalTranslations)
	assert.True(t, entry.HasSupportedLocales)
	assert.True(t, entry.HasPreview)
}

func TestNewPdfPageEntry_CopiesEveryField(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{"en": {"title": "Invoice"}}
	entry := NewPdfPageEntry(generator_dto.ManifestPdfEntry{
		LocalTranslations:   translations,
		PackagePath:         "project/dist/pdfs/invoice",
		OriginalSourcePath:  "pdfs/invoice.pk",
		StyleBlock:          ".pdf{}",
		HasSupportedLocales: true,
		HasPreview:          true,
	})

	assert.Equal(t, "project/dist/pdfs/invoice", entry.PackagePath)
	assert.Equal(t, "pdfs/invoice.pk", entry.OriginalSourcePath)
	assert.Equal(t, ".pdf{}", entry.StyleBlock)
	assert.Equal(t, translations, entry.LocalTranslations)
	assert.True(t, entry.HasSupportedLocales)
	assert.True(t, entry.HasPreview)
}

func TestNewErrorPageDispatch_CopiesRoutingMetadata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		want      ErrorPageDispatch
		errorData generator_dto.ManifestErrorPageEntry
	}{
		{
			name:      "exact status page",
			errorData: generator_dto.ManifestErrorPageEntry{ScopePath: "/app/", StatusCode: 404},
			want:      ErrorPageDispatch{ScopePath: "/app/", StatusCode: 404},
		},
		{
			name:      "status range page",
			errorData: generator_dto.ManifestErrorPageEntry{ScopePath: "/", StatusCodeMin: 500, StatusCodeMax: 599},
			want:      ErrorPageDispatch{ScopePath: "/", StatusCodeMin: 500, StatusCodeMax: 599},
		},
		{
			name:      "catch-all page",
			errorData: generator_dto.ManifestErrorPageEntry{ScopePath: "/", IsCatchAll: true},
			want:      ErrorPageDispatch{ScopePath: "/", IsCatchAll: true},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, *NewErrorPageDispatch(testCase.errorData))
		})
	}
}

func TestPageEntry_InitialiseLocalStore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		translations i18n_domain.Translations
		name         string
		wantStore    bool
	}{
		{name: "builds a store from local translations", translations: i18n_domain.Translations{"en": {"greeting": "Hello"}}, wantStore: true},
		{name: "leaves the store empty without translations", translations: nil, wantStore: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			entry := NewComponentPageEntry("project/dist/pages/home", "pages/home.pk")
			entry.LocalTranslations = testCase.translations
			entry.SetBaseDir("/project")
			entry.SetJSArtefactToPartialNameMap(map[string]string{"pk-js/card.js": "card"})
			entry.InitialiseLocalStore()
			assert.Equal(t, testCase.wantStore, entry.GetLocalStore() != nil)
			assert.Equal(t, "/project", entry.baseDir)
			assert.Equal(t, "card", entry.jsArtefactToPartialName["pk-js/card.js"])
		})
	}
}

func TestPageEntry_StaticFlags(t *testing.T) {
	t.Parallel()

	entry := NewComponentPageEntry("project/dist/pages/home", "pages/home.pk")
	entry.HasAuthPolicy = true
	entry.HasPreview = true
	assert.True(t, entry.GetHasAuthPolicy())
	assert.True(t, entry.GetHasPreview())
}
