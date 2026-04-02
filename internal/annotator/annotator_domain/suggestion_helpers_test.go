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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	testCases := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{

		{name: "Identical strings", s1: "hello", s2: "hello", expected: 0},
		{name: "Single substitution", s1: "kitten", s2: "sitten", expected: 1},
		{name: "Single insertion", s1: "sit", s2: "sits", expected: 1},
		{name: "Single deletion", s1: "saturday", s2: "saurday", expected: 1},
		{name: "Multiple edits", s1: "saturday", s2: "sunday", expected: 3},
		{name: "Transposition (costs 2)", s1: "martha", s2: "marhta", expected: 2},

		{name: "One string is empty", s1: "hello", s2: "", expected: 5},
		{name: "Other string is empty", s1: "", s2: "world", expected: 5},
		{name: "Both strings are empty", s1: "", s2: "", expected: 0},
		{name: "One is a prefix of the other", s1: "test", s2: "testing", expected: 3},

		{name: "Different case", s1: "Hello", s2: "hello", expected: 1},
		{name: "All different case", s1: "WORLD", s2: "world", expected: 5},

		{name: "Identical unicode", s1: "你好", s2: "你好", expected: 0},
		{name: "Single unicode insertion", s1: "你好世界", s2: "你好世界a", expected: 1},
		{name: "Mixed ascii and unicode", s1: "hello你好", s2: "hellö你好", expected: 1},
		{name: "Complex unicode edits", s1: " résumé", s2: "resume", expected: 3},
		{name: "Emoji", s1: "😄", s2: "😃", expected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := levenshteinDistance(tc.s1, tc.s2)
			assert.Equal(t, tc.expected, actual, "Distance between '%s' and '%s' should be %d", tc.s1, tc.s2, tc.expected)
		})
	}
}

func TestFindClosestMatch(t *testing.T) {
	candidates := []string{
		"Title",
		"Content",
		"AuthorName",
		"PublicationDate",
		"IsEnabled",
		"category",
		"ContentType",
		"author",
		"TitleBar",
	}

	testCases := []struct {
		name        string
		input       string
		expected    string
		description string
		candidates  []string
	}{

		{
			name:        "Exact match",
			input:       "Title",
			candidates:  candidates,
			expected:    "Title",
			description: "Should return the exact match if it exists.",
		},
		{
			name:        "Case-insensitive exact match",
			input:       "title",
			candidates:  candidates,
			expected:    "Title",
			description: "Should find the correct candidate even with different casing.",
		},
		{
			name:        "Simple typo (substitution)",
			input:       "Titel",
			candidates:  candidates,
			expected:    "Title",
			description: "Should correct a single character substitution.",
		},
		{
			name:        "Simple typo (insertion)",
			input:       "Titles",
			candidates:  candidates,
			expected:    "Title",
			description: "Should correct a single character insertion.",
		},
		{
			name:        "Simple typo (deletion)",
			input:       "Ttle",
			candidates:  candidates,
			expected:    "Title",
			description: "Should correct a single character deletion.",
		},
		{
			name:        "Transposition typo",
			input:       "AuhtorName",
			candidates:  candidates,
			expected:    "AuthorName",
			description: "Should correct a two-character transposition.",
		},
		{
			name:        "Finds best of multiple mediocre matches",
			input:       "catego",
			candidates:  candidates,
			expected:    "category",
			description: "Should pick the closest match even if it's not a perfect typo.",
		},

		{
			name:        "Empty input",
			input:       "",
			candidates:  candidates,
			expected:    "",
			description: "Should return empty string for empty input.",
		},
		{
			name:        "Empty candidates list",
			input:       "Title",
			candidates:  []string{},
			expected:    "",
			description: "Should return empty string if there are no candidates.",
		},
		{
			name:        "Nil candidates list",
			input:       "Title",
			candidates:  nil,
			expected:    "",
			description: "Should handle a nil candidate slice gracefully.",
		},

		{
			name:        "No close match",
			input:       "username",
			candidates:  candidates,
			expected:    "",
			description: "Should return an empty string if no candidate is within the similarity threshold.",
		},
		{
			name:        "Very short input with no close match",
			input:       "id",
			candidates:  candidates,
			expected:    "",
			description: "Should not suggest anything for a very short, dissimilar word.",
		},
		{
			name:        "Threshold test: just within the limit",
			input:       "Contentious",
			candidates:  candidates,
			expected:    "Content",
			description: "Should suggest a match that is just within the threshold.",
		},
		{
			name:        "Threshold test: too far",
			input:       "Authorization",
			candidates:  candidates,
			expected:    "",
			description: "Should not suggest a match that is too different.",
		},

		{
			name:        "Tie-breaking: should prefer first candidate in case of equal distance",
			input:       "Auth",
			candidates:  []string{"Author", "AuthService"},
			expected:    "Author",
			description: "Should be deterministic and pick the first best match found.",
		},
		{
			name:        "Unicode: Simple typo",
			input:       "résumé",
			candidates:  []string{"resume"},
			expected:    "resume",
			description: "Should handle typos involving non-ASCII characters.",
		},
		{
			name:        "Unicode: No close match",
			input:       "你好",
			candidates:  candidates,
			expected:    "",
			description: "Should correctly calculate large distances for completely different scripts.",
		},
		{
			name:        "Punctuation: Input with hyphen",
			input:       "is-enable",
			candidates:  candidates,
			expected:    "IsEnabled",
			description: "Should handle common separators like hyphens.",
		},
		{
			name:        "Punctuation: Input with underscore",
			input:       "author_name",
			candidates:  candidates,
			expected:    "AuthorName",
			description: "Should handle common separators like underscores.",
		},
		{
			name:        "Numbers: Simple typo",
			input:       "User1",
			candidates:  []string{"User2"},
			expected:    "User2",
			description: "Should correctly handle typos involving numbers.",
		},
		{
			name:        "Threshold with very short word (3 letters)",
			input:       "Cat",
			candidates:  []string{"category", "car"},
			expected:    "car",
			description: "Should have a threshold of 2 for 3-letter words.",
		},
		{
			name:        "Threshold with very short word (2 letters)",
			input:       "Is",
			candidates:  candidates,
			expected:    "",
			description: "Should use minimum threshold of 2 for 2-letter words, preventing loose matches.",
		},
		{
			name:        "Multiple candidates, one is a substring",
			input:       "Conten",
			candidates:  candidates,
			expected:    "Content",
			description: "Should prefer the closer edit distance over substrings.",
		},
		{
			name:        "Multiple candidates, one is a superstring",
			input:       "ContentTypeExtra",
			candidates:  candidates,
			expected:    "ContentType",
			description: "Should correctly suggest a superstring if it's within the threshold.",
		},
		{
			name:        "Case where best match is lowercase",
			input:       "categorie",
			candidates:  candidates,
			expected:    "category",
			description: "Should return the candidate with its original casing.",
		},
		{
			name:        "Input is a complete subset of a candidate",
			input:       "Date",
			candidates:  candidates,
			expected:    "",
			description: "Should not suggest a much longer word just because it contains the input.",
		},
		{
			name:        "Input with repeated characters typo",
			input:       "Tiiitle",
			candidates:  candidates,
			expected:    "Title",
			description: "Should handle repeated character typos.",
		},
		{
			name:        "Complex typo with multiple edits",
			input:       "PublucationDete",
			candidates:  candidates,
			expected:    "PublicationDate",
			description: "Should find a match even with multiple errors.",
		},
		{
			name:        "Another tie-breaking test",
			input:       "Titles",
			candidates:  []string{"TitleBar", "Title"},
			expected:    "Title",
			description: "Should pick the one with the lower Levenshtein distance.",
		},
		{
			name:        "Whitespace in input",
			input:       "Author Name",
			candidates:  candidates,
			expected:    "AuthorName",
			description: "Should handle whitespace in input by comparing to a spaceless candidate.",
		},
		{
			name:        "Exact match but with different casing",
			input:       "ISENABLED",
			candidates:  candidates,
			expected:    "IsEnabled",
			description: "Should return the original cased candidate for an exact case-insensitive match.",
		},
		{
			name:        "Very long input with a close match",
			input:       "aVeryLongAndSpecificIdentifierName",
			candidates:  []string{"aVeryLongAndSpecificIdentifierNeme"},
			expected:    "aVeryLongAndSpecificIdentifierNeme",
			description: "Should work correctly for longer strings.",
		},
		{
			name:        "Very long input with no close match",
			input:       "aCompletelyUnrelatedLongIdentifier",
			candidates:  candidates,
			expected:    "",
			description: "Should correctly fail to find a match for long, dissimilar strings.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := findClosestMatch(tc.input, tc.candidates)
			assert.Equal(t, tc.expected, actual, tc.description)
		})
	}
}

func TestFindClosestMatch_WithSynonyms(t *testing.T) {
	candidates := []string{
		"DeleteUser",
		"CreatePost",
		"UpdateSettings",
		"FindItem",
		"DeletePost",
	}

	testCases := []struct {
		name        string
		input       string
		expected    string
		description string
		candidates  []string
	}{
		{
			name:        "Should suggest canonical term for a known synonym",
			input:       "RemoveUser",
			candidates:  candidates,
			expected:    "DeleteUser",
			description: "Should suggest 'DeleteUser' when 'RemoveUser' is typed.",
		},
		{
			name:        "Should suggest canonical term for a different synonym",
			input:       "AddPost",
			candidates:  candidates,
			expected:    "CreatePost",
			description: "Should suggest 'CreatePost' when 'AddPost' is typed.",
		},
		{
			name:        "Should return exact match for canonical term",
			input:       "DeleteUser",
			candidates:  candidates,
			expected:    "DeleteUser",
			description: "An exact match should be returned by the typo checker, not the synonym checker.",
		},
		{
			name:        "Typo correction should have priority over synonyms",
			input:       "DeletUser",
			candidates:  candidates,
			expected:    "DeleteUser",
			description: "A typo of 'DeleteUser' should be corrected and returned before synonym logic is even checked.",
		},
		{
			name:        "Should suggest correct synonym when subject matches",
			input:       "RemovePost",
			candidates:  candidates,
			expected:    "DeletePost",
			description: "Should suggest 'DeletePost' for 'RemovePost', matching both action and subject.",
		},
		{
			name:        "Should handle case variations in synonym lookup",
			input:       "removeuser",
			candidates:  candidates,
			expected:    "DeleteUser",
			description: "Synonym matching should be case-insensitive.",
		},
		{
			name:        "Should handle input with no subject",
			input:       "Erase",
			candidates:  []string{"Delete"},
			expected:    "Delete",
			description: "Should match synonyms even with no subject part.",
		},
		{
			name:        "No matching synonym found",
			input:       "QueryUser",
			candidates:  candidates,
			expected:    "",
			description: "Should return nothing if neither a typo nor a synonym match is found.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := findClosestMatch(tc.input, tc.candidates)
			assert.Equal(t, tc.expected, actual, tc.description)
		})
	}
}

func TestSplitActionAndSubject(t *testing.T) {
	testCases := []struct {
		input           string
		expectedAction  string
		expectedSubject string
	}{
		{input: "DeleteUser", expectedAction: "Delete", expectedSubject: "User"},
		{input: "Create", expectedAction: "Create", expectedSubject: ""},
		{input: "GetID", expectedAction: "Get", expectedSubject: "ID"},
		{input: "FindAllUsers", expectedAction: "Find", expectedSubject: "AllUsers"},
		{input: "aSimpleFunction", expectedAction: "a", expectedSubject: "SimpleFunction"},
		{input: "URLShortener", expectedAction: "URL", expectedSubject: "Shortener"},
		{input: "", expectedAction: "", expectedSubject: ""},
		{input: "lowercase", expectedAction: "lowercase", expectedSubject: ""},
		{input: "ID", expectedAction: "ID", expectedSubject: ""},
		{input: "HTMLParser", expectedAction: "HTML", expectedSubject: "Parser"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			action, subject := splitActionAndSubject(tc.input)
			assert.Equal(t, tc.expectedAction, action)
			assert.Equal(t, tc.expectedSubject, subject)
		})
	}
}
