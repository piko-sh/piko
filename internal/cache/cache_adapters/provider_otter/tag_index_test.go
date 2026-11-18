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
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagIndex_Invalidate_UnlinksKeyFromUnrequestedTags(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})

	invalidated := index.Invalidate([]string{"alpha"})

	if len(invalidated) != 1 || invalidated[0] != "k1" {
		t.Fatalf("Invalidate([alpha]) = %v, want [k1]", invalidated)
	}
	if remaining := index.Get("beta"); len(remaining) != 0 {
		t.Errorf("Get(\"beta\") = %v, want empty: the key is stranded in a tag it was invalidated out of", remaining)
	}
	if tags := index.GetTags("k1"); len(tags) != 0 {
		t.Errorf("GetTags(\"k1\") = %v, want empty", tags)
	}
}

func TestTagIndex_Invalidate_DoesNotReturnAKeyRetaggedSinceItsRemoval(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})
	index.Invalidate([]string{"alpha"})

	index.Add("k1", []string{"gamma"})

	if invalidated := index.Invalidate([]string{"beta"}); len(invalidated) != 0 {
		t.Errorf("Invalidate([beta]) = %v, want empty: k1 no longer carries beta", invalidated)
	}
	if remaining := index.Get("gamma"); len(remaining) != 1 {
		t.Errorf("Get(\"gamma\") = %v, want k1 still present", remaining)
	}
}

func TestTagIndex_Invalidate_LeavesUnrelatedKeysUntouched(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "shared"})
	index.Add("k2", []string{"shared"})

	index.Invalidate([]string{"alpha"})

	remaining := index.Get("shared")
	if len(remaining) != 1 {
		t.Fatalf("Get(\"shared\") = %v, want exactly k2", remaining)
	}
	if _, ok := remaining["k2"]; !ok {
		t.Errorf("Get(\"shared\") = %v, want k2", remaining)
	}
}

func TestTagIndex_Invalidate_EmptiesEveryBucketItDrains(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})
	index.Add("k2", []string{"beta", "gamma"})

	index.Invalidate([]string{"alpha", "gamma"})

	if remaining := index.Get("beta"); len(remaining) != 0 {
		t.Errorf("Get(\"beta\") = %v, want empty: both keys carried beta and both were invalidated", remaining)
	}
}

func TestTagIndex_SingleTagLinks(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.AddSingle("alpha", "k1")
	index.AddSingle("beta", "k1")

	assert.ElementsMatch(t, []string{"alpha", "beta"}, index.GetTags("k1"))

	index.RemoveSingle("alpha", "k1")
	assert.Equal(t, []string{"beta"}, index.GetTags("k1"))
	assert.Nil(t, index.Get("alpha"))

	index.RemoveSingle("beta", "k1")
	assert.Nil(t, index.GetTags("k1"))
}

func TestTagIndex_Add_ReplacesOnlyTheTagsThatChanged(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "shared"})
	index.Add("k2", []string{"shared"})

	index.Add("k1", []string{"shared", "gamma"})

	assert.ElementsMatch(t, []string{"shared", "gamma"}, index.GetTags("k1"))
	assert.Nil(t, index.Get("alpha"), "a tag no key holds any more must be dropped")
	assert.Equal(t, map[string]struct{}{"k1": {}, "k2": {}}, index.Get("shared"))
	assert.Equal(t, map[string]struct{}{"k1": {}}, index.Get("gamma"))
	assert.Equal(t, int64(2), index.taggedKeys.Load())
}

func TestTagIndex_Add_SameTagsLeavesTheIndexAsItWas(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})
	before := index.keyToTags["k1"]

	index.Add("k1", []string{"beta", "alpha"})

	assert.ElementsMatch(t, []string{"alpha", "beta"}, index.GetTags("k1"))
	assert.Equal(t, map[string]struct{}{"k1": {}}, index.Get("alpha"))
	assert.Equal(t, map[string]struct{}{"k1": {}}, index.Get("beta"))
	assert.True(t, reflect.ValueOf(before).UnsafePointer() == reflect.ValueOf(index.keyToTags["k1"]).UnsafePointer(),
		"rewriting the same tags must keep the key's tag set rather than rebuild it")
}

func TestTagIndex_Add_EmptyTagsKeepTheKeysTags(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha"})

	index.Add("k1", nil)

	assert.Equal(t, []string{"alpha"}, index.GetTags("k1"))
}

func TestTagIndex_Add_CutsToTheLimitBeforeDroppingRepeats(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.maxTagsPerKey = 2

	index.Add("k1", []string{"alpha", "alpha", "beta"})

	assert.Equal(t, []string{"alpha"}, index.GetTags("k1"))
	assert.Nil(t, index.Get("beta"))
}

func TestTagIndex_Add_ReplacesALongTagList(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	first := numberedStrings("tag", 0, 20)
	second := numberedStrings("tag", 5, 25)
	index.Add("k1", first)
	index.Add("k2", []string{"tag-0"})

	index.Add("k1", second)

	assert.ElementsMatch(t, second, index.GetTags("k1"))
	assert.Equal(t, map[string]struct{}{"k2": {}}, index.Get("tag-0"))
	assert.Nil(t, index.Get("tag-4"))
	assert.Equal(t, map[string]struct{}{"k1": {}}, index.Get("tag-24"))
}

func TestTagIndex_RestoreIfUntagged(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.restoreIfUntagged("k1", []string{"alpha", "beta"})
	assert.ElementsMatch(t, []string{"alpha", "beta"}, index.GetTags("k1"))

	index.Add("k2", []string{"gamma"})
	index.restoreIfUntagged("k2", []string{"alpha"})
	assert.Equal(t, []string{"gamma"}, index.GetTags("k2"), "a key tagged since must keep its new tags")
}

func TestUniqueStrings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "no values", values: nil, want: nil},
		{name: "short list without repeats", values: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "short list with repeats", values: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "long list without repeats", values: numberedStrings("v", 0, 20), want: numberedStrings("v", 0, 20)},
		{
			name:   "long list with repeats",
			values: append(numberedStrings("v", 0, 20), numberedStrings("v", 10, 15)...),
			want:   numberedStrings("v", 0, 20),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, uniqueStrings(testCase.values))
		})
	}
}

func TestStringLookup(t *testing.T) {
	t.Parallel()

	for _, size := range []int{3, linearStringSearchLimit, linearStringSearchLimit + 5} {
		t.Run(fmt.Sprintf("size %d", size), func(t *testing.T) {
			t.Parallel()
			lookup := newStringLookup(numberedStrings("v", 0, size))
			assert.True(t, lookup.contains("v-0"))
			assert.True(t, lookup.contains(fmt.Sprintf("v-%d", size-1)))
			assert.False(t, lookup.contains(fmt.Sprintf("v-%d", size)))
			assert.Equal(t, size > linearStringSearchLimit, lookup.set != nil)
		})
	}
}

func numberedStrings(prefix string, from, to int) []string {
	values := make([]string, 0, to-from)
	for number := from; number < to; number++ {
		values = append(values, fmt.Sprintf("%s-%d", prefix, number))
	}
	return values
}
