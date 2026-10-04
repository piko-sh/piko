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

// PdfConfig holds the limits applied when rendering PDF documents.
//
// Raw HTML inserted with p-html reaches the PDF layouter, so these bound the memory and
// CPU a hostile document can consume. Every default is generous enough to stay out of the
// way of genuine documents.
type PdfConfig struct {
	// MaxRawHTMLBytes is the maximum total size of raw HTML expanded while laying out one
	// document.
	MaxRawHTMLBytes *int `json:"maxRawHtmlBytes" yaml:"maxRawHtmlBytes" default:"4194304" env:"PIKO_PDF_MAX_RAW_HTML_BYTES" flag:"pdfMaxRawHtmlBytes" usage:"Maximum total bytes of raw HTML expanded in one PDF."`

	// MaxNestingDepth is the maximum element nesting depth of a document.
	MaxNestingDepth *int `json:"maxNestingDepth" yaml:"maxNestingDepth" default:"512" env:"PIKO_PDF_MAX_NESTING_DEPTH" flag:"pdfMaxNestingDepth" usage:"Maximum element nesting depth of a PDF document."`

	// MaxBoxNodes is the maximum number of document nodes and layout boxes in one layout.
	MaxBoxNodes *int `json:"maxBoxNodes" yaml:"maxBoxNodes" default:"200000" env:"PIKO_PDF_MAX_BOX_NODES" flag:"pdfMaxBoxNodes" usage:"Maximum document nodes and layout boxes in one PDF."`

	// MaxColspan is the value a table cell's colspan is clamped to.
	MaxColspan *int `json:"maxColspan" yaml:"maxColspan" default:"1000" env:"PIKO_PDF_MAX_COLSPAN" flag:"pdfMaxColspan" usage:"Value a PDF table cell's colspan is clamped to."`

	// MaxRowspan is the value a table cell's rowspan is clamped to.
	MaxRowspan *int `json:"maxRowspan" yaml:"maxRowspan" default:"65534" env:"PIKO_PDF_MAX_ROWSPAN" flag:"pdfMaxRowspan" usage:"Value a PDF table cell's rowspan is clamped to."`

	// MaxTableColumns is the maximum number of columns in one table.
	MaxTableColumns *int `json:"maxTableColumns" yaml:"maxTableColumns" default:"10000" env:"PIKO_PDF_MAX_TABLE_COLUMNS" flag:"pdfMaxTableColumns" usage:"Maximum columns in one PDF table."`

	// MaxGridTracks is the maximum number of tracks on one axis of a grid container.
	MaxGridTracks *int `json:"maxGridTracks" yaml:"maxGridTracks" default:"10000" env:"PIKO_PDF_MAX_GRID_TRACKS" flag:"pdfMaxGridTracks" usage:"Maximum tracks on one axis of a PDF grid container."`

	// MaxGridCells is the maximum total number of grid cells examined while placing grid
	// items in one layout.
	MaxGridCells *int `json:"maxGridCells" yaml:"maxGridCells" default:"10000000" env:"PIKO_PDF_MAX_GRID_CELLS" flag:"pdfMaxGridCells" usage:"Maximum grid cells examined while placing grid items in one PDF."`

	// MaxRepeatCount is the maximum integer count of a CSS repeat() track function.
	MaxRepeatCount *int `json:"maxRepeatCount" yaml:"maxRepeatCount" default:"10000" env:"PIKO_PDF_MAX_REPEAT_COUNT" flag:"pdfMaxRepeatCount" usage:"Maximum count of a CSS repeat() track function in a PDF."`

	// MaxPages is the maximum number of pages one document may produce.
	MaxPages *int `json:"maxPages" yaml:"maxPages" default:"10000" env:"PIKO_PDF_MAX_PAGES" flag:"pdfMaxPages" usage:"Maximum pages one PDF may produce."`

	// MaxImagePixels is the maximum pixel area (width times height) of one embedded image.
	MaxImagePixels *int `json:"maxImagePixels" yaml:"maxImagePixels" default:"100000000" env:"PIKO_PDF_MAX_IMAGE_PIXELS" flag:"pdfMaxImagePixels" usage:"Maximum pixel area of one image embedded in a PDF."`
}
