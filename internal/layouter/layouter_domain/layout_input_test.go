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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSizingMode_String(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		value    SizingMode
	}{
		{
			name:     "normal returns the correct keyword",
			value:    SizingModeNormal,
			expected: "normal",
		},
		{
			name:     "min-content returns the correct keyword",
			value:    SizingModeMinContent,
			expected: "min-content",
		},
		{
			name:     "max-content returns the correct keyword",
			value:    SizingModeMaxContent,
			expected: "max-content",
		},
		{
			name:     "out-of-range value returns unknown",
			value:    SizingMode(99),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.String())
		})
	}
}
