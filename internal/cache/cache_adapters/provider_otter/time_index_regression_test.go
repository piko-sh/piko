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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeIndexOrdersInstantsAndPagesAfterRemovedCursor(t *testing.T) {
	idx := NewSortedIndex[string]()
	now := time.Now()
	idx.Add("a", now)
	idx.Add("b", now.UTC())
	idx.Add("c", now.In(time.FixedZone("offset", 3600)))
	idx.Add("future", now.Add(time.Hour))
	require.Equal(t, []string{"a", "b", "c"}, idx.KeysLessThanOrEqualLimit(now, 10))
	require.Equal(t, []string{"a", "b"}, idx.KeysAfterThrough("", nil, now, 2))
	idx.Remove("b")
	require.Equal(t, []string{"c"}, idx.KeysAfterThrough("b", now, now, 2))
	require.Empty(t, idx.KeysLessThanOrEqualLimit(now, 0))
}
func BenchmarkDueTimeIndex(b *testing.B) {
	for _, history := range []int{1000, 100000} {
		b.Run(fmt.Sprint(history), func(b *testing.B) {
			idx := NewSortedIndex[int]()
			now := time.Now()
			for i := range history {
				idx.Add(i, now.Add(time.Hour))
			}
			for i := range 100 {
				idx.Add(history+i, now)
			}
			b.ReportAllocs()
			for b.Loop() {
				_ = idx.KeysLessThanOrEqualLimit(now, 100)
			}
		})
	}
}
