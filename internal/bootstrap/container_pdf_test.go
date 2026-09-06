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

package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

func TestWithPdfLayoutLimits_SetsOnlyPositiveFields(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	WithPdfLayoutLimits(layouter_dto.LayoutLimits{
		MaxRawHTMLBytes: 1,
		MaxNestingDepth: 2,
		MaxBoxNodes:     3,
		MaxColspan:      4,
		MaxRowspan:      5,
		MaxTableColumns: 6,
		MaxGridTracks:   7,
		MaxGridCells:    8,
		MaxRepeatCount:  9,
		MaxPages:        0,
	})(c)

	pdf := c.ensureOverrides().Pdf
	require.NotNil(t, pdf.MaxRawHTMLBytes)
	assert.Equal(t, 1, *pdf.MaxRawHTMLBytes)
	assert.Equal(t, 2, *pdf.MaxNestingDepth)
	assert.Equal(t, 3, *pdf.MaxBoxNodes)
	assert.Equal(t, 4, *pdf.MaxColspan)
	assert.Equal(t, 5, *pdf.MaxRowspan)
	assert.Equal(t, 6, *pdf.MaxTableColumns)
	assert.Equal(t, 7, *pdf.MaxGridTracks)
	assert.Equal(t, 8, *pdf.MaxGridCells)
	assert.Equal(t, 9, *pdf.MaxRepeatCount)
	assert.Nil(t, pdf.MaxPages, "an unset limit leaves the override unset")
}

func TestWithPdfMaxImagePixels(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	WithPdfMaxImagePixels(0)(c)
	assert.Nil(t, c.ensureOverrides().Pdf.MaxImagePixels)

	WithPdfMaxImagePixels(500)(c)
	require.NotNil(t, c.ensureOverrides().Pdf.MaxImagePixels)
	assert.Equal(t, 500, *c.ensureOverrides().Pdf.MaxImagePixels)
}

func TestPdfLayoutLimits(t *testing.T) {
	t.Parallel()

	assert.Equal(t, layouter_dto.LayoutLimits{}, pdfLayoutLimits(config.PdfConfig{}), "unset fields stay zero for the defaults")

	limits := pdfLayoutLimits(config.PdfConfig{
		MaxRawHTMLBytes: new(10),
		MaxNestingDepth: new(11),
		MaxBoxNodes:     new(12),
		MaxColspan:      new(13),
		MaxRowspan:      new(14),
		MaxTableColumns: new(15),
		MaxGridTracks:   new(16),
		MaxGridCells:    new(17),
		MaxRepeatCount:  new(18),
		MaxPages:        new(19),
		MaxImagePixels:  new(20),
	})
	assert.Equal(t, layouter_dto.LayoutLimits{
		MaxRawHTMLBytes: 10,
		MaxNestingDepth: 11,
		MaxBoxNodes:     12,
		MaxColspan:      13,
		MaxRowspan:      14,
		MaxTableColumns: 15,
		MaxGridTracks:   16,
		MaxGridCells:    17,
		MaxRepeatCount:  18,
		MaxPages:        19,
	}, limits)

	assert.Len(t, pdfServiceLimitOptions(config.PdfConfig{}), 2)
}
