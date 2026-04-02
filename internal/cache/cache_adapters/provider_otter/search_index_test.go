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

package provider_otter

import (
	"testing"
)

func TestInvertedIndex_Add(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})

	results := index.Search("hello")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'hello', got %d", len(results))
	}

	results = index.Search("world")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'world', got %d", len(results))
	}

	results = index.Search("news")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'news', got %d", len(results))
	}
}

func TestInvertedIndex_Add_Update(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.Search("hello")
	if len(results) != 1 {
		t.Fatalf("expected 1 result initially, got %d", len(results))
	}

	index.Add("doc1", []string{"goodbye universe"})

	results = index.Search("hello")
	if len(results) != 0 {
		t.Errorf("expected 0 results for old term 'hello', got %d", len(results))
	}

	results = index.Search("world")
	if len(results) != 0 {
		t.Errorf("expected 0 results for old term 'world', got %d", len(results))
	}

	results = index.Search("goodbye")
	if len(results) != 1 {
		t.Errorf("expected 1 result for new term 'goodbye', got %d", len(results))
	}

	results = index.Search("universe")
	if len(results) != 1 {
		t.Errorf("expected 1 result for new term 'universe', got %d", len(results))
	}
}

func TestInvertedIndex_Search_MultiTerm(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})
	index.Add("doc4", []string{"hello world news"})

	results := index.Search("hello world")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'hello world', got %d", len(results))
	}

	results = index.Search("hello world news")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'hello world news', got %d", len(results))
	}

	results = index.Search("hello there")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'hello there', got %d", len(results))
	}
}

func TestInvertedIndex_Search_NoMatch(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("expected nil or empty slice for non-existent term, got %d results", len(results))
	}
}

func TestInvertedIndex_Search_EmptyQuery(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.Search("")
	if len(results) != 0 {
		t.Errorf("expected nil or empty slice for empty query, got %d results", len(results))
	}
}

func TestInvertedIndex_Remove(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})

	index.Remove("doc1")

	results := index.Search("hello")
	if len(results) != 1 {
		t.Errorf("expected 1 result after removing doc1, got %d", len(results))
	}
	if results[0] != "doc2" {
		t.Errorf("expected doc2 in results, got %s", results[0])
	}

	results = index.Search("world")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'world' after removing doc1, got %d", len(results))
	}
	if results[0] != "doc3" {
		t.Errorf("expected doc3 in results, got %s", results[0])
	}
}

func TestInvertedIndex_Remove_NonExistent(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	index.Remove("nonexistent")

	results := index.Search("hello")
	if len(results) != 1 {
		t.Errorf("expected 1 result after removing non-existent document, got %d", len(results))
	}
}

func TestInvertedIndex_Clear(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	index.Clear()

	results := index.Search("hello")
	if len(results) != 0 {
		t.Errorf("expected empty index after Clear, got %d results", len(results))
	}
}

func TestInvertedIndex_Tokenize(t *testing.T) {
	tests := []struct {
		expected map[string]bool
		name     string
		text     string
	}{
		{
			name: "basic text",
			text: "hello world",
			expected: map[string]bool{
				"hello": true,
				"world": true,
			},
		},
		{
			name: "with punctuation",
			text: "hello, world!",
			expected: map[string]bool{
				"hello": true,
				"world": true,
			},
		},
		{
			name: "with numbers",
			text: "hello123 world456",
			expected: map[string]bool{
				"hello123": true,
				"world456": true,
			},
		},
		{
			name: "case insensitive",
			text: "HELLO World",
			expected: map[string]bool{
				"hello": true,
				"world": true,
			},
		},
		{
			name: "single char terms filtered",
			text: "a b c hello",
			expected: map[string]bool{
				"hello": true,
			},
		},
		{
			name: "mixed case and punctuation",
			text: "The Quick Brown-Fox, jumps!",
			expected: map[string]bool{
				"the":   true,
				"quick": true,
				"brown": true,
				"fox":   true,
				"jumps": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testIndex := NewInvertedIndex[string]()
			testIndex.Add("doc1", []string{tt.text})

			for term := range tt.expected {
				results := testIndex.Search(term)
				if len(results) != 1 {
					t.Errorf("term '%s': expected 1 result, got %d", term, len(results))
				}
			}

			unexpectedTerms := []string{"xyz", "nonexistent", "a", "b"}
			for _, term := range unexpectedTerms {
				if tt.expected[term] {
					continue
				}
				results := testIndex.Search(term)
				if len(results) != 0 {
					t.Errorf("unexpected term '%s' matched, got %d results", term, len(results))
				}
			}
		})
	}
}

func TestInvertedIndex_MultipleTexts(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello", "world", "news"})

	for _, term := range []string{"hello", "world", "news"} {
		results := index.Search(term)
		if len(results) != 1 {
			t.Errorf("term '%s': expected 1 result, got %d", term, len(results))
		}
		if results[0] != "doc1" {
			t.Errorf("term '%s': expected doc1, got %s", term, results[0])
		}
	}
}

func TestInvertedIndex_DuplicateTerms(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello hello", "hello world"})

	results := index.Search("hello")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'hello', got %d", len(results))
	}
	if results[0] != "doc1" {
		t.Errorf("expected doc1, got %s", results[0])
	}
}

func TestInvertedIndex_Search_PartialTermMatch(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello"})

	results := index.Search("hel")
	if len(results) != 0 {
		t.Errorf("expected 0 results for partial match 'hel', got %d", len(results))
	}

	results = index.Search("hello")
	if len(results) != 1 {
		t.Errorf("expected 1 result for exact match 'hello', got %d", len(results))
	}
}

func TestInvertedIndex_EmptyText(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{""})
	index.Add("doc2", []string{"hello"})

	results := index.Search("hello")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestInvertedIndex_Concurrent(t *testing.T) {
	index := NewInvertedIndex[string]()

	for i := range 100 {
		index.Add(string(rune('A'+i)), []string{"document text"})
	}

	done := make(chan bool)

	go func() {
		for i := range 50 {
			index.Add(string(rune('Z'+i)), []string{"new document"})
		}
		done <- true
	}()

	for range 5 {
		go func() {
			for range 20 {
				index.Search("document")
			}
			done <- true
		}()
	}

	for range 6 {
		<-done
	}

	results := index.Search("document")
	if len(results) == 0 {
		t.Error("expected some results after concurrent operations")
	}
}
