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

func TestSplitPluralForms_Empty(t *testing.T) {
	forms := SplitPluralForms("")
	assert.Nil(t, forms)
}

func TestSplitPluralForms_NoPipe(t *testing.T) {
	forms := SplitPluralForms("Hello, World!")
	assert.Len(t, forms, 1)
	assert.Equal(t, "Hello, World!", forms[0])
}

func TestSplitPluralForms_TwoForms(t *testing.T) {
	forms := SplitPluralForms("one item|{count} items")
	assert.Len(t, forms, 2)
	assert.Equal(t, "one item", forms[0])
	assert.Equal(t, "{count} items", forms[1])
}

func TestSplitPluralForms_ThreeForms(t *testing.T) {
	forms := SplitPluralForms("no items|one item|{count} items")
	assert.Len(t, forms, 3)
	assert.Equal(t, "no items", forms[0])
	assert.Equal(t, "one item", forms[1])
	assert.Equal(t, "{count} items", forms[2])
}

func TestSplitPluralForms_EscapedPipe(t *testing.T) {
	forms := SplitPluralForms("value || other|multiple")

	assert.Len(t, forms, 2)
	assert.Equal(t, "value | other", forms[0])
	assert.Equal(t, "multiple", forms[1])
}

func TestSplitPluralForms_EmptyForms(t *testing.T) {
	forms := SplitPluralForms("|middle|")
	assert.Len(t, forms, 3)
	assert.Equal(t, "", forms[0])
	assert.Equal(t, "middle", forms[1])
	assert.Equal(t, "", forms[2])
}

func TestHasPluralForms_True(t *testing.T) {
	assert.True(t, HasPluralForms("one|many"))
	assert.True(t, HasPluralForms("item|items"))
	assert.True(t, HasPluralForms("a|b|c"))
}

func TestHasPluralForms_False(t *testing.T) {
	assert.False(t, HasPluralForms("no pipe here"))
	assert.False(t, HasPluralForms("escaped || pipe"))
	assert.False(t, HasPluralForms(""))
}

func TestSelectPluralForm_Empty(t *testing.T) {
	result := SelectPluralForm(1, "en", nil)
	assert.Equal(t, "", result)
}

func TestSelectPluralForm_SingleForm(t *testing.T) {
	forms := []string{"items"}
	result := SelectPluralForm(5, "en", forms)
	assert.Equal(t, "items", result)
}

func TestSelectPluralForm_English(t *testing.T) {
	forms := []string{"one item", "{count} items"}

	tests := []struct {
		expected string
		count    int
	}{
		{count: 0, expected: "{count} items"},
		{count: 1, expected: "one item"},
		{count: 2, expected: "{count} items"},
		{count: 10, expected: "{count} items"},
		{count: 100, expected: "{count} items"},
	}

	for _, tc := range tests {
		result := SelectPluralForm(tc.count, "en", forms)
		assert.Equal(t, tc.expected, result, "count=%d", tc.count)
	}
}

func TestSelectPluralForm_EnglishGB(t *testing.T) {
	forms := []string{"one item", "{count} items"}

	result := SelectPluralForm(1, "en-GB", forms)
	assert.Equal(t, "one item", result)

	result = SelectPluralForm(5, "en-GB", forms)
	assert.Equal(t, "{count} items", result)
}

func TestSelectPluralForm_French(t *testing.T) {
	forms := []string{"un article", "{count} articles"}

	tests := []struct {
		expected string
		count    int
	}{
		{count: 0, expected: "un article"},
		{count: 1, expected: "un article"},
		{count: 2, expected: "{count} articles"},
	}

	for _, tc := range tests {
		result := SelectPluralForm(tc.count, "fr", forms)
		assert.Equal(t, tc.expected, result, "count=%d", tc.count)
	}
}

func TestSelectPluralForm_Russian_ThreeForms(t *testing.T) {
	forms := []string{"{n} яблоко", "{n} яблока", "{n} яблок"}

	tests := []struct {
		expected string
		count    int
	}{
		{count: 1, expected: "{n} яблоко"},
		{count: 2, expected: "{n} яблока"},
		{count: 5, expected: "{n} яблок"},
		{count: 11, expected: "{n} яблок"},
		{count: 21, expected: "{n} яблоко"},
		{count: 22, expected: "{n} яблока"},
		{count: 25, expected: "{n} яблок"},
	}

	for _, tc := range tests {
		result := SelectPluralForm(tc.count, "ru", forms)
		assert.Equal(t, tc.expected, result, "count=%d", tc.count)
	}
}

func TestSelectPluralForm_Polish(t *testing.T) {
	forms := []string{"{n} plik", "{n} pliki", "{n} plików"}

	tests := []struct {
		expected string
		count    int
	}{
		{count: 1, expected: "{n} plik"},
		{count: 2, expected: "{n} pliki"},
		{count: 5, expected: "{n} plików"},
		{count: 12, expected: "{n} plików"},
		{count: 22, expected: "{n} pliki"},
	}

	for _, tc := range tests {
		result := SelectPluralForm(tc.count, "pl", forms)
		assert.Equal(t, tc.expected, result, "count=%d", tc.count)
	}
}

func TestSelectPluralForm_Chinese(t *testing.T) {
	forms := []string{"{count}个项目"}
	result := SelectPluralForm(5, "zh", forms)
	assert.Equal(t, "{count}个项目", result)
}

func TestSelectPluralForm_Arabic_SixForms(t *testing.T) {
	forms := []string{"صفر", "واحد", "اثنان", "قليل", "كثير", "آخر"}

	tests := []struct {
		expected string
		count    int
	}{
		{count: 0, expected: "صفر"},
		{count: 1, expected: "واحد"},
		{count: 2, expected: "اثنان"},
		{count: 5, expected: "قليل"},
		{count: 11, expected: "كثير"},
		{count: 100, expected: "آخر"},
	}

	for _, tc := range tests {
		result := SelectPluralForm(tc.count, "ar", forms)
		assert.Equal(t, tc.expected, result, "count=%d", tc.count)
	}
}

func TestSelectPluralForm_UnknownLocale(t *testing.T) {
	forms := []string{"singular", "plural"}

	result := SelectPluralForm(1, "xx-YY", forms)
	assert.Equal(t, "singular", result)

	result = SelectPluralForm(5, "xx-YY", forms)
	assert.Equal(t, "plural", result)
}

func TestGetPluralCategory_NegativeNumbers(t *testing.T) {
	category := getPluralCategory(-1, "en")
	assert.Equal(t, pluralOther, category)

	category = getPluralCategory(-5, "en")
	assert.Equal(t, pluralOther, category)
}

func BenchmarkSplitPluralForms(b *testing.B) {
	template := "one item|{count} items"
	b.ResetTimer()

	for b.Loop() {
		_ = SplitPluralForms(template)
	}
}

func BenchmarkHasPluralForms(b *testing.B) {
	template := "one item|{count} items"
	b.ResetTimer()

	for b.Loop() {
		_ = HasPluralForms(template)
	}
}

func BenchmarkSelectPluralForm(b *testing.B) {
	forms := []string{"one item", "{count} items"}
	b.ResetTimer()

	i := 0
	for b.Loop() {
		_ = SelectPluralForm(i%100, "en", forms)
		i++
	}
}
