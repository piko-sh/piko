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
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

const (
	// propertyRandomSamples controls how many random Hebrew-looking
	// strings the property tests synthesise per invocation.
	propertyRandomSamples = 500

	// concurrentStemGoroutines is the number of goroutines used by
	// the thread-safety test.
	concurrentStemGoroutines = 50

	// concurrentStemIterations is the number of stemming calls each
	// thread-safety goroutine performs.
	concurrentStemIterations = 200

	// pathologicalRepeatCount is the length of the stress-test
	// repeated-letter input.
	pathologicalRepeatCount = 1000
)

func TestProperty_StemIsIdempotent(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	seeds := append(idempotenceCuratedSeeds(), idempotenceRandomSeeds()...)
	for _, seed := range seeds {
		first := stemmer.Stem(seed)
		second := stemmer.Stem(first)
		assert.Equal(t, first, second, "stemmer lost idempotence on %q", seed)
	}
}

func TestProperty_FinalFormsFoldedInOutput(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	finalForms := []rune{'\u05DA', '\u05DD', '\u05DF', '\u05E3', '\u05E5'}
	seeds := append(idempotenceCuratedSeeds(), idempotenceRandomSeeds()...)
	for _, seed := range seeds {
		result := stemmer.Stem(seed)
		for _, finalRune := range finalForms {
			assert.NotContains(t, result, string(finalRune), "final form leaked for input %q -> %q", seed, result)
		}
	}
}

func TestProperty_OutputRuneLengthNeverExceedsInput(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	for _, seed := range idempotenceRandomSeeds() {
		cleaned := normaliseFinalForms(stripNikkud(seed))
		result := stemmer.Stem(seed)
		assert.LessOrEqual(t, utf8.RuneCountInString(result), utf8.RuneCountInString(cleaned),
			"stem grew input: %q -> %q", seed, result)
	}
}

func TestProperty_NonHebrewInputPreserved(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	inputs := []string{
		"hello",
		"שלום world",
		"مرحبا",
		"こんにちは",
		"🙂",
		"",
		" ",
		"123456",
	}
	for _, input := range inputs {
		result := stemmer.Stem(input)
		if input == "" {
			assert.Empty(t, result)
			continue
		}
		assert.NotPanics(t, func() { _ = stemmer.Stem(input) })
		assert.LessOrEqual(t, utf8.RuneCountInString(result), utf8.RuneCountInString(input))
	}
}

func TestProperty_PathologicalInputsDoNotPanic(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	inputs := []string{
		strings.Repeat("ה", pathologicalRepeatCount),
		strings.Repeat("\u05B7", pathologicalRepeatCount),
		strings.Repeat("ך", pathologicalRepeatCount),
		strings.Repeat("שלום ", pathologicalRepeatCount),
	}
	for _, input := range inputs {
		assert.NotPanics(t, func() { _ = stemmer.Stem(input) }, "pathological input panicked")
	}
}

func TestProperty_StemmerConcurrent(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	seeds := idempotenceCuratedSeeds()
	if len(seeds) == 0 {
		t.Fatal("curated seeds must not be empty")
	}

	var waitGroup sync.WaitGroup
	for routine := range concurrentStemGoroutines {
		waitGroup.Go(func() {
			for iteration := range concurrentStemIterations {
				seed := seeds[(routine+iteration)%len(seeds)]
				_ = stemmer.Stem(seed)
			}
		})
	}
	waitGroup.Wait()
}

func TestProperty_InflectionEquivalence(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	pairs := map[string]string{
		"אנשים": "איש",
		"נשים":  "אישה",
		"ראשים": "ראש",
		"ילדים": "ילד",
		"בנקים": "בנק",
		"הולכת": "הלכ",
		"אומרת": "אמר",
		"אוכלת": "אכל",
		"לומדת": "למד",
		"עושה":  "עשה",
		"רואה":  "ראה",
		"אוהבת": "אהב",
		"כתבתי": "כתב",
		"למדתי": "למד",
	}
	for surface, lemma := range pairs {
		stemmed := stemmer.Stem(surface)
		assert.True(t,
			stemmed == lemma || strings.Contains(lemma, stemmed) || strings.Contains(stemmed, lemma),
			"stem(%q)=%q incompatible with lemma %q", surface, stemmed, lemma,
		)
	}
}

func FuzzStem(f *testing.F) {
	stemmer := &Stemmer{}
	for _, seed := range idempotenceCuratedSeeds() {
		f.Add(seed)
	}
	f.Add("")
	f.Add("abc")
	f.Add("\x80\x81\x82")
	f.Add("שלום")
	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("stem panicked on %q: %v", input, recovered)
			}
		}()
		result := stemmer.Stem(input)
		if !utf8.ValidString(result) {
			t.Fatalf("stem produced invalid UTF-8 for input %q", input)
		}
	})
}

func idempotenceCuratedSeeds() []string {
	return []string{
		"בית", "בנק", "ספר", "ילד", "ילדים",
		"חברות", "הבנק", "בבנק", "לבנק", "מבנק",
		"שהילדים", "כשהגיע", "והחברות", "בנקיהם",
		"כתבתי", "למדתי", "הולכת", "אוהבת",
		"עושה", "רואה", "אנשים", "נשים",
		"ישראלית", "גדולה", "יפה", "יפים",
		"",
	}
}

func idempotenceRandomSeeds() []string {
	source := rand.NewChaCha8([32]byte{
		0xA1, 0xB2, 0xC3, 0xD4, 0xE5, 0xF6, 0x17, 0x28,
		0x39, 0x4A, 0x5B, 0x6C, 0x7D, 0x8E, 0x9F, 0x0A,
		0x5A, 0x17, 0x12, 0xAB, 0xDE, 0xCA, 0xF0, 0x01,
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
	})
	random := rand.New(source)
	seeds := make([]string, 0, propertyRandomSamples)
	for range propertyRandomSamples {
		length := 2 + random.IntN(10)
		builder := make([]rune, 0, length)
		for range length {
			builder = append(builder, rune(0x05D0+random.IntN(27)))
		}
		seeds = append(seeds, string(builder))
	}
	return seeds
}
