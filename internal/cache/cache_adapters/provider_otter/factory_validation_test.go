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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
)

func TestOtterProviderFactory_RejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		options cache_dto.Options[string, string]
	}{
		{
			name:    "weight bound without a weigher",
			options: cache_dto.Options[string, string]{MaximumWeight: 1024},
		},
		{
			name: "weigher without a weight bound",
			options: cache_dto.Options[string, string]{
				Weigher: func(_ string, value string) uint32 { return uint32(len(value)) },
			},
		},
		{
			name: "both bounds at once",
			options: cache_dto.Options[string, string]{
				MaximumEntries: 100,
				MaximumWeight:  1024,
				Weigher:        func(_ string, value string) uint32 { return uint32(len(value)) },
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			provider, err := OtterProviderFactory(testCase.options)

			require.Error(t, err, "the public factory must validate, not just the service path")
			assert.Nil(t, provider)
		})
	}
}

func TestOtterProviderFactory_AcceptsAValidWeightBound(t *testing.T) {
	t.Parallel()

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumWeight: 1024,
		Weigher:       func(_ string, value string) uint32 { return uint32(len(value)) },
	})

	require.NoError(t, err)
	require.NotNil(t, provider)
}
