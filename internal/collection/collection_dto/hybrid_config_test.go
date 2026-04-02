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

package collection_dto

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultHybridConfig(t *testing.T) {
	t.Parallel()

	config := DefaultHybridConfig()

	assert.Equal(t, 60*time.Second, config.RevalidationTTL)
	assert.True(t, config.StaleIfError)
	assert.Equal(t, time.Duration(0), config.MaxStaleAge)
	assert.Equal(t, ETagSourceModtimeHash, config.ETagSource)
}

func TestHybridRevalidationResult_IsValid(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		r := &HybridRevalidationResult{}
		assert.True(t, r.IsValid())
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		r := &HybridRevalidationResult{Error: errors.New("timeout")}
		assert.False(t, r.IsValid())
	})
}

func TestHybridRevalidationResult_NeedsUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		r    *HybridRevalidationResult
		name string
		want bool
	}{
		{
			name: "needs update",
			r: &HybridRevalidationResult{
				ETagChanged: true,
				NewItems:    []ContentItem{{ID: "1"}},
			},
			want: true,
		},
		{
			name: "etag unchanged",
			r: &HybridRevalidationResult{
				ETagChanged: false,
				NewItems:    []ContentItem{{ID: "1"}},
			},
			want: false,
		},
		{
			name: "no new items",
			r: &HybridRevalidationResult{
				ETagChanged: true,
				NewItems:    nil,
			},
			want: false,
		},
		{
			name: "has error",
			r: &HybridRevalidationResult{
				Error:       errors.New("failed"),
				ETagChanged: true,
				NewItems:    []ContentItem{{ID: "1"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.r.NeedsUpdate())
		})
	}
}
