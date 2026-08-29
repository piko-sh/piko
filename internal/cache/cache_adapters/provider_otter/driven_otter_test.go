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
