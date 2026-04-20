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

package linguistics_bigrams_hebrew_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/wdk/linguistics/linguistics_bigrams_hebrew"
)

func TestFactory_ReturnsHebrewAnalyser(t *testing.T) {
	t.Parallel()

	analyser, factoryError := linguistics_bigrams_hebrew.Factory()

	require.NoError(t, factoryError)
	require.NotNil(t, analyser)
	assert.Equal(t, linguistics_bigrams_hebrew.Language, analyser.GetLanguage())
}

func TestBigramFrequencyRatio_RegistryLookup(t *testing.T) {
	t.Parallel()

	analyser := linguistics_domain.CreateBigramAnalyser(linguistics_bigrams_hebrew.Language)

	require.NotNil(t, analyser)
	assert.Equal(t, linguistics_bigrams_hebrew.Language, analyser.GetLanguage())
}

func TestBigramFrequencyRatio_ShortInputReturnsFalse(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	ratio, analysed := analyser.BigramFrequencyRatio("של")

	assert.Equal(t, 0.0, ratio)
	assert.False(t, analysed)
}

func TestBigramFrequencyRatio_CommonHebrewTextScoresLow(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	ratio, analysed := analyser.BigramFrequencyRatio("הילד של אמא")

	require.True(t, analysed)
	assert.Less(t, ratio, 0.5, "common Hebrew text should produce a low uncommon ratio")
}

func TestBigramFrequencyRatio_RandomLatinScoresHigh(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	ratio, analysed := analyser.BigramFrequencyRatio("qxzjvbnmplrt")

	require.True(t, analysed)
	assert.InDelta(t, 1.0, ratio, 0.0001, "non-Hebrew input should be entirely uncommon")
}

func TestBigramFrequencyRatio_StripsNikkudBeforeLookup(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	plainRatio, plainOK := analyser.BigramFrequencyRatio("הילד של אמא")
	pointedRatio, pointedOK := analyser.BigramFrequencyRatio("הַיֶּלֶד שֶׁל אִמָּא")

	require.True(t, plainOK)
	require.True(t, pointedOK)
	assert.InDelta(t, plainRatio, pointedRatio, 0.0001, "nikkud must be stripped before matching")
}

func TestBigramFrequencyRatio_FoldsFinalForms(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	finalForms := []string{
		"הילךאבג",
		"הילםאבג",
		"הילןאבג",
		"הילףאבג",
		"הילץאבג",
	}
	regularForms := []string{
		"הילכאבג",
		"הילמאבג",
		"הילנאבג",
		"הילפאבג",
		"הילצאבג",
	}

	for index, finalInput := range finalForms {
		regularInput := regularForms[index]
		finalRatio, finalOK := analyser.BigramFrequencyRatio(finalInput)
		regularRatio, regularOK := analyser.BigramFrequencyRatio(regularInput)

		require.True(t, finalOK)
		require.True(t, regularOK)
		assert.InDelta(t, regularRatio, finalRatio, 0.0001,
			"final form %q should fold to match %q", finalInput, regularInput)
	}
}

func TestBigramFrequencyRatio_IgnoresPunctuationAndDigits(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	plainRatio, plainOK := analyser.BigramFrequencyRatio("הילד של אמא")
	noisyRatio, noisyOK := analyser.BigramFrequencyRatio("הילד, של 1234 אמא!")

	require.True(t, plainOK)
	require.True(t, noisyOK)
	assert.InDelta(t, plainRatio, noisyRatio, 0.0001)
}

func TestBigramFrequencyRatio_TruncatesOversizedInput(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	oversized := strings.Repeat("א", 10000)

	ratio, analysed := analyser.BigramFrequencyRatio(oversized)

	require.True(t, analysed)
	assert.GreaterOrEqual(t, ratio, 0.0)
	assert.LessOrEqual(t, ratio, 1.0)
}

func TestBigramFrequencyRatio_AllNikkudReturnsFalse(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	ratio, analysed := analyser.BigramFrequencyRatio("\u05B0\u05B1\u05B2\u05B3")

	assert.Equal(t, 0.0, ratio)
	assert.False(t, analysed)
}

func TestBigramFrequencyRatio_EmptyReturnsFalse(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	ratio, analysed := analyser.BigramFrequencyRatio("")

	assert.Equal(t, 0.0, ratio)
	assert.False(t, analysed)
}

func TestBigramFrequencyRatio_ConcurrentUse(t *testing.T) {
	t.Parallel()

	analyser := &linguistics_bigrams_hebrew.BigramAnalyser{}

	var waitGroup sync.WaitGroup
	for range 64 {
		waitGroup.Go(func() {
			_, _ = analyser.BigramFrequencyRatio("הילד של אמא עם החבר")
		})
	}
	waitGroup.Wait()
}
