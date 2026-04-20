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

package linguistics_stemmer_hebrew

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupIrregular_KnownBrokenPlurals(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"אנשימ":  "איש",
		"נשימ":   "אישה",
		"בנות":   "בת",
		"אחיות":  "אחות",
		"ראשימ":  "ראש",
		"ערימ":   "עיר",
		"שמות":   "שמ",
		"בתימ":   "בית",
		"אימהות": "אמ",
	}
	for surface, expected := range testCases {
		base, ok := lookupIrregular(surface)
		assert.True(t, ok, "expected %q to be in irregular map", surface)
		assert.Equal(t, expected, base, "surface %q", surface)
	}
}

func TestLookupIrregular_WeakRootVerbs(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"הולכת":  "הלכ",
		"אומרת":  "אמר",
		"אוכלת":  "אכל",
		"לומדת":  "למד",
		"כותבתי": "",
		"עושה":   "עשה",
		"רואה":   "ראה",
		"אוהבת":  "אהב",
	}
	for surface, expected := range testCases {
		base, ok := lookupIrregular(surface)
		if expected == "" {
			assert.False(t, ok, "expected %q to NOT be in irregular map", surface)
			continue
		}
		assert.True(t, ok, "expected %q to be in irregular map", surface)
		assert.Equal(t, expected, base, "surface %q", surface)
	}
}

func TestLookupIrregular_NotPresentFallsThrough(t *testing.T) {
	t.Parallel()

	_, ok := lookupIrregular("בית")
	assert.False(t, ok, "basic regular noun should not be in map")

	_, ok = lookupIrregular("בנק")
	assert.False(t, ok)

	_, ok = lookupIrregular("")
	assert.False(t, ok, "empty string should not be in map")
}

func TestStemIntegratesIrregulars(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"אנשים":  "איש",
		"נשים":   "אישה",
		"אחיות":  "אחות",
		"ערים":   "עיר",
		"אוהבת":  "אהב",
		"הולכת":  "הלכ",
		"עושה":   "עשה",
		"רואה":   "ראה",
		"קוראות": "קרא",
	}
	for surface, expected := range testCases {
		assert.Equal(t, expected, stem(surface), "stem(%q)", surface)
	}
}
