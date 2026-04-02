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
