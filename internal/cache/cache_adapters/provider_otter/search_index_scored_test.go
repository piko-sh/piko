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
	"strings"
	"testing"
)

func TestInvertedIndex_SearchScored_SingleTerm(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})

	results := index.SearchScored("hello")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'hello', got %d", len(results))
	}

	for _, r := range results {
		if r.Score <= 0 {
			t.Errorf("expected positive BM25 score for key %v, got %f", r.Key, r.Score)
		}
	}
}

func TestInvertedIndex_SearchScored_MultiTerm_AND(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})
	index.Add("doc4", []string{"hello world news"})

	results := index.SearchScored("hello world")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'hello world' (AND logic), got %d", len(results))
	}

	foundKeys := make(map[string]bool)
	for _, r := range results {
		foundKeys[r.Key] = true
	}

	if !foundKeys["doc1"] {
		t.Error("expected doc1 in results (contains both 'hello' and 'world')")
	}
	if !foundKeys["doc4"] {
		t.Error("expected doc4 in results (contains both 'hello' and 'world')")
	}
}

func TestInvertedIndex_SearchScored_NoMatch(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.SearchScored("nonexistent")
	if results != nil {
		t.Errorf("expected nil for non-existent term, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_EmptyQuery(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.SearchScored("")
	if results != nil {
		t.Errorf("expected nil for empty query, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_EmptyIndex(t *testing.T) {
	index := NewInvertedIndex[string]()

	results := index.SearchScored("hello")
	if results != nil {
		t.Errorf("expected nil for empty index, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_Ordering(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"premium widget"})
	index.Add("doc2", []string{"premium basic standard deluxe ultra eco widget gadget device tool item"})

	results := index.SearchScored("premium")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Score <= results[1].Score {
		t.Errorf("expected first result to have higher score than second: %f <= %f",
			results[0].Score, results[1].Score)
	}
}

func TestInvertedIndex_SearchScored_SortedDescending(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"alpha beta gamma delta"})
	index.Add("doc2", []string{"alpha"})
	index.Add("doc3", []string{"alpha beta"})

	results := index.SearchScored("alpha")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted descending: index %d score %f > index %d score %f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestInvertedIndex_SearchScored_AfterRemove(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	index.Remove("doc1")

	results := index.SearchScored("hello")
	if len(results) != 1 {
		t.Fatalf("expected 1 result after removal, got %d", len(results))
	}
	if results[0].Key != "doc2" {
		t.Errorf("expected doc2, got %v", results[0].Key)
	}
}

func TestInvertedIndex_SearchScored_AfterClear(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	index.Clear()

	results := index.SearchScored("hello")
	if results != nil {
		t.Errorf("expected nil after clear, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_WithAnalyseFunc(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.SetAnalyseFunction(func(text string) []string {
		words := strings.Fields(strings.ToLower(text))
		seen := make(map[string]struct{})
		var result []string
		for _, w := range words {
			if _, ok := seen[w]; !ok {
				seen[w] = struct{}{}
				result = append(result, w)
			}
		}
		return result
	})

	index.Add("doc1", []string{"running quickly"})
	index.Add("doc2", []string{"run slow"})

	results := index.SearchScored("running")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'running', got %d", len(results))
	}
	if results[0].Key != "doc1" {
		t.Errorf("expected doc1, got %v", results[0].Key)
	}
}

func TestInvertedIndex_SearchScored_StemmedAnalyseFunc(t *testing.T) {
	index := NewInvertedIndex[string]()

	stemmer := func(text string) []string {
		words := strings.Fields(strings.ToLower(text))
		seen := make(map[string]struct{})
		var result []string
		for _, w := range words {

			stem := w
			for _, suffix := range []string{"ing", "ly", "ed", "s"} {
				if len(stem) > len(suffix)+2 && strings.HasSuffix(stem, suffix) {
					stem = strings.TrimSuffix(stem, suffix)
					break
				}
			}
			if _, ok := seen[stem]; !ok {
				seen[stem] = struct{}{}
				result = append(result, stem)
			}
		}
		return result
	}
	index.SetAnalyseFunction(stemmer)

	index.Add("doc1", []string{"running quickly"})
	index.Add("doc2", []string{"run slow"})
	index.Add("doc3", []string{"walked happily"})

	results := index.SearchScored("walked")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'walked' (stemmed), got %d", len(results))
	}
	if results[0].Key != "doc3" {
		t.Errorf("expected doc3, got %v", results[0].Key)
	}
}

func TestInvertedIndex_SearchScored_BackwardCompatible(t *testing.T) {

	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})

	searchResults := index.Search("hello world")
	scoredResults := index.SearchScored("hello world")

	if len(searchResults) != len(scoredResults) {
		t.Fatalf("Search returned %d results, SearchScored returned %d - should match",
			len(searchResults), len(scoredResults))
	}

	searchKeys := make(map[string]bool)
	for _, k := range searchResults {
		searchKeys[k] = true
	}

	for _, r := range scoredResults {
		if !searchKeys[r.Key] {
			t.Errorf("SearchScored returned key %v not found in Search results", r.Key)
		}
	}
}

func TestInvertedIndex_BM25_DocumentLengthTracking(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there friend"})

	if index.totalDocuments != 2 {
		t.Errorf("expected totalDocuments=2, got %d", index.totalDocuments)
	}

	if index.totalTerms != 5 {
		t.Errorf("expected totalTerms=5, got %d", index.totalTerms)
	}

	if index.docLengths["doc1"] != 2 {
		t.Errorf("expected doc1 length=2, got %d", index.docLengths["doc1"])
	}
	if index.docLengths["doc2"] != 3 {
		t.Errorf("expected doc2 length=3, got %d", index.docLengths["doc2"])
	}

	index.Remove("doc1")

	if index.totalDocuments != 1 {
		t.Errorf("after remove: expected totalDocuments=1, got %d", index.totalDocuments)
	}
	if index.totalTerms != 3 {
		t.Errorf("after remove: expected totalTerms=3, got %d", index.totalTerms)
	}
}

func TestInvertedIndex_BM25_ClearResetsStats(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	index.Clear()

	if index.totalDocuments != 0 {
		t.Errorf("expected totalDocuments=0 after clear, got %d", index.totalDocuments)
	}
	if index.totalTerms != 0 {
		t.Errorf("expected totalTerms=0 after clear, got %d", index.totalTerms)
	}
	if len(index.docLengths) != 0 {
		t.Errorf("expected empty docLengths after clear, got %d entries", len(index.docLengths))
	}
}

func TestInvertedIndex_SearchScored_Concurrent(t *testing.T) {
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
				index.SearchScored("document")
			}
			done <- true
		}()
	}

	for range 6 {
		<-done
	}

	results := index.SearchScored("document")
	if len(results) == 0 {
		t.Error("expected some results after concurrent operations")
	}

	for _, r := range results {
		if r.Score <= 0 {
			t.Errorf("expected positive score, got %f for key %v", r.Score, r.Key)
		}
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_ReturnsPartialMatches(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})
	index.Add("doc3", []string{"world news"})
	index.Add("doc4", []string{"unrelated content"})

	andResults := index.SearchScored("hello world")

	orResults := index.SearchScored("hello world", WithTermMatch(TermMatchAny))

	if len(andResults) != 1 {
		t.Errorf("AND: expected 1 result (only doc1 has both terms), got %d", len(andResults))
	}
	if len(orResults) != 3 {
		t.Errorf("OR: expected 3 results (doc1, doc2, doc3), got %d", len(orResults))
	}

	for _, r := range orResults {
		if r.Key == "doc4" {
			t.Error("OR: doc4 should not match (has neither 'hello' nor 'world')")
		}
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_MoreTermsScoreHigher(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("both", []string{"hello world"})
	index.Add("one", []string{"hello there"})

	results := index.SearchScored("hello world", WithTermMatch(TermMatchAny))
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Key != "both" {
		t.Errorf("expected 'both' to rank first (matches 2/2 terms), got %v", results[0].Key)
	}
	if results[0].Score <= results[1].Score {
		t.Errorf("doc matching 2 terms should outscore doc matching 1 term: %f <= %f",
			results[0].Score, results[1].Score)
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_SingleTerm(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	andResults := index.SearchScored("hello")
	orResults := index.SearchScored("hello", WithTermMatch(TermMatchAny))

	if len(andResults) != len(orResults) {
		t.Fatalf("single term: AND returned %d, OR returned %d - should match",
			len(andResults), len(orResults))
	}

	andKeys := make(map[string]bool)
	for _, r := range andResults {
		andKeys[r.Key] = true
	}
	for _, r := range orResults {
		if !andKeys[r.Key] {
			t.Errorf("single term: OR returned key %v not in AND results", r.Key)
		}
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_NoMatch(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.SearchScored("nonexistent", WithTermMatch(TermMatchAny))
	if results != nil {
		t.Errorf("OR: expected nil for non-existent term, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_EmptyQuery(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})

	results := index.SearchScored("", WithTermMatch(TermMatchAny))
	if results != nil {
		t.Errorf("OR: expected nil for empty query, got %d results", len(results))
	}
}

func TestInvertedIndex_SearchScored_TermMatchAny_MultiWordQueryPartialOverlap(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"example configuration file"})
	index.Add("doc2", []string{"pk encryption key"})
	index.Add("doc3", []string{"example pk setup"})
	index.Add("doc4", []string{"unrelated document"})

	andResults := index.SearchScored("example pk file")
	if len(andResults) != 0 {
		t.Errorf("AND: expected 0 results, got %d", len(andResults))
	}

	orResults := index.SearchScored("example pk file", WithTermMatch(TermMatchAny))
	if len(orResults) != 3 {
		t.Fatalf("OR: expected 3 results, got %d", len(orResults))
	}

	topTwo := map[string]bool{orResults[0].Key: true, orResults[1].Key: true}
	if !topTwo["doc1"] || !topTwo["doc3"] {
		t.Errorf("OR: expected doc1 and doc3 in top 2 (both match 2/3 terms), got %v and %v",
			orResults[0].Key, orResults[1].Key)
	}
	if orResults[2].Key != "doc2" {
		t.Errorf("OR: expected doc2 last (matches 1/3 terms), got %v", orResults[2].Key)
	}

	for _, r := range orResults {
		if r.Key == "doc4" {
			t.Error("OR: doc4 should not match")
		}
	}
}

func TestInvertedIndex_SearchScored_DefaultIsAND(t *testing.T) {
	index := NewInvertedIndex[string]()

	index.Add("doc1", []string{"hello world"})
	index.Add("doc2", []string{"hello there"})

	results := index.SearchScored("hello world")
	if len(results) != 1 {
		t.Errorf("default (AND): expected 1 result for 'hello world', got %d", len(results))
	}

	explicitAnd := index.SearchScored("hello world", WithTermMatch(TermMatchAll))
	if len(explicitAnd) != 1 {
		t.Errorf("explicit AND: expected 1 result, got %d", len(explicitAnd))
	}
}
