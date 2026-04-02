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

func TestSortedIndex_Add_Ascending(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)

	keys := index.Keys(true)
	expected := []string{"a", "b", "c"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Add_Descending(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("c", 3.0)
	index.Add("b", 2.0)
	index.Add("a", 1.0)

	keys := index.Keys(true)
	expected := []string{"a", "b", "c"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Add_Random(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("b", 2.0)
	index.Add("d", 4.0)
	index.Add("a", 1.0)
	index.Add("c", 3.0)

	keys := index.Keys(true)
	expected := []string{"a", "b", "c", "d"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Add_Update(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)

	index.Add("a", 4.0)

	keys := index.Keys(true)
	expected := []string{"b", "c", "a"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Keys_Descending(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)

	keys := index.Keys(false)
	expected := []string{"c", "b", "a"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Remove(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)

	index.Remove("b")

	keys := index.Keys(true)
	expected := []string{"a", "c"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys after removal, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Remove_NonExistent(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)

	index.Remove("nonexistent")

	keys := index.Keys(true)
	if len(keys) != 2 {
		t.Errorf("expected 2 keys after removing non-existent, got %d", len(keys))
	}
}

func TestSortedIndex_KeysFiltered(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)
	index.Add("d", 4.0)
	index.Add("e", 5.0)

	filter := map[string]struct{}{
		"a": {},
		"c": {},
		"e": {},
	}

	keys := index.KeysFiltered(filter, true)
	expected := []string{"a", "c", "e"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d filtered keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_KeysFiltered_Descending(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)
	index.Add("d", 4.0)
	index.Add("e", 5.0)

	filter := map[string]struct{}{
		"a": {},
		"c": {},
		"e": {},
	}

	keys := index.KeysFiltered(filter, false)
	expected := []string{"e", "c", "a"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d filtered keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_Clear(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 2.0)
	index.Add("c", 3.0)

	index.Clear()

	keys := index.Keys(true)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after clear, got %d", len(keys))
	}

	if index.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", index.Size())
	}
}

func TestSortedIndex_Size(t *testing.T) {
	index := NewSortedIndex[string]()

	if index.Size() != 0 {
		t.Errorf("expected initial size 0, got %d", index.Size())
	}

	index.Add("a", 1.0)
	if index.Size() != 1 {
		t.Errorf("expected size 1 after adding, got %d", index.Size())
	}

	index.Add("b", 2.0)
	if index.Size() != 2 {
		t.Errorf("expected size 2 after adding, got %d", index.Size())
	}

	index.Remove("a")
	if index.Size() != 1 {
		t.Errorf("expected size 1 after removing, got %d", index.Size())
	}
}

func TestSortedIndex_DuplicateValues(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 1.0)
	index.Add("b", 1.0)
	index.Add("c", 2.0)

	keys := index.Keys(true)

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	foundC := false
	for i, key := range keys {
		if key == "c" {
			foundC = true
			if i != 2 {
				t.Errorf("expected 'c' to be last (position 2), got position %d", i)
			}
		}
	}

	if !foundC {
		t.Error("expected 'c' in results")
	}
}

func TestSortedIndex_StringValues(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", "zebra")
	index.Add("b", "apple")
	index.Add("c", "monkey")

	keys := index.Keys(true)
	expected := []string{"b", "c", "a"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_IntValues(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 100)
	index.Add("b", 50)
	index.Add("c", 200)

	keys := index.Keys(true)
	expected := []string{"b", "a", "c"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], key)
		}
	}
}

func TestSortedIndex_MixedTypes(t *testing.T) {
	index := NewSortedIndex[string]()

	index.Add("a", 100)
	index.Add("b", "200")
	index.Add("c", 50)

	keys := index.Keys(true)

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
}

func TestSortedIndex_Empty(t *testing.T) {
	index := NewSortedIndex[string]()

	keys := index.Keys(true)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys from empty index, got %d", len(keys))
	}

	filter := map[string]struct{}{"a": {}}
	keys = index.KeysFiltered(filter, true)
	if len(keys) != 0 {
		t.Errorf("expected 0 filtered keys from empty index, got %d", len(keys))
	}

	if index.Size() != 0 {
		t.Errorf("expected size 0 for empty index, got %d", index.Size())
	}
}

func TestSortedIndex_LargeDataset(t *testing.T) {
	index := NewSortedIndex[int]()

	for i := range 1000 {
		index.Add(i, float64(i))
	}

	keys := index.Keys(true)

	if len(keys) != 1000 {
		t.Fatalf("expected 1000 keys, got %d", len(keys))
	}

	for i := range 999 {
		if keys[i] >= keys[i+1] {
			t.Errorf("keys not in ascending order at position %d: %d >= %d", i, keys[i], keys[i+1])
		}
	}

	keysDesc := index.Keys(false)
	for i := range 999 {
		if keysDesc[i] <= keysDesc[i+1] {
			t.Errorf("keys not in descending order at position %d: %d <= %d", i, keysDesc[i], keysDesc[i+1])
		}
	}
}

func TestSortedIndex_Concurrent(t *testing.T) {
	index := NewSortedIndex[int]()

	for i := range 100 {
		index.Add(i, float64(i))
	}

	done := make(chan bool)

	for range 5 {
		go func() {
			for i := range 20 {
				index.Add(i+100, float64(i+100))
			}
			done <- true
		}()
	}

	for range 5 {
		go func() {
			for range 20 {
				index.Keys(true)
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}

	keys := index.Keys(true)
	if len(keys) == 0 {
		t.Error("expected some keys after concurrent operations")
	}

	for i := range len(keys) - 1 {
		if keys[i] >= keys[i+1] {
			t.Errorf("keys not sorted after concurrent operations at position %d", i)
		}
	}
}

func TestSortedIndex_RangeQueries(t *testing.T) {
	index := NewSortedIndex[int]()

	for i := range 10 {
		index.Add(i, float64(i*10))
	}

	t.Run("greater_than", func(t *testing.T) {

		keys := index.KeysGreaterThan(45.0, true)
		expected := []int{5, 6, 7, 8, 9}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})

	t.Run("greater_than_or_equal", func(t *testing.T) {

		keys := index.KeysGreaterThanOrEqual(50.0, true)
		expected := []int{5, 6, 7, 8, 9}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})

	t.Run("less_than", func(t *testing.T) {

		keys := index.KeysLessThan(35.0, true)
		expected := []int{0, 1, 2, 3}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})

	t.Run("less_than_or_equal", func(t *testing.T) {

		keys := index.KeysLessThanOrEqual(30.0, true)
		expected := []int{0, 1, 2, 3}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})

	t.Run("between", func(t *testing.T) {

		keys := index.KeysBetween(20.0, 60.0, true)
		expected := []int{2, 3, 4, 5, 6}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})

	t.Run("descending_order", func(t *testing.T) {

		keys := index.KeysGreaterThan(45.0, false)
		expected := []int{9, 8, 7, 6, 5}
		if len(keys) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(keys))
		}
		for i, key := range keys {
			if key != expected[i] {
				t.Errorf("at position %d: expected %d, got %d", i, expected[i], key)
			}
		}
	})
}
