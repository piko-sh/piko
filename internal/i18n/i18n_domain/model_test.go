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

package i18n_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslations_EmptyMap(t *testing.T) {
	translations := make(Translations)

	assert.Empty(t, translations)
}

func TestTranslations_SingleLocale(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"greeting": "Hello",
			"farewell": "Goodbye",
		},
	}

	assert.Len(t, translations, 1)
	assert.Len(t, translations["en-GB"], 2)
	assert.Equal(t, "Hello", translations["en-GB"]["greeting"])
	assert.Equal(t, "Goodbye", translations["en-GB"]["farewell"])
}

func TestTranslations_MultipleLocales(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"greeting": "Hello",
		},
		"fr-FR": {
			"greeting": "Bonjour",
		},
		"de-DE": {
			"greeting": "Hallo",
		},
	}

	assert.Len(t, translations, 3)
	assert.Equal(t, "Hello", translations["en-GB"]["greeting"])
	assert.Equal(t, "Bonjour", translations["fr-FR"]["greeting"])
	assert.Equal(t, "Hallo", translations["de-DE"]["greeting"])
}

func TestTranslations_LocaleAccess(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"key1": "value1",
		},
	}

	locale, exists := translations["en-GB"]
	assert.True(t, exists)
	assert.NotNil(t, locale)

	locale, exists = translations["es-ES"]
	assert.False(t, exists)
	assert.Nil(t, locale)
}

func TestTranslations_KeyAccess(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"existing.key": "value",
		},
	}

	value, exists := translations["en-GB"]["existing.key"]
	assert.True(t, exists)
	assert.Equal(t, "value", value)

	value, exists = translations["en-GB"]["missing.key"]
	assert.False(t, exists)
	assert.Empty(t, value)
}

func TestTranslations_DotNotationKeys(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"user.profile.name":  "Name",
			"user.profile.email": "Email",
			"common.buttons.ok":  "OK",
		},
	}

	assert.Len(t, translations["en-GB"], 3)
	assert.Equal(t, "Name", translations["en-GB"]["user.profile.name"])
	assert.Equal(t, "Email", translations["en-GB"]["user.profile.email"])
	assert.Equal(t, "OK", translations["en-GB"]["common.buttons.ok"])
}

func TestTranslations_EmptyValues(t *testing.T) {
	translations := Translations{
		"en-GB": {
			"empty":    "",
			"nonempty": "value",
		},
	}

	assert.Equal(t, "", translations["en-GB"]["empty"])
	assert.Equal(t, "value", translations["en-GB"]["nonempty"])
}

func TestTranslations_EmptyLocale(t *testing.T) {
	translations := Translations{
		"en-GB": {},
	}

	assert.Len(t, translations, 1)
	assert.Empty(t, translations["en-GB"])
}
