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

package config

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cssinliner"
)

func TestBuildModeConfig_CSSImportDefaultsMatchTheCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field string
		want  int
	}{
		{
			name:  "import depth",
			field: "CSSImportMaxDepth",
			want:  cssinliner.DefaultMaxImportDepth,
		},
		{
			name:  "total inlined bytes",
			field: "CSSImportMaxBytes",
			want:  cssinliner.DefaultMaxInlinedBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field, found := reflect.TypeFor[BuildModeConfig]().FieldByName(tt.field)
			require.True(t, found, "field %s must exist", tt.field)

			tag, hasTag := field.Tag.Lookup("default")
			require.True(t, hasTag, "field %s must carry a default tag", tt.field)

			tagged, err := strconv.Atoi(tag)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tagged,
				"the default tag on %s must match the constant the inliner falls back to", tt.field)
		})
	}
}
