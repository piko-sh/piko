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

	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

func TestPdfConfig_DefaultsMatchTheCode(t *testing.T) {
	t.Parallel()

	defaults := layouter_dto.LayoutLimits{}.Resolved()
	tests := []struct {
		name  string
		field string
		want  int
	}{
		{name: "raw HTML bytes", field: "MaxRawHTMLBytes", want: defaults.MaxRawHTMLBytes},
		{name: "nesting depth", field: "MaxNestingDepth", want: defaults.MaxNestingDepth},
		{name: "box nodes", field: "MaxBoxNodes", want: defaults.MaxBoxNodes},
		{name: "colspan", field: "MaxColspan", want: defaults.MaxColspan},
		{name: "rowspan", field: "MaxRowspan", want: defaults.MaxRowspan},
		{name: "table columns", field: "MaxTableColumns", want: defaults.MaxTableColumns},
		{name: "grid tracks", field: "MaxGridTracks", want: defaults.MaxGridTracks},
		{name: "grid cells", field: "MaxGridCells", want: defaults.MaxGridCells},
		{name: "repeat count", field: "MaxRepeatCount", want: defaults.MaxRepeatCount},
		{name: "pages", field: "MaxPages", want: defaults.MaxPages},
		{name: "image pixels", field: "MaxImagePixels", want: pdfwriter_domain.DefaultMaxImagePixels},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field, found := reflect.TypeFor[PdfConfig]().FieldByName(tt.field)
			require.True(t, found, "field %s must exist", tt.field)

			tag, hasTag := field.Tag.Lookup("default")
			require.True(t, hasTag, "field %s must carry a default tag", tt.field)

			tagged, err := strconv.Atoi(tag)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tagged,
				"the default tag on %s must match the limit the PDF pipeline falls back to", tt.field)
		})
	}
}
