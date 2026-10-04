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
	"github.com/stretchr/testify/require"
)

func TestStore_NewStore(t *testing.T) {
	store := NewStore("en-GB")
	assert.NotNil(t, store)
	assert.Empty(t, store.Locales())
}

func TestStore_AddTranslations(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	})

	assert.True(t, store.HasLocale("en-GB"))
	assert.Equal(t, []string{"en-GB"}, store.Locales())
}

func TestStore_Get_Found(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, ${name}!",
	})

	entry, found := store.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello, ${name}!", entry.Template)
}

func TestStore_Get_NotFound(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello",
	})

	entry, found := store.Get("en-GB", "missing")
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestStore_Get_LocaleFallback(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	})
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello, mate!",
	})

	entry, found := store.Get("en-GB", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello, mate!", entry.Template)

	entry, found = store.Get("en-GB", "farewell")
	require.True(t, found)
	assert.Equal(t, "Goodbye", entry.Template)
}

func TestStore_Get_DefaultLocaleFallback(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})

	entry, found := store.Get("fr", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello", entry.Template)
}

func TestStore_Get_MissingLocale(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})

	entry, found := store.Get("fr", "greeting")
	require.True(t, found)
	assert.Equal(t, "Hello", entry.Template)
}

func TestStore_AddLocale_PreParsed(t *testing.T) {
	store := NewStore("en")
	parts, _ := ParseTemplate("Hello, ${name}!")
	store.AddLocale("en", map[string]*Entry{
		"greeting": {
			Template: "Hello, ${name}!",
			Parts:    parts,
		},
	})

	entry, found := store.Get("en", "greeting")
	require.True(t, found)
	assert.Len(t, entry.Parts, 3)
}

func TestStore_AddTranslations_ParsesPlurals(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"items": "one item|${count} items",
	})

	entry, found := store.Get("en", "items")
	require.True(t, found)
	assert.True(t, entry.HasPlurals)
	assert.Len(t, entry.PluralForms, 2)
	assert.Equal(t, "one item", entry.PluralForms[0])
	assert.Equal(t, "${count} items", entry.PluralForms[1])
}

func TestStore_AddTranslations_NoPluralsForms(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}!",
	})

	entry, found := store.Get("en", "greeting")
	require.True(t, found)
	assert.False(t, entry.HasPlurals)
	assert.Nil(t, entry.PluralForms)
}

func TestStore_SetDefaultLocale(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})
	store.AddTranslations("de", map[string]string{
		"greeting": "Hallo",
	})

	entry, _ := store.Get("fr", "greeting")
	assert.Equal(t, "Hello", entry.Template)

	store.SetDefaultLocale("de")

	entry, _ = store.Get("fr", "greeting")
	assert.Equal(t, "Hallo", entry.Template)
}

func TestStore_HasLocale(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en-GB", map[string]string{
		"greeting": "Hello",
	})

	assert.True(t, store.HasLocale("en-GB"))
	assert.False(t, store.HasLocale("en"))
	assert.False(t, store.HasLocale("fr"))
}

func TestStore_Locales(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en-GB", map[string]string{"a": "1"})
	store.AddTranslations("fr-FR", map[string]string{"a": "2"})
	store.AddTranslations("de-DE", map[string]string{"a": "3"})

	locales := store.Locales()
	assert.Len(t, locales, 3)
	assert.Contains(t, locales, "en-GB")
	assert.Contains(t, locales, "fr-FR")
	assert.Contains(t, locales, "de-DE")
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})

	done := make(chan bool)
	for range 10 {
		go func() {
			for range 100 {
				_, _ = store.Get("en", "greeting")
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestBuildFallbackChain(t *testing.T) {
	tests := []struct {
		locale        string
		defaultLocale string
		expected      []string
	}{
		{
			locale:        "en-GB",
			defaultLocale: "en",
			expected:      []string{"en"},
		},
		{
			locale:        "en",
			defaultLocale: "en",
			expected:      nil,
		},
		{
			locale:        "fr-FR",
			defaultLocale: "en-US",
			expected:      []string{"fr", "en"},
		},
		{
			locale:        "zh-Hans-CN",
			defaultLocale: "en",
			expected:      []string{"zh"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.locale, func(t *testing.T) {
			chain := buildFallbackChain(tc.locale, tc.defaultLocale)
			assert.Equal(t, tc.expected, chain)
		})
	}
}

func BenchmarkStore_Get(b *testing.B) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello, ${name}!",
	})
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Get("en", "greeting")
	}
}

func BenchmarkStore_GetWithFallback(b *testing.B) {
	store := NewStore("en")
	store.AddTranslations("en", map[string]string{
		"greeting": "Hello",
	})
	store.AddTranslations("en-GB", map[string]string{})
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Get("en-GB", "greeting")
	}
}

func TestStore_AddTranslations_ReportsTemplateProblems(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		translations    map[string]string
		wantProblems    []string
		wantKey         string
		wantParts       []string
		wantPluralParts [][]string
	}{
		{
			name:         "templates that parse report no problems",
			translations: map[string]string{"greeting": "Hello ${name}", "plural": "one|${count} many"},
			wantProblems: nil,
			wantKey:      "greeting",
			wantParts:    []string{"Hello ", "${name}"},
		},
		{
			name:         "a broken template renders as its literal text",
			translations: map[string]string{"broken": "Hello ${name"},
			wantProblems: []string{"en-GB:broken: Unterminated expression: expected '}'"},
			wantKey:      "broken",
			wantParts:    []string{"Hello ${name"},
		},
		{
			name:            "a broken first plural form keeps its text as the entry parts",
			translations:    map[string]string{"items": "one ${broken|${count} items"},
			wantProblems:    []string{"en-GB:items[0]: Unterminated expression: expected '}'"},
			wantKey:         "items",
			wantParts:       []string{"one ${broken"},
			wantPluralParts: [][]string{{"one ${broken"}, {"${count}", " items"}},
		},
		{
			name:            "a broken later plural form is reported with its index",
			translations:    map[string]string{"items": "one item|${count items"},
			wantProblems:    []string{"en-GB:items[1]: Unterminated expression: expected '}'"},
			wantKey:         "items",
			wantParts:       []string{"one item"},
			wantPluralParts: [][]string{{"one item"}, {"${count items"}},
		},
		{
			name:         "problems are ordered by key",
			translations: map[string]string{"b": "${", "a": "${"},
			wantProblems: []string{
				"en-GB:a: Unterminated expression: expected '}'",
				"en-GB:b: Unterminated expression: expected '}'",
			},
			wantKey:   "a",
			wantParts: []string{"${"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewStore("en-GB")

			problems := store.AddTranslations("en-GB", tc.translations)

			var described []string
			for _, problem := range problems {
				described = append(described, problem.String())
			}
			assert.Equal(t, tc.wantProblems, described)

			entry, found := store.Get("en-GB", tc.wantKey)
			require.True(t, found)
			assert.Equal(t, tc.wantParts, describeParts(entry.Parts))
			if tc.wantPluralParts != nil {
				require.Len(t, entry.PluralFormsParts, len(tc.wantPluralParts))
				for index, want := range tc.wantPluralParts {
					assert.Equal(t, want, describeParts(entry.PluralFormsParts[index]), "plural form %d", index)
				}
			}
		})
	}
}

func TestStore_ResolveMessage_UnparsedBrokenTemplateRendersLiterally(t *testing.T) {
	t.Parallel()

	store := NewStore("en-GB")
	store.AddLocale("en-GB", map[string]*Entry{
		"broken": {Template: "Hi ${name", Parts: nil, PluralForms: nil, PluralFormsParts: nil, HasPlurals: false},
	})

	got, found := store.ResolveMessage("broken", "en-GB", map[string]any{"name": "Ana"}, 0)

	require.True(t, found)
	assert.Equal(t, "Hi ${name", got)
}

func describeParts(parts []TemplatePart) []string {
	described := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part.Kind {
		case PartExpression:
			described = append(described, "${"+part.ExprSource+"}")
		case PartLinkedMessage:
			described = append(described, "@"+part.LinkedKey)
		default:
			described = append(described, part.Literal)
		}
	}
	return described
}

func TestStore_AddAllTranslations_OrdersProblemsByLocale(t *testing.T) {
	t.Parallel()

	store := NewStore("en-GB")

	problems := store.AddAllTranslations(Translations{
		"fr-FR": {"b": "${", "ok": "Bonjour"},
		"en-GB": {"a": "${", "ok": "Hello"},
	})

	var described []string
	for _, problem := range problems {
		described = append(described, problem.Locale+":"+problem.Key)
	}
	assert.Equal(t, []string{"en-GB:a", "fr-FR:b"}, described)
	assert.True(t, store.HasLocale("en-GB"))
	assert.True(t, store.HasLocale("fr-FR"))
}
