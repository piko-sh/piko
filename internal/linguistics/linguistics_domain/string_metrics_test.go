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

package linguistics_domain

import (
	"testing"
)

func TestWagnerFischer_KnownDistances(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{name: "identical", a: "test", b: "test", expected: 0},
		{name: "empty strings", a: "", b: "", expected: 0},
		{name: "one empty", a: "test", b: "", expected: 4},
		{name: "other empty", a: "", b: "test", expected: 4},
		{name: "single substitution", a: "cat", b: "bat", expected: 2},
		{name: "single insertion", a: "cat", b: "cats", expected: 1},
		{name: "single deletion", a: "cats", b: "cat", expected: 1},
		{name: "kitten to sitting", a: "kitten", b: "sitting", expected: 5},
		{name: "saturday to sunday", a: "saturday", b: "sunday", expected: 4},
		{name: "configuration typo", a: "configuration", b: "configurtion", expected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			distance := WagnerFischer(tc.a, tc.b, 1, 1, 2)
			if distance != tc.expected {
				t.Errorf("WagnerFischer(%q, %q) = %d, want %d", tc.a, tc.b, distance, tc.expected)
			}
		})
	}
}

func TestWagnerFischer_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"hello", "world"},
		{"cat", "dog"},
		{"test", "testing"},
	}

	for _, pair := range pairs {
		distAB := WagnerFischer(pair[0], pair[1], 1, 1, 2)
		distBA := WagnerFischer(pair[1], pair[0], 1, 1, 2)

		if distAB != distBA {
			t.Errorf("Distance should be symmetric: dist(%q,%q)=%d but dist(%q,%q)=%d",
				pair[0], pair[1], distAB, pair[1], pair[0], distBA)
		}
	}
}

func TestJaro_KnownSimilarities(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		minScore float64
	}{
		{name: "identical", a: "MARTHA", b: "MARTHA", minScore: 1.0},
		{name: "transposition", a: "MARTHA", b: "MARHTA", minScore: 0.94},
		{name: "different", a: "DIXON", b: "DICKSONX", minScore: 0.76},
		{name: "empty both", a: "", b: "", minScore: 1.0},
		{name: "empty one", a: "test", b: "", minScore: 0.0},
		{name: "very different", a: "ABC", b: "XYZ", minScore: 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := Jaro(tc.a, tc.b)
			if score < tc.minScore {
				t.Errorf("Jaro(%q, %q) = %.3f, want >= %.3f", tc.a, tc.b, score, tc.minScore)
			}
			t.Logf("Jaro(%q, %q) = %.3f", tc.a, tc.b, score)
		})
	}
}

func TestJaro_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"MARTHA", "MARHTA"},
		{"DIXON", "DICKSONX"},
		{"hello", "world"},
	}

	for _, pair := range pairs {
		scoreAB := Jaro(pair[0], pair[1])
		scoreBA := Jaro(pair[1], pair[0])

		if abs(scoreAB-scoreBA) > 0.0001 {
			t.Errorf("Jaro should be symmetric: Jaro(%q,%q)=%.4f but Jaro(%q,%q)=%.4f",
				pair[0], pair[1], scoreAB, pair[1], pair[0], scoreBA)
		}
	}
}

func TestJaroWinkler_KnownSimilarities(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		minScore float64
	}{
		{name: "name typo", a: "SHACKLEFORD", b: "SHACKELFORD", minScore: 0.98},
		{name: "first char diff", a: "DUNNINGHAM", b: "CUNNIGHAM", minScore: 0.89},
		{name: "middle typo", a: "NICHLESON", b: "NICHULSON", minScore: 0.95},
		{name: "suffix diff", a: "JONES", b: "JOHNSON", minScore: 0.83},
		{name: "char substitution", a: "MASSEY", b: "MASSIE", minScore: 0.93},
		{name: "configuration typo", a: "configuration", b: "configurtion", minScore: 0.98},
		{name: "transposition", a: "ocnfiguration", b: "configuration", minScore: 0.97},
	}

	const boostThreshold = 0.7
	const prefixSize = 4

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := JaroWinkler(tc.a, tc.b, boostThreshold, prefixSize)
			if score < tc.minScore {
				t.Errorf("JaroWinkler(%q, %q) = %.3f, want >= %.3f", tc.a, tc.b, score, tc.minScore)
			}
			t.Logf("JaroWinkler(%q, %q) = %.3f", tc.a, tc.b, score)
		})
	}
}

func TestJaroWinkler_PrefixBoost(t *testing.T) {
	const boostThreshold = 0.7
	const prefixSize = 4

	baseJaro := Jaro("DIXON", "DICKSON")

	withBoost := JaroWinkler("DIXON", "DICKSON", boostThreshold, prefixSize)

	if withBoost < baseJaro {
		t.Errorf("Jaro-Winkler (%.3f) should be >= base Jaro (%.3f)", withBoost, baseJaro)
	}

	t.Logf("Base Jaro: %.3f, With Winkler boost: %.3f (prefix 'DI' matched)", baseJaro, withBoost)
}

func TestJaroWinkler_EmptyStrings(t *testing.T) {
	const boostThreshold = 0.7
	const prefixSize = 4

	score := JaroWinkler("", "", boostThreshold, prefixSize)
	if score != 1.0 {
		t.Errorf("JaroWinkler(\"\", \"\") = %.3f, want 1.0", score)
	}

	score = JaroWinkler("test", "", boostThreshold, prefixSize)
	if score != 0.0 {
		t.Errorf("JaroWinkler(\"test\", \"\") = %.3f, want 0.0", score)
	}
}

func TestStringMetrics_TypoScenarios(t *testing.T) {
	typos := []struct {
		correct string
		typo    string
		reason  string
	}{
		{correct: "documentation", typo: "documnetation", reason: "transposed mn"},
		{correct: "deployment", typo: "deploymen", reason: "missing t"},
		{correct: "configuration", typo: "configurtion", reason: "missing a"},
		{correct: "development", typo: "developmnet", reason: "transposed mn"},
		{correct: "debugging", typo: "debuging", reason: "missing g"},
	}

	for _, scenario := range typos {
		t.Run(scenario.typo, func(t *testing.T) {

			similarity := JaroWinkler(scenario.correct, scenario.typo, 0.7, 4)

			if similarity < 0.85 {
				t.Errorf("Typo %q → %q (%s): similarity %.3f < 0.85",
					scenario.typo, scenario.correct, scenario.reason, similarity)
			}

			t.Logf("%q → %q (%.3f) - %s", scenario.typo, scenario.correct, similarity, scenario.reason)
		})
	}
}

func BenchmarkWagnerFischer(b *testing.B) {
	const a = "configuration"
	const c = "configurtion"

	b.ResetTimer()
	for b.Loop() {
		WagnerFischer(a, c, 1, 1, 2)
	}
}

func BenchmarkJaro(b *testing.B) {
	const a = "MARTHA"
	const c = "MARHTA"

	b.ResetTimer()
	for b.Loop() {
		Jaro(a, c)
	}
}

func BenchmarkJaroWinkler(b *testing.B) {
	const a = "configuration"
	const c = "configurtion"

	b.ResetTimer()
	for b.Loop() {
		JaroWinkler(a, c, 0.7, 4)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
