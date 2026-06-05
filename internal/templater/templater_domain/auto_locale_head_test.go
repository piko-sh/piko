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

package templater_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestApplyAutoLocaleHead_FillsEmptyFields(t *testing.T) {
	t.Parallel()

	meta := &templater_dto.InternalMetadata{}
	auto := &LocaleSEOHead{
		Language:       "fr",
		CanonicalURL:   "https://example.com/what-we-do/",
		AlternateLinks: []map[string]string{{"hreflang": "fr", "href": "https://example.com/fr/what-we-do/"}},
	}

	applyAutoLocaleHead(meta, auto)

	assert.Equal(t, "fr", meta.Language)
	assert.Equal(t, "https://example.com/what-we-do/", meta.CanonicalURL)
	assert.Len(t, meta.AlternateLinks, 1)
}

func TestApplyAutoLocaleHead_PreservesExplicitValues(t *testing.T) {
	t.Parallel()

	meta := &templater_dto.InternalMetadata{}
	meta.Language = "en"
	meta.CanonicalURL = "https://custom.example/page"
	meta.AlternateLinks = []map[string]string{{"hreflang": "en", "href": "https://custom.example/page"}}

	auto := &LocaleSEOHead{
		Language:       "fr",
		CanonicalURL:   "https://example.com/what-we-do/",
		AlternateLinks: []map[string]string{{"hreflang": "fr", "href": "https://example.com/fr/what-we-do/"}},
	}

	applyAutoLocaleHead(meta, auto)

	assert.Equal(t, "en", meta.Language, "explicit language must win")
	assert.Equal(t, "https://custom.example/page", meta.CanonicalURL, "explicit canonical must win")
	assert.Equal(t, "https://custom.example/page", meta.AlternateLinks[0]["href"], "explicit alternates must win")
}

func TestApplyAutoLocaleHead_NilIsNoOp(t *testing.T) {
	t.Parallel()

	meta := &templater_dto.InternalMetadata{}
	applyAutoLocaleHead(meta, nil)

	assert.Empty(t, meta.Language)
	assert.Empty(t, meta.CanonicalURL)
	assert.Empty(t, meta.AlternateLinks)
}
